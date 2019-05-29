// Copyright 2019 ReactiveOps
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
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type ruleset struct {
	NotificationProfiles map[string]string `yaml:"notification_profiles"`
	MonitorSets          []MonitorSet      `yaml:"rulesets"`
}

// A MonitorSet represents a collection of Monitors that applies to an object.
type MonitorSet struct {
	ObjectType   string       `yaml:"type"`                    // The type of object.  Example: deployment
	Annotations  []Annotation `yaml:"match_annotations"`       // Annotations an object must possess to be considered applicable for the monitors.
	BoundObjects []string     `yaml:"bound_objects,omitempty"` // A collection of ObjectTypes that are bound to the MonitorSet.
	Monitors     []Monitor    `yaml:"monitors"`                // A collection of Monitors.
}

// An Annotation represent a kubernetes annotation.
type Annotation struct {
	Name  string `yaml:"name"`  // The annotation name.
	Value string `yaml:"value"` // The value of the annotation.
}

// Thresholds represent the alerting thresholds for a monitor.
type Thresholds struct {
	Ok               *json.Number `yaml:"ok"`                // The threshold to return to OK status.
	Critical         *json.Number `yaml:"critical"`          // The threshold to trigger a Critical state.
	Warning          *json.Number `yaml:"warning"`           // The threshold to trigger a Warning state.
	Unknown          *json.Number `yaml:"unknown"`           // The threshold to trigger an Unknown state.
	CriticalRecovery *json.Number `yaml:"critical_recovery"` // The threshold to clear a Critical state.
	WarningRecovery  *json.Number `yaml:"warning_recovery"`  // The threshold to clear a Warning state.
}

// A Monitor represents a datadog Monitor.
type Monitor struct {
	Name              string     `yaml:"name"`                // The name of the monitor.
	Type              string     `yaml:"type"`                // The type of montior.  Must be a valid datadog monitor type.
	Query             string     `yaml:"query"`               // The monitor query.
	Message           string     `yaml:"message"`             // A message included with monitor notifications.
	Tags              []string   `yaml:"tags"`                // A collection of tags to add to your monitor.
	NoDataTimeframe   int        `yaml:"no_data_timeframe"`   // Number of minutes before a monitor will notify if data stops reporting.
	NotifyAudit       bool       `yaml:"notify_audit"`        // boolean that indicates whether tagged users are notified if the monitor changes.
	NotifyNoData      bool       `yaml:"notify_no_data"`      // boolean that indicates if the monitor notifies if data stops reporting.
	RenotifyInterval  int        `yaml:"renotify_interval"`   // Number of minutes after the last notification a monitor will re-notify.
	NewHostDelay      int        `yaml:"new_host_delay"`      // Number of seconds to wait for a new host before evaluating the monitor status.
	EvaluationDelay   int        `yaml:"evaluation_delay"`    // Number of seconds to delay evaluation.
	Timeout           int        `yaml:"timeout"`             // Number of minutes before the monitor will automatically resolve if it's not reporting data.
	EscalationMessage string     `yaml:"escalation_message"`  // Message to include with re-notifications.
	Thresholds        Thresholds `yaml:"thresholds"`          // Map of thresholds for the alert.
	RequireFullWindow bool       `yaml:"require_full_window"` // boolean indicating if a monitor needs a full window of data to be evaluated.
	Locked            bool       `yaml:"locked"`              // boolean indicating if changes are only allowed form the creator or admins.
}

// An Event represents an update of a Kubernetes object and contains metadata about the update.
type Event struct {
	Key          string // A key identifying the object.  This is in the format <object-type>/<object-name>
	EventType    string // The type of event - update, delete, or create
	Namespace    string // The namespace of the event's object
	ResourceType string // The type of resource that was updated.
}

