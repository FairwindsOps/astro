package handler


import (
  log "github.com/sirupsen/logrus"
  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
)


func OnUpdate(obj interface{}, eventType string) {
  log.Info("Handler got an OnUpdate event.")

  switch t := obj.(type) {
  case *appsv1.Deployment:
    OnDeploymentChanged(obj.(*appsv1.Deployment), eventType)
  case *corev1.Namespace:
    log.Info("Its a namespace.")
  default:
    log.Warnf("Object has unknown type of %T", t)
  }
}