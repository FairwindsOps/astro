package handler


import (
  log "github.com/sirupsen/logrus"
  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
)


func OnUpdate(obj interface{}) {
	log.Info("Handler got an OnUpdate event.")

	switch t := obj.(type) {
	case *appsv1.Deployment:
		OnUpdatedDeployment(obj.(*appsv1.Deployment))
	case *corev1.Namespace:
		log.Info("Its a namespace.")
	default:
		log.Warnf("Object has unknown type of %T", t)
	}

}


func OnDelete(obj interface{}) {
	log.Info("Handler got an OnDelete event.")

  switch t := obj.(type) {
  case *appsv1.Deployment:
    OnDeletedDeployment(obj.(*appsv1.Deployment))
  case *corev1.Namespace:
    log.Info("Its a namespace")
  default:
    log.Warnf("Object has unknown type of %T", t)
  }
}


func OnCreate(obj interface{}) {
	log.Info("Handler got an OnCreate event.")

  switch t := obj.(type) {
  case *appsv1.Deployment:
    OnCreatedDeployment(obj.(*appsv1.Deployment))
  case *corev1.Namespace:
    log.Info("Its a namespace")
  default:
    log.Warnf("Object has unknown type of %T", t)
  }
}
