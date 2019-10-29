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
	"fmt"
	"github.com/fairwindsops/astro/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fairwindsops/astro/pkg/controller"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var logLevels = map[string]log.Level{
	"panic": log.PanicLevel,
	"fatal": log.FatalLevel,
	"error": log.ErrorLevel,
	"warn":  log.WarnLevel,
	"info":  log.InfoLevel,
	"debug": log.DebugLevel,
	"trace": log.TraceLevel,
}

var rootCmd = &cobra.Command{
	Use:   "astro",
	Short: "Kubernetes datadog monitor manager",
	Long:  "A kubernetes agent that manages datadog monitors.",
	Run:   run,
}

var logLevel string
var metricsPort string

func init() {
	rootCmd.PersistentFlags().StringVarP(&logLevel, "level", "l", "info", "Log level setting. Default is INFO. Should be one of PANIC, FATAL, ERROR, WARN, INFO, DEBUG, or TRACE")
	rootCmd.PersistentFlags().StringVarP(&metricsPort, "metrics-port", "p", ":8080", "The address to serve prometheus metrics.")
}

func run(cmd *cobra.Command, args []string) {
	log.SetReportCaller(true)
	log.SetOutput(os.Stdout)
	log.SetLevel(logLevels[strings.ToLower(logLevel)])

	// create a channel for sending a stop to kube watcher threads
	stop := make(chan bool, 1)
	defer close(stop)
	go controller.NewController(stop)

	// Start metrics endpoint
	go func() {
		metrics.RegisterMetrics()
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(metricsPort, nil); err != nil {
			log.Error(err, "unable to serve the metrics endpoint")
			os.Exit(1)
		}
	}()

	// create a channel to respond to SIGTERMs
	signals := make(chan os.Signal, 1)
	defer close(signals)

	signal.Notify(signals, syscall.SIGTERM)
	signal.Notify(signals, syscall.SIGINT)
	s := <-signals
	stop <- true
	log.Info("Exiting, got signal: ", s)
}

// Execute is the main entry point into the command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
