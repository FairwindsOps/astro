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

package handler

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	ddapi "github.com/zorkian/go-datadog-api"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fairwindsops/astro/pkg/config"
	"github.com/fairwindsops/astro/pkg/datadog"
	"github.com/fairwindsops/astro/pkg/kube"
	"github.com/fairwindsops/astro/pkg/metrics"
)

// OnDeploymentChanged is a handler that should be called when a deployment changes.
func OnDeploymentChanged(deployment *appsv1.Deployment, event config.Event) {
	cfg := config.GetInstance()
	dd := datadog.GetInstance()
	kubeClient := kube.GetInstance()
	overrides := parseOverrides(deployment)

	switch strings.ToLower(event.EventType) {
	case "delete":
		if !cfg.DryRun {
			log.Debug("Deleting resource monitors.")
			metrics.ChangeCounter.WithLabelValues("deployments", "delete").Inc()
			dd.DeleteMonitors([]string{cfg.OwnerTag, fmt.Sprintf("astro:object_type:%s", event.ResourceType), fmt.Sprintf("astro:resource:%s", event.Key)})
		}
	case "create", "update":
		var record []string
		var monitors []ddapi.Monitor

		ns, err := kubeClient.Client.CoreV1().Namespaces().Get(context.TODO(), event.Namespace, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Error getting namespace %s: %+v", event.Namespace, err)
			return
		}

		monitors = append(*cfg.GetMatchingMonitors(deployment.Annotations, event.ResourceType, overrides), *cfg.GetBoundMonitors(ns.Annotations, event.ResourceType, overrides)...)
		for _, monitor := range monitors {
			err = applyTemplate(deployment, &monitor, &event)
			if err != nil {
				metrics.TemplateErrorCounter.Inc()
				log.Errorf("Error applying template for monitor %s: %v", *monitor.Name, err)
				return
			}
			log.Debugf("Reconcile monitor %s", *monitor.Name)
			if !cfg.DryRun {
				_, err := dd.AddOrUpdate(&monitor)
				metrics.ChangeCounter.WithLabelValues("deployments", "create_update").Inc()
				record = append(record, *monitor.Name)
				if err != nil {
					metrics.ErrorCounter.Inc()
					log.Errorf("Error adding/updating monitor")
				}
			} else {
				log.Info("Running as DryRun, skipping DataDog update")
			}
		}

		if strings.ToLower(event.EventType) == "update" && !cfg.DryRun {
			// if there are any additional monitors, they should be removed.  This could happen if an object
			// was previously monitored and now no longer is.
			err = datadog.DeleteExtinctMonitors(record, []string{cfg.OwnerTag, fmt.Sprintf("astro:object_type:%s", event.ResourceType), fmt.Sprintf("astro:resource:%s", event.Key)})
			if err != nil {
				log.Errorf("Error deleting extinct monitors: %v", err)
				return
			}
		}
	default:
		log.Warnf("Update type %s is not valid, skipping.", event.EventType)
	}
}
