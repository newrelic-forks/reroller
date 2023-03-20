package main

import (
	"os"
	"path"
	"strings"
	"time"

	reroller "github.com/newrelic-forks/reroller/src"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const hourFmt = "15:04"

func main() {
	pflag.String("kubeconfig", path.Join(os.ExpandEnv("$HOME"), ".kube", "config"), "path to kubeconfig")
	pflag.String("namespaces", "", "namespaces to query (comma-separated)")
	pflag.Bool("unannotated", false, "process unannotated rollouts")
	pflag.Bool("dry-run", false, "do not actually reroll anything")
	pflag.String("log-level", "info", "log level (verbosity)")
	pflag.Duration("interval", 0, "run every [interval], empty to run one. time.ParseDuration format")
	pflag.Duration("cooldown", 48*time.Hour, "do not re-deploy more often than this. time.ParseDuration format")
	pflag.String("after", "00:00", "re-deploy only after this hour (e.g. 23:00)")
	pflag.String("before", "00:00", "re-deploy only before this hour (e.g. 03:00)")
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("REROLLER")
	viper.AutomaticEnv()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		log.Fatal(err)
	}

	lvl, _ := log.ParseLevel(viper.GetString("log-level"))
	log.SetLevel(lvl)

	after, err := time.Parse(hourFmt, viper.GetString("after"))
	if err != nil {
		log.Fatalf("Error parsing --after: %v", err)
	}
	before, err := time.Parse(hourFmt, viper.GetString("before"))
	if err != nil {
		log.Fatalf("Error parsing --before: %v", err)
	}

	c := reroller.Config{
		Namespaces:  strings.Split(viper.GetString("namespaces"), ","),
		Unannotated: viper.GetBool("unannotated"),
		DryRun:      viper.GetBool("dry-run"),
		Cooldown:    viper.GetDuration("cooldown"),
		Interval:    viper.GetDuration("interval"),
		Schedule: reroller.Schedule{
			After:  after,
			Before: before,
		},
	}

	rr, err := reroller.NewWithKubeconfig(viper.GetString("kubeconfig"), c)
	if err != nil {
		log.Fatal(err)
	}

	rr.Run()
}
