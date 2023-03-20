package reroller

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const restartedAtAnnotation = "reroller.roob.re/restartedAt"

type Rollout interface {
	Name() string
	Containers() []corev1.Container
	ContainerStatuses() ([]corev1.ContainerStatus, error)
	Annotations() map[string]string
	Restart() error
}

func containerStatuses(client *kubernetes.Clientset, matchLabels map[string]string) ([]corev1.ContainerStatus, error) {
	pods, err := client.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set(matchLabels).AsSelector().String(),
	})
	if err != nil {
		return nil, err
	}

	var statuses []corev1.ContainerStatus
	for _, pod := range pods.Items {
		statuses = append(statuses, pod.Status.ContainerStatuses...)
	}

	return statuses, nil
}

func DeploymentRollout(client *kubernetes.Clientset, deployment *appsv1.Deployment) Rollout {
	return &deploymentRollout{
		depl:   deployment,
		client: client,
	}
}

type deploymentRollout struct {
	depl   *appsv1.Deployment
	client *kubernetes.Clientset
}

func (dr *deploymentRollout) Name() string {
	return "deployment " + dr.depl.Namespace + "/" + dr.depl.Name
}

func (dr *deploymentRollout) Annotations() map[string]string {
	return dr.depl.Annotations
}

func (dr *deploymentRollout) Containers() []corev1.Container {
	return dr.depl.Spec.Template.Spec.Containers
}

func (dr *deploymentRollout) ContainerStatuses() ([]corev1.ContainerStatus, error) {
	return containerStatuses(dr.client, dr.depl.Spec.Selector.MatchLabels)
}

func (dr *deploymentRollout) Restart() (err error) {
	if dr.depl.Spec.Template.ObjectMeta.Annotations == nil {
		dr.depl.Spec.Template.ObjectMeta.Annotations = map[string]string{}
	}
	dr.depl.Spec.Template.ObjectMeta.Annotations[restartedAtAnnotation] = time.Now().Format(time.RFC3339)
	dr.depl, err = dr.client.AppsV1().Deployments(dr.depl.Namespace).Update(context.TODO(), dr.depl, metav1.UpdateOptions{})

	return err
}

func DaemonSetRollout(client *kubernetes.Clientset, daemonSet *appsv1.DaemonSet) Rollout {
	return &daemonSetRollout{
		ds:     daemonSet,
		client: client,
	}
}

type daemonSetRollout struct {
	ds     *appsv1.DaemonSet
	client *kubernetes.Clientset
}

func (dsr *daemonSetRollout) Name() string {
	return "daemonSet " + dsr.ds.Namespace + "/" + dsr.ds.Name
}

func (dsr *daemonSetRollout) Annotations() map[string]string {
	return dsr.ds.Annotations
}

func (dsr *daemonSetRollout) Containers() []corev1.Container {
	return dsr.ds.Spec.Template.Spec.Containers
}

func (dsr *daemonSetRollout) ContainerStatuses() ([]corev1.ContainerStatus, error) {
	return containerStatuses(dsr.client, dsr.ds.Spec.Selector.MatchLabels)
}

func (dsr *daemonSetRollout) Restart() (err error) {
	if dsr.ds.Spec.Template.ObjectMeta.Annotations == nil {
		dsr.ds.Spec.Template.ObjectMeta.Annotations = map[string]string{}
	}
	dsr.ds.Spec.Template.ObjectMeta.Annotations[restartedAtAnnotation] = time.Now().Format(time.RFC3339)
	dsr.ds, err = dsr.client.AppsV1().DaemonSets(dsr.ds.Namespace).Update(context.TODO(), dsr.ds, metav1.UpdateOptions{})

	return err
}
