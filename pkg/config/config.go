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

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	ddapi "github.com/zorkian/go-datadog-api"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ruleset struct {
	ClusterVariables map[string]string `yaml:"cluster_variables,omitempty"`
	MonitorSets      []MonitorSet      `yaml:"rulesets,omitempty"`
}

// A MonitorSet represents a collection of Monitors that applies to an object.
type MonitorSet struct {
	ObjectType   string                   `yaml:"type"`                    // The type of object.  Example: deployment
	Annotations  []Annotation             `yaml:"match_annotations"`       // Annotations an object must possess to be considered applicable for the monitors.
	BoundObjects []string                 `yaml:"bound_objects,omitempty"` // A collection of ObjectTypes that are bound to the MonitorSet.
	Monitors     map[string]ddapi.Monitor `yaml:"monitors"`                // A collection of Monitors.
}

// An Annotation represent a kubernetes annotation.
type Annotation struct {
	Name  string `yaml:"name"`  // The annotation name.
	Value string `yaml:"value"` // The value of the annotation.
}

// An Event represents an update of a Kubernetes object and contains metadata about the update.
type Event struct {
	EventType    string             // The type of event - update, delete, or create
	Key          string             // A key identifying the object.  This is in the format <object-type>/<object-name>
	Namespace    string             // The namespace of the event's object
	OldMeta      *metav1.ObjectMeta // Metadata from old kubernetes object in update or delete events
	NewMeta      *metav1.ObjectMeta // Metadata from new kubernetes object in update or add events
	ResourceType string             // The type of resource that was updated.
}

// Config represents the application configuration.
type Config struct {
	DatadogAPIKey          string   // datadog api key for the datadog account
	DatadogAppKey          string   // datadog app key for the datadog account
	ClusterName            string   // A unique name for the cluster
	OwnerTag               string   // A unique tag to identify the owner of monitors
	MonitorDefinitionsPath []string // A url or local path for the configuration file
	Rulesets               *ruleset // The collection of rulesets to manage
	DryRun                 bool     // when set to true monitors will not be managed in datadog
}

// Override represents any datadog monitor fields annotations can be overridden
type Override struct {
	Field string
	Value string
}

// GetMatchingMonitors returns a collection of monitors that apply to the specified objectType and annotations.
func (config *Config) GetMatchingMonitors(annotations map[string]string, objectType string, overrides map[string][]Override) *[]ddapi.Monitor {
	var validMonitors []ddapi.Monitor

	for _, mSet := range *config.getMatchingRulesets(annotations, objectType, overrides) {
		for _, v := range mSet.Monitors {
			validMonitors = append(validMonitors, v)
		}
	}
	return &validMonitors
}

// GetStaticMonitors returns a collection of monitors from the config file that do not depend on resources in the kube cluster.
func (config *Config) GetStaticMonitors() *[]ddapi.Monitor {
	var validMonitors []ddapi.Monitor
	for _, monitorSet := range config.Rulesets.MonitorSets {
		if monitorSet.ObjectType == "static" {
			for _, v := range monitorSet.Monitors {
				validMonitors = append(validMonitors, v)
			}
		}
	}
	return &validMonitors
}

func (config *Config) getMatchingRulesets(annotations map[string]string, objectType string, overrides map[string][]Override) *[]MonitorSet {
	var validMSets []MonitorSet

	for monitorSetIdx, monitorSet := range config.Rulesets.MonitorSets {
		if monitorSet.ObjectType == objectType {
			var hasAllAnnotations = false

			for _, annotation := range monitorSet.Annotations {
				val, found := annotations[annotation.Name]
				if found && val == annotation.Value {
					hasAllAnnotations = true
				} else {
					hasAllAnnotations = false
					log.Debugf("Annotation %s with value %s does not exist, so monitor %d does not match", annotation.Name, annotation.Value, monitorSetIdx)
					break
				}
			}

			if hasAllAnnotations {
				for name := range monitorSet.Monitors {
					if _, exists := overrides[name]; exists {
						tmpMonitor := monitorSet.Monitors[name]
						tmpOverrides := overrides[name]
						for i, o := range overrides[name] {
							switch o.Field {
							case "name":
								tmpMonitor.Name = &tmpOverrides[i].Value
							case "type":
								tmpMonitor.Type = &tmpOverrides[i].Value
							case "query":
								tmpMonitor.Query = &tmpOverrides[i].Value
							case "message":
								tmpMonitor.Message = &tmpOverrides[i].Value
							case "threshold-critical":
								if tmpMonitor.Options == nil {
									tmpMonitor.Options = &ddapi.Options{}
								}
								if tmpMonitor.Options.Thresholds == nil {
									tmpMonitor.Options.Thresholds = &ddapi.ThresholdCount{}
								}
								threshold := json.Number(tmpOverrides[i].Value)
								tmpMonitor.Options.Thresholds.Critical = &threshold
							case "threshold-warning":
								if tmpMonitor.Options == nil {
									tmpMonitor.Options = &ddapi.Options{}
								}
								if tmpMonitor.Options.Thresholds == nil {
									tmpMonitor.Options.Thresholds = &ddapi.ThresholdCount{}
								}
								threshold := json.Number(tmpOverrides[i].Value)
								tmpMonitor.Options.Thresholds.Warning = &threshold
							default:
								log.Warnf("override provided does mot match any monitor fields. provided field: %s", o.Field)
							}
						}
						monitorSet.Monitors[name] = tmpMonitor
					} else {
						config.reloadRulesets()
					}
				}
				validMSets = append(validMSets, monitorSet)
			}
		}
	}
	return &validMSets
}

