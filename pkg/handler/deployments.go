package handler


import (
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
)


func OnUpdatedDeployment(deployment *appsv1.Deployment) {
	log.Infof("I finally made it to my update handler.")
}


func OnCreatedDeployment(deployment *appsv1.Deployment) {
	log.Infof("I finally made it to my creation handler.")
}

func OnDeletedDeployment(deployment *appsv1.Deployment) {
	log.Infof("I finally made it to my deletion handler.")
}