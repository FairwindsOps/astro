package handler


import (
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
  "github.com/reactiveops/dd-manager/conf"
)


func OnUpdatedDeployment(deployment *appsv1.Deployment) {
	log.Infof("I finally made it to my update handler.")

  cfg := conf.New()
  monitors := cfg.GetMatchingMonitors(deployment.Annotations, "deployment")
  for _, monitor := range *monitors {
    log.Infof("looping monitor %s", monitor.Name)
  }

}


func OnCreatedDeployment(deployment *appsv1.Deployment) {
	log.Infof("I finally made it to my creation handler.")

  cfg := conf.New()
  monitors := cfg.GetMatchingMonitors(deployment.Annotations, "deployment")
  for _, monitor := range *monitors {
    log.Infof("looping monitor %s", monitor.Name)

    //TODO - template out the monitor
  }


}

func OnDeletedDeployment(deployment *appsv1.Deployment) {
	log.Infof("I finally made it to my deletion handler.")
}
