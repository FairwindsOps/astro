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
		log.Infof("object has unknown type of %T", t)
	}

}


func OnDelete(obj interface{}) {
	log.Info("Handler got an OnDelete event.")
}


func OnCreate(obj interface{}) {
	log.Info("Handler got an OnCreate event.")
}