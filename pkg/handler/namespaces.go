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
	"fmt"
	"strings"

	"github.com/fairwindsops/dd-manager/pkg/config"
	"github.com/fairwindsops/dd-manager/pkg/datadog"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// OnNamespaceChanged is a handler that should be called when a namespace chanages.
func OnNamespaceChanged(namespace *corev1.Namespace, event config.Event) {
	cfg := config.GetInstance()
	dd := datadog.GetInstance()
	overrides := parseOverrides(namespace)

	switch strings.ToLower(event.EventType) {
	case "delete":
		if cfg.DryRun == false {
			log.Info("Deleting resource monitors.")
			dd.DeleteMonitors([]string{cfg.OwnerTag, fmt.Sprintf("dd-manager:object_type:%s", event.ResourceType), fmt.Sprintf("dd-manager:resource:%s", event.Key)})
		}
	case "create", "update":
		var record []string
		for _, monitor := range *cfg.GetMatchingMonitors(namespace.Annotations, event.ResourceType, overrides) {
			err := applyTemplate(namespace, &monitor, &event)
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
		// Update any bound monitors for this namespace
		updateBoundResources(namespace)
		if strings.ToLower(event.EventType) == "update" && !cfg.DryRun {
			// if there are any additional monitors, they should be removed.  This could happen if an object
			// was previously monitored and now no longer is.
			datadog.DeleteExtinctMonitors(record, []string{cfg.OwnerTag, fmt.Sprintf("dd-manager:object_type:%s", event.ResourceType), fmt.Sprintf("dd-manager:resource:%s", event.Key)})
		}
	default:
		log.Warnf("Update type %s is not valid, skipping.", event.EventType)
	}
}
