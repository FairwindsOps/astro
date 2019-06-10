package kube

import (
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
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
	config, err := rest.InClusterConfig()
	if err != nil {
		// not in cluster
		kubeconfig := getKubeConfigPath()
		localConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err)
		}
		clientset, err := kubernetes.NewForConfig(localConfig)
		if err != nil {
			panic(err)
		}
		return clientset
	}
	// in cluster
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return clientset
}

// getKubeConfigPath returns a valid kubeconfig path.
func getKubeConfigPath() string {
	var path string

	if os.Getenv("KUBECONFIG") != "" {
		path = os.Getenv("KUBECONFIG")
	} else if home, err := homedir.Dir(); err == nil {
		path = filepath.Join(home, ".kube", "config")
	} else {
		log.Fatal("kubeconfig not found.  Please ensure ~/.kube/config exists or KUBECONFIG is set.")
		os.Exit(1)
	}

	// kubeconfig doesn't exist
	if _, err := os.Stat(path); err != nil {
		log.Fatalf("%s doesn't exist - do you have a kubeconfig configured?\n", path)
		os.Exit(1)
	}
	return path
}
