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

package handler

import (
	"fmt"

	"github.com/fairwindsops/dd-manager/pkg/config"
	"github.com/fairwindsops/dd-manager/pkg/kube"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func updateBoundResources(namespace *corev1.Namespace) {
	kubeClient := kube.GetInstance()
	deploys, err := kubeClient.Client.AppsV1().Deployments(namespace.Name).List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error getting bound deployments for namespace %q.", namespace.Name)
		return
	}
	for _, dep := range deploys.Items {
		var evt config.Event
		evt.Key = fmt.Sprintf("%s/%s", namespace.Name, dep.Name)
		if err != nil {
			log.Errorf("Error handling bound deployment update event")
			return
		}
		evt.EventType = "update"
		evt.ResourceType = "deployment"
		evt.Namespace = namespace.Name
		OnDeploymentChanged(&dep, evt)
	}
}