// GetBoundMonitors returns a collection of monitors that are indirectly bound to objectTypes in the namespace specified.
func (config *Config) GetBoundMonitors(nsAnnotations map[string]string, objectType string, overrides map[string][]Override) *[]ddapi.Monitor {
	var linkedMonitors []ddapi.Monitor
	mSets := config.getMatchingRulesets(nsAnnotations, "binding", overrides)

	for _, mSet := range *mSets {
		if contains(mSet.BoundObjects, objectType) {
			// object is linked to the ruleset
			mSet.AppendTag("astro:bound_object")
			for _, v := range mSet.Monitors {
				linkedMonitors = append(linkedMonitors, v)
			}
		}
	}
	return &linkedMonitors
}

// AppendTag appends a tag to every monitor in a MonitorSet
func (mSet *MonitorSet) AppendTag(tag string) {
	for key, monitor := range mSet.Monitors {
		tmp := mSet.Monitors[key]
		tmp.Tags = append(monitor.Tags, tag)
		mSet.Monitors[key] = tmp
	}
}

var instance *Config
var once sync.Once

// GetInstance is a singleton that returns the Configuration for the application.
func GetInstance() *Config {
	once.Do(func() {
		instance = &Config{
			DatadogAPIKey:          getEnv("DD_API_KEY", ""),
			DatadogAppKey:          getEnv("DD_APP_KEY", ""),
			ClusterName:            getEnv("CLUSTER_NAME", ""),
			OwnerTag:               getEnv("OWNER", "astro"),
			MonitorDefinitionsPath: envAsMap("DEFINITIONS_PATH", []string{"conf.yml"}, ";"),
			DryRun:                 envAsBool("DRY_RUN", false),
		}

		instance.reloadRulesets()

		if instance.DatadogAPIKey == "" || instance.DatadogAppKey == "" {
			log.Warnf("Datadog keys are not set, setting mode to dry run.")
			instance.DryRun = true
		}
		ticker := time.NewTicker(time.Minute)
		go func() {
			for range ticker.C {
				instance.reloadRulesets()
			}
		}()
	})
	return instance
}

func contains(slice []string, key string) bool {
	for _, element := range slice {
		if element == key {
			return true
		}
	}
	return false
}

func (config *Config) reloadRulesets() {
	rulesetCollection := &ruleset{
		ClusterVariables: make(map[string]string),
	}

	for _, cfg := range config.MonitorDefinitionsPath {
		log.Debugf("Loading rulesets from %s", cfg)
		rSet := &ruleset{}

		yml, err := loadFromPath(cfg)
		if err != nil {
			log.Errorf("Could not load config file %s: %v", cfg, err)
			return
		}

		err = yaml.Unmarshal(yml, rSet)
		if err != nil {
			log.Errorf("Error unmarshalling config file %s: %v", cfg, err)
			continue
		}

		if rSet.MonitorSets != nil {
			rulesetCollection.MonitorSets = append(rulesetCollection.MonitorSets, rSet.MonitorSets...)
		}

		if rSet.ClusterVariables != nil {
			for k, v := range rSet.ClusterVariables {
				rulesetCollection.ClusterVariables[k] = v
			}
		}
	}
	config.Rulesets = rulesetCollection
}

func loadFromPath(path string) ([]byte, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		// path is a url
		response, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		return ioutil.ReadAll(response.Body)
	}

	if _, err := os.Stat(path); err == nil {
		// path is local
		return ioutil.ReadFile(path)
	}

	return nil, errors.New("not a valid path or URL")
}

func getEnv(key string, defaultVal string) string {
	log.Debugf("Getting environment variable %s", key)
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Debugf(fmt.Sprintf("Using default value %s for %s", defaultVal, key))
	return defaultVal
}

func envAsMap(key string, defaultVal []string, delimiter string) []string {
	if value, exists := os.LookupEnv(key); exists {
		return strings.Split(value, delimiter)
	}
	return defaultVal
}

func envAsBool(key string, defaultVal bool) bool {
	val := getEnv(key, strconv.FormatBool(defaultVal))
	if val, err := strconv.ParseBool(val); err == nil {
		return val
	}
	log.Debugf("Using default value %t for %s", defaultVal, key)
	return defaultVal
}

func envAsInt(key string, defaultVal int) int {
	val := getEnv(key, "")
	if val, err := strconv.Atoi(val); err == nil {
		return val
	}
	log.Debugf("Using default value %d for %s", defaultVal, key)
	return defaultVal
}
