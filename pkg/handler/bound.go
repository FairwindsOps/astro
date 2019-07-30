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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func updateBoundResources(namespace *corev1.Namespace, kc *kube.ClientInstance) {
	deploys, err := kc.Client.AppsV1().Deployments(namespace.Name).List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error getting bound deployments for namespace %q.", namespace.Name)
		return
	}
	for _, dep := range deploys.Items {
		evt := setupBoundEvent(&dep)
		OnDeploymentChanged(&dep, evt)
	}
}

func setupBoundEvent(obj interface{}) config.Event {
	var evt config.Event
	switch object := obj.(type) {
	case *appsv1.Deployment:
		evt.Key = fmt.Sprintf("%s/%s", object.Namespace, object.Name)
		evt.Namespace = object.Namespace
		evt.ResourceType = "deployment"
	default:
		log.Warnf("Object has unknown type of %T", object)
	}
	evt.EventType = "update"
	return evt
}
