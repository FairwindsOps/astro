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
  log "github.com/sirupsen/logrus"
  corev1 "k8s.io/api/core/v1"
  "github.com/reactiveops/dd-manager/pkg/config"
  "github.com/reactiveops/dd-manager/pkg/util"
  "strings"	
  "fmt"
)


// OnNamespaceChanged is a handler that should be called when a namespace chanages.
func OnNamespaceChanged(namespace *corev1.Namespace, event config.Event) {
	cfg := config.New()

  switch strings.ToLower(event.EventType) {
  case "delete":
    if cfg.DryRun == false {
      log.Info("Deleting resource monitors.")
      util.DeleteMonitors([]string{cfg.OwnerTag, fmt.Sprintf("dd-manager:object_type:%s", event.ResourceType), fmt.Sprintf("dd-manager:resource:%s", event.Key)})
    }
  case "create", "update":
    for _, monitor := range *cfg.GetMatchingMonitors(namespace.Annotations, event.ResourceType) {
      log.Infof("Reconcile monitor %s", monitor.Name)
      applyTemplate(namespace, &monitor, &event)
      if cfg.DryRun == false {
        util.AddOrUpdate(&monitor)
      }
    }
  default:
    log.Warnf("Update type %s is not valid, skipping.", event.EventType)
  }
}