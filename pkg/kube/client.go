package kube

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sync"
)

// ClientInstance is a wrapper around the kubernetes interface for testing purposes
type ClientInstance struct {
	Client kubernetes.Interface
}

var kubeClient *ClientInstance
var once sync.Once

// GetInstance returns a Kubernetes interface based on the current configuration
func GetInstance() *ClientInstance {
	once.Do(func() {
		if kubeClient == nil {
			kubeClient = &ClientInstance{
				Client: getKubeClient(),
			}
		}
	})
	return kubeClient
}

// SetInstance sets the Kuberentes interface to use - this is for testing only
func SetInstance(kc ClientInstance) {
	kubeClient = &kc
}

func getKubeClient() kubernetes.Interface {
	kubeConf, err := config.GetConfig()
	if err != nil {
		log.Fatalf("Error getting kubeconfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(kubeConf)
	if err != nil {
		log.Fatalf("Error creating kubernetes client: %v", err)
	}
	return clientset
}
