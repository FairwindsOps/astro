// Copyright 2019 FairwindsOps Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/fairwindsops/astro/pkg/controller"
	"github.com/fairwindsops/astro/pkg/kube"
	"github.com/fairwindsops/astro/pkg/metrics"
)

var (
	logLevels = map[string]log.Level{
		"panic": log.PanicLevel,
		"fatal": log.FatalLevel,
		"error": log.ErrorLevel,
		"warn":  log.WarnLevel,
		"info":  log.InfoLevel,
		"debug": log.DebugLevel,
		"trace": log.TraceLevel,
	}
	rootCmd = &cobra.Command{
		Use:   "astro",
		Short: "Kubernetes datadog monitor manager",
		Long:  "A kubernetes agent that manages datadog monitors.",
		Run:   leaderElection,
	}
	logLevel    string
	metricsPort string
	namespace   string
)

const (
	defaultLeaseDuration = 15 * time.Second
	defaultRenewDeadline = 10 * time.Second
	defaultRetryPeriod   = 2 * time.Second
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&logLevel, "level", "l", "info", "Log level setting. Default is INFO. Should be one of PANIC, FATAL, ERROR, WARN, INFO, DEBUG, or TRACE")
	rootCmd.PersistentFlags().StringVarP(&metricsPort, "metrics-port", "p", ":8080", "The address to serve prometheus metrics.")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "kube-system", "The namespace where astro is running")
}
func leaderElection(cmd *cobra.Command, args []string) {
	log.SetOutput(os.Stdout)
	log.SetLevel(logLevels[strings.ToLower(logLevel)])

	// Start metrics endpoint
	go func() {
		metrics.RegisterMetrics()
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(metricsPort, nil); err != nil {
			log.Error(err, "unable to serve the metrics endpoint")
			os.Exit(1)
		}
	}()

	id, err := os.Hostname()
	if err != nil {
		log.Fatalf("Unable to get hostname: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	kubeClient := kube.GetInstance()
	lock, err := resourcelock.New(
		resourcelock.LeasesResourceLock,
		namespace,
		"astro",
		kubeClient.Client.CoreV1(),
		kubeClient.Client.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity: id,
		},
	)
	if err != nil {
		log.Fatalf("Unable to create leader election lock: %v", err)
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: defaultLeaseDuration,
		RenewDeadline: defaultRenewDeadline,
		RetryPeriod:   defaultRetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				run(ctx, cancel)
			},
			OnStoppedLeading: func() {
				log.Infof("%s is no longer the leader", id)
			},
			OnNewLeader: func(identity string) {
				if id == identity {
					log.Debug("I'm now the leader")
					return
				}
				log.Infof("%s is now the leader", identity)
			},
		},
	})
}

func run(ctx context.Context, cancel context.CancelFunc) {
	// create a channel to respond to SIGTERMs
	signals := make(chan os.Signal, 1)
	defer close(signals)

	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		log.Info("Exiting, received termination signal")
		cancel()
	}()
	log.Info("Entering main run loop and starting watchers.")

	// TODO - run all controllers
	//controller.New(ctx)
	//k8sController := controller.KubeResourceController{}
	staticController := controller.StaticController{}
	//k8sController.Run(ctx)
	staticController.Run(ctx)
}

// Execute is the main entry point into the command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
