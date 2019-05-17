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
    for _, monitor := range *cfg.GetMatchingMonitors(deployment.Annotations, event.ResourceType) {
      log.Infof("Reconcile monitor %s", monitor.Name)
      applyTemplate(deployment, &monitor, &event)
      util.AddOrUpdate(&monitor)
    }
  default:
    log.Warnf("Update type %s is not valid, skipping.", event.EventType)
  }
}