// Config represents the application configuration.
type Config struct {
	DatadogAPIKey          string               // datadog api key for the datadog account.
	DatadogAppKey          string               // datadog app key for the datadog account.
	ClusterName            string               // A unique name for the cluster.
	OwnerTag               string               // A unique tag to identify the owner of monitors.
	MonitorDefinitionsPath string               // A url or local path for the configuration file.
	Rulesets               *ruleset             // The collection of rulesets to manage.
	KubeClient             kubernetes.Interface // A kubernetes client to interact with the cluster.
	DryRun                 bool                 // when set to true monitors will not be managed in datadog.
}

// GetMatchingMonitors returns a collection of monitors that apply to the specified objectType and annotations.
func (config *Config) GetMatchingMonitors(annotations map[string]string, objectType string) *[]Monitor {
	var validMonitors []Monitor

	for _, mSet := range *config.getMatchingRulesets(annotations, objectType) {
		validMonitors = append(validMonitors, mSet.Monitors...)
	}
	return &validMonitors
}

func (config *Config) getMatchingRulesets(annotations map[string]string, objectType string) *[]MonitorSet {
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
					log.Infof("Annotation %s with value %s does not exist, so monitor %d does not match", annotation.Name, annotation.Value, monitorSetIdx)
					break
				}
			}

			if hasAllAnnotations {
				validMSets = append(validMSets, monitorSet)
			}
		}
	}
	return &validMSets
}

// GetBoundMonitors returns a collection of monitors that are indirectly bound to objectTypes in the namespace specified.
func (config *Config) GetBoundMonitors(namespace string, objectType string) *[]Monitor {
	var linkedMonitors []Monitor

	// get info about the namespace the object resides in
	ns, err := config.KubeClient.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})

	if err != nil {
		log.Errorf("Error getting namespace %s: %+v", namespace, err)
	} else {
		mSets := config.getMatchingRulesets(ns.Annotations, "binding")
		for _, mSet := range *mSets {
			if contains(mSet.BoundObjects, objectType) {
				// object is linked to the ruleset
				linkedMonitors = append(linkedMonitors, mSet.Monitors...)
			}
		}
	}
	return &linkedMonitors
}

var instance *Config
var once sync.Once

// New is a singleton that returns the Configuration for the application.
func New() *Config {
	once.Do(func() {
		instance = &Config{
			DatadogAPIKey:          getEnv("DD_API_KEY", ""),
			DatadogAppKey:          getEnv("DD_APP_KEY", ""),
			ClusterName:            getEnv("CLUSTER_NAME", ""),
			OwnerTag:               getEnv("OWNER", "dd-manager"),
			MonitorDefinitionsPath: getEnv("DEFINITIONS_PATH", "conf.yml"),
			KubeClient:             getKubeClient(),
			DryRun:                 envAsBool("DRY_RUN", false),
		}

		instance.Rulesets = loadMonitorDefinitions(instance.MonitorDefinitionsPath)

		if instance.DatadogAPIKey == "" || instance.DatadogAppKey == "" {
			log.Warnf("Datadog keys are not set, setting mode to dry run.")
			instance.DryRun = true
		}
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

func loadMonitorDefinitions(path string) *ruleset {
	rSet := &ruleset{}
	//yml, err := ioutil.ReadFile(path)
	yml, err := loadFromPath(path)
	if err != nil {
		log.Fatalf("Could not load config file %s: %v", path, err)
		return rSet
	}

	err = yaml.Unmarshal(yml, rSet)
	if err != nil {
		log.Fatalf("Error unmarshalling config file %s: %v", path, err)
	}
	return rSet
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
	log.Infof("Getting environment variable %s", key)
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Info(fmt.Sprintf("Using default value %s for %s", defaultVal, key))
	return defaultVal
}

func envAsBool(key string, defaultVal bool) bool {
	val := getEnv(key, strconv.FormatBool(defaultVal))
	if val, err := strconv.ParseBool(val); err == nil {
		return val
	}
	log.Info(fmt.Sprintf("Using default value %t for %s", defaultVal, key))
	return defaultVal
}

func envAsInt(key string, defaultVal int) int {
	val := getEnv(key, "")
	if val, err := strconv.Atoi(val); err == nil {
		return val
	}
	log.Info(fmt.Sprintf("Using default value %d for %s", defaultVal, key))
	return defaultVal
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
