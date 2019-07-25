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
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/fairwindsops/dd-manager/pkg/config"
	log "github.com/sirupsen/logrus"
	ddapi "github.com/zorkian/go-datadog-api"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// OnUpdate is a handler that should be called when an object is updated.
// obj is the Kubernetes object that was updated.
// event is the Event metadata representing the update.
func OnUpdate(obj interface{}, event config.Event) {
	log.Infof("Handler got an OnUpdate event of type %s", event.EventType)

	if event.EventType == "delete" {
		onDelete(event)
		return
	}

	switch t := obj.(type) {
	case *appsv1.Deployment:
		OnDeploymentChanged(obj.(*appsv1.Deployment), event)
	case *corev1.Namespace:
		OnNamespaceChanged(obj.(*corev1.Namespace), event)
	default:
		log.Warnf("Object has unknown type of %T", t)
	}
}

func onDelete(event config.Event) {
	log.Info("OnDelete()")
	switch strings.ToLower(event.ResourceType) {
	case "namespace":
		OnNamespaceChanged(&corev1.Namespace{}, event)
	case "deployment":
		OnDeploymentChanged(&appsv1.Deployment{}, event)
	default:
		log.Warnf("object has unknown resource type %s", event.ResourceType)
	}
}

func applyTemplateToField(obj interface{}, tmplString string) (string, error) {
	var buf bytes.Buffer
	tpl, err := template.New("").Funcs(ancillaryVariables()).Parse(tmplString)
	if err != nil {
		return "", err
	}
	err = tpl.Execute(&buf, obj)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func applyTemplate(obj interface{}, monitor *ddapi.Monitor, event *config.Event) error {
	if monitor.Name != nil {
		name, err := applyTemplateToField(obj, *monitor.Name)
		if err != nil {
			return err
		}
		monitor.Name = &name
	}

	if monitor.Query != nil {
		query, err := applyTemplateToField(obj, *monitor.Query)
		if err != nil {
			return err
		}
		monitor.Query = &query
	}

	if monitor.Message != nil {
		message, err := applyTemplateToField(obj, *monitor.Message)
		if err != nil {
			return err
		}
		monitor.Message = &message
	}

	if monitor.Options != nil && monitor.Options.EscalationMessage != nil {
		message, err := applyTemplateToField(obj, *monitor.Options.EscalationMessage)
		if err != nil {
			return err
		}
		monitor.Options.EscalationMessage = &message
	}

	// apply identifying tags
	cfg := config.GetInstance()
	monitor.Tags = append(monitor.Tags,
		cfg.OwnerTag,
		fmt.Sprintf("dd-manager:object_type:%s", event.ResourceType),
		fmt.Sprintf("dd-manager:resource:%s", event.Key))
	return nil
}

func ancillaryVariables() map[string]interface{} {
	return map[string]interface{}{
		"ClusterVariables": func() map[string]string { return config.GetInstance().Rulesets.ClusterVariables },
	}
}

func parseOverrides(obj interface{}) map[string][]config.Override {
	overrides := make(map[string][]config.Override)
	switch obj.(type) {
	case *appsv1.Deployment:
		obj = obj.(*appsv1.Deployment)
		for key, value := range obj.(*appsv1.Deployment).Annotations {
			if isOverride(key) {
				overrideName, overrideKind := parseOverrideKey(key)
				thisOverride := config.Override{
					Field: overrideKind,
					Value: value,
				}
				overrides[overrideName] = append(overrides[overrideName], thisOverride)
			}
		}
	case *corev1.Namespace:
		for key, value := range obj.(*corev1.Namespace).Annotations {
			if isOverride(key) {
				overrideName, overrideKind := parseOverrideKey(key)
				thisOverride := config.Override{
					Field: overrideKind,
					Value: value,
				}
				overrides[overrideName] = append(overrides[overrideName], thisOverride)
			}
		}
	}
	return overrides
}

func parseOverrideKey(key string) (string, string) {
	split := strings.Split(key, "/")
	override := split[len(split)-1]
	splitOverride := strings.Split(override, ".")
	return splitOverride[len(splitOverride)-2], splitOverride[len(splitOverride)-1]
}

func isOverride(annotationKey string) bool {
	matched, err := regexp.Match(`^dd-manager\.fairwinds\.com/override\..*`, []byte(annotationKey))
	if err != nil {
		log.Errorf("Error parsing regexp of annotation key: %v", annotationKey)
	}
	if matched {
		log.Infof("Override found with annotation '%s'", annotationKey)
	}
	return matched
}
