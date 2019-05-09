package handler


import (
  log "github.com/sirupsen/logrus"
  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  "github.com/reactiveops/dd-manager/conf"
)


func Test(event conf.Event) {
  log.Infof(event.EventType)
}

func OnUpdate(obj interface{}, event conf.Event) {
  log.Infof("Handler got an OnUpdate event of type %s", event.EventType)

  switch t := obj.(type) {
  case *appsv1.Deployment:
    OnDeploymentChanged(obj.(*appsv1.Deployment), event)
  case *corev1.Namespace:
    OnNamespaceChanged(obj.(*corev1.Namespace), event)
  default:
    log.Warnf("Object has unknown type of %T", t)
  }
}