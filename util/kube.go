package util

import (
	"os"
	"fmt"
	log "github.com/sirupsen/logrus"
	homedir "github.com/mitchellh/go-homedir"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"path/filepath"
)




func GetKubeClient() kubernetes.Interface {
	config, err := rest.InClusterConfig()
	if err != nil {
		// not in cluster
		kubeconfig := getKubeConfig()
		localConfig, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
		clientset, _ := kubernetes.NewForConfig(localConfig)
		return clientset
	} else {
		// in cluster
		clientset, _ := kubernetes.NewForConfig(config)
		return clientset
	}
}


// getKubeConfig returns a valid kubeconfig path.
func getKubeConfig() string {
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
		log.Fatal(fmt.Printf("%s doesn't exist - do you have a kubeconfig configured?\n", path))
		os.Exit(1)
	}
	return path
}
