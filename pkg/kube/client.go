// Copyright 2019 FairwindsOps Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kube

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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

// SetInstance sets the Kubernetes interface to use - this is for testing only
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
