package handler


import (
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
  "github.com/reactiveops/dd-manager/pkg/config"
  "github.com/reactiveops/dd-manager/pkg/util"
  "strings"
  "fmt"
)


func OnDeploymentChanged(deployment *appsv1.Deployment, event config.Event) {
  cfg := config.New()

  switch strings.ToLower(event.EventType) {
  case "delete":
    log.Info("Deleting resource monitors.")
    util.DeleteMonitors([]string{cfg.OwnerTag, fmt.Sprintf("dd-manager:object_type:%s", event.ResourceType), fmt.Sprintf("dd-manager:resource:%s", event.Key)})
  case "create", "update":
    var monitors []config.Monitor
    monitors = append(*cfg.GetMatchingMonitors(deployment.Annotations, event.ResourceType), *cfg.GetBoundMonitors(event.Namespace, event.ResourceType)...)
    for _, monitor := range monitors {
      log.Infof("Reconcile monitor %s", monitor.Name)
      applyTemplate(deployment, &monitor, &event)
      util.AddOrUpdate(&monitor)
    }
  default:
    log.Warnf("Update type %s is not valid, skipping.", event.EventType)
  }
}