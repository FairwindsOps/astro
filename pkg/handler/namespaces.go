package handler

import (
  log "github.com/sirupsen/logrus"
  corev1 "k8s.io/api/core/v1"
  "github.com/reactiveops/dd-manager/conf"
  "github.com/reactiveops/dd-manager/pkg/util"
  "strings"	
  "fmt"
)



func OnNamespaceChanged(namespace *corev1.Namespace, event conf.Event) {
	cfg := conf.New()

  switch strings.ToLower(event.EventType) {
  case "delete":
    log.Info("Deleting resource monitors.")
    util.DeleteMonitors([]string{cfg.OwnerTag, fmt.Sprintf("dd-manager:object_type:%s", event.ResourceType), fmt.Sprintf("dd-manager:resource:%s", event.Key)})
  case "create", "update":
    for _, monitor := range *cfg.GetMatchingMonitors(namespace.Annotations, event.ResourceType) {
      log.Infof("Reconcile monitor %s", monitor.Name)
      applyTemplate(namespace, &monitor, &event)

      util.AddOrUpdate(&monitor)
    }
  default:
    log.Warnf("Update type %s is not valid, skipping.", event.EventType)
  }
}