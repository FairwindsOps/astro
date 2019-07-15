// Copyright 2019 ReactiveOps
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
	"fmt"
	"strings"

	"github.com/reactiveops/dd-manager/pkg/config"
	"github.com/reactiveops/dd-manager/pkg/datadog"
	log "github.com/sirupsen/logrus"
	ddapi "github.com/zorkian/go-datadog-api"
	appsv1 "k8s.io/api/apps/v1"
)

// OnDeploymentChanged is a handler that should be called when a deployment changes.
func OnDeploymentChanged(deployment *appsv1.Deployment, event config.Event) {
	cfg := config.GetInstance()
	dd := datadog.GetInstance()

	switch strings.ToLower(event.EventType) {
	case "delete":
		if cfg.DryRun == false {
			log.Info("Deleting resource monitors.")
			dd.DeleteMonitors([]string{cfg.OwnerTag, fmt.Sprintf("dd-manager:object_type:%s", event.ResourceType), fmt.Sprintf("dd-manager:resource:%s", event.Key)})
		}
	case "create", "update":
		var record []string
		var monitors []ddapi.Monitor
		monitors = append(*cfg.GetMatchingMonitors(deployment.Annotations, event.ResourceType), *cfg.GetBoundMonitors(event.Namespace, event.ResourceType)...)
		for _, monitor := range monitors {
			err := applyTemplate(deployment, &monitor, &event)
			if err != nil {
				log.Errorf("Error applying template for monitor %s: %v", *monitor.Name, err)
				return
			}
			log.Infof("Reconcile monitor %s", *monitor.Name)
			if cfg.DryRun == false {
				_, err := dd.AddOrUpdate(&monitor)
				record = append(record, *monitor.Name)
				if err != nil {
					log.Errorf("Error adding/updating monitor")
				}
			} else {
				log.Info("Running as DryRun, skipping DataDog update")
			}
		}

		if strings.ToLower(event.EventType) == "update" && !cfg.DryRun {
			// if there are any additional monitors, they should be removed.  This could happen if an object
			// was previously monitored and now no longer is.
			datadog.DeleteExtinctMonitors(record, []string{cfg.OwnerTag, fmt.Sprintf("dd-manager:object_type:%s", event.ResourceType), fmt.Sprintf("dd-manager:resource:%s", event.Key)})
		}
	default:
		log.Warnf("Update type %s is not valid, skipping.", event.EventType)
	}
}
