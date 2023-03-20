package reroller

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/newrelic-forks/reroller/src/registry"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const rerollerAnnotation = "reroller.roob.re/reroll"

type Reroller struct {
	K8S      *kubernetes.Clientset
	Registry func(image string) ([]string, error)
	Config
}

type Config struct {
	Namespaces  []string
	Unannotated bool
	DryRun      bool
	Cooldown    time.Duration
	Interval    time.Duration
	Schedule    Schedule
}

func restConfig(kubeconfig string) (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, err
	}

	if kubeconfig == "" {
		return nil, errors.New("could not get config from env and kubeconfig path is empty: " + err.Error())
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err == nil {
		return config, err
	}

	return nil, err
}

func New(c Config) (*Reroller, error) {
	return NewWithKubeconfig("", c)
}

func NewWithKubeconfig(kubeconfig string, c Config) (*Reroller, error) {
	rr := &Reroller{
		Registry: registry.ImageDigests,
		Config:   c,
	}

	// creates the in-cluster config
	config, err := restConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	rr.K8S = clientset
	return rr, nil
}

func (rr *Reroller) Run() {
	if rr.Interval == 0 {
		rr.RunOnce()
		return
	}

	for {
		if rr.Schedule.ShouldRunNow() {
			rr.RunOnce()
		} else {
			log.Debugf("Skipping check since %v is not between the schedule", time.Now())
		}

		time.Sleep(rr.Interval)
	}
}

func (rr *Reroller) RunOnce() {
	var rollouts []Rollout
	rollouts = append(rollouts, rr.deploymentRollouts()...)
	rollouts = append(rollouts, rr.daemonSetRollouts()...)

	log.Infof("found %d rollouts to check in ns [%s]", len(rollouts), strings.Join(rr.Namespaces, ", "))

	for _, rollout := range rollouts {
		log.Debugf("considering %s", rollout.Name())
		if !rr.shouldReroll(rollout.Annotations()) {
			log.Debugf("%s is not annotated, skipping", rollout.Name())
			continue
		}

		if !hasAlwaysPullPolicy(rollout.Containers()) {
			log.Debugf("%s does not have pullPolicy == Always, skipping", rollout.Name())
			continue
		}

		statuses, err := rollout.ContainerStatuses()
		if err != nil {
			log.Errorf("error getting container statuses: %v", err)
			continue
		}

		log.Infof("checking updates for %d containers in %s", len(statuses), rollout.Name())
		if rr.hasUpdate(statuses) {
			log.Infof("Restarting " + rollout.Name())

			if rr.DryRun {
				log.Warnf("dry-run: not actually restarting")
				continue
			}

			err = rollout.Restart()
			if err != nil {
				log.Errorf("error restarting %s: %v", rollout.Name(), err)
			}
		}
	}
}

func (rr *Reroller) deploymentRollouts() (rollouts []Rollout) {
	log.Debugf("Fetching deployments in ns %s", rr.Namespaces)
	for _, ns := range rr.Namespaces {
		deployments, err := rr.K8S.AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Errorf("error getting deployments in %s: %v", ns, err.Error())
			return
		}

		for _, depl := range deployments.Items {
			if depl.Status.AvailableReplicas > 0 {
				rollouts = append(rollouts, DeploymentRollout(rr.K8S, depl.DeepCopy()))
			}
		}
	}

	return
}

func (rr *Reroller) daemonSetRollouts() (rollouts []Rollout) {
	log.Debugf("Fetching daemonSets in ns %s", rr.Namespaces)
	for _, ns := range rr.Namespaces {
		daemonSets, err := rr.K8S.AppsV1().DaemonSets(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Errorf("error getting daemonSets in %s: %v", ns, err.Error())
			return
		}

		for _, ds := range daemonSets.Items {
			if ds.Status.NumberAvailable > 0 {
				rollouts = append(rollouts, DaemonSetRollout(rr.K8S, ds.DeepCopy()))
			}
		}
	}

	return
}

func (rr *Reroller) shouldReroll(annotations map[string]string) bool {
	rawVal, found := annotations[rerollerAnnotation]
	var val bool

	if !found {
		val = rr.Unannotated
	} else {
		val, _ = strconv.ParseBool(rawVal)
	}

	// If annotation says don't restart, return that
	if !val {
		return false
	}

	// Check when was last restart
	lastRestartedStr, redeployed := annotations[restartedAtAnnotation]
	if !redeployed {
		// If never redeployed, green light
		log.Tracef("rollout was never redeployed, ok to continue")
		return true
	}
	lastRestarted, err := time.Parse(time.RFC3339, lastRestartedStr)
	if err != nil {
		log.Warn("error parsing last restart time, ignoring")
		return true
	}

	if time.Since(lastRestarted) < rr.Cooldown {
		// Don't redeoploy if last time is not above threshold
		log.Warnf("last redeploy was %v ago (<%v), skipping", lastRestarted, rr.Cooldown)
		return false
	}

	return true
}

func (rr *Reroller) hasUpdate(statuses []v1.ContainerStatus) bool {
	// Iterate over all containers in all the pods of the rollout
	for _, status := range statuses {
		imagePieces := strings.Split(status.ImageID, "@")
		if len(imagePieces) < 2 {
			log.Errorf("Malformed imageID '%s', skipping upgrade check", status.ImageID)
			continue
		}
		digest := imagePieces[1]

		upstreamDigests, err := rr.Registry(status.Image)
		if err != nil {
			log.Errorf("Could not fetch latest digest for %s: %v", status.Image, err)
			continue
		}

		found := false
		for _, ud := range upstreamDigests {
			if digest == ud {
				found = true
				break
			}
		}

		if !found {
			log.Tracef("%s not found un upstream manifest list %v", digest, upstreamDigests)
			return true
		}

		log.Debugf("No new digest found for %s", status.Image)
	}

	return false
}

func hasAlwaysPullPolicy(containers []v1.Container) bool {
	for _, ct := range containers {
		if ct.ImagePullPolicy == v1.PullAlways {
			return true
		}
	}

	return false
}
