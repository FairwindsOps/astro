package config

import (
  "os"
  "strconv"
  "fmt"
  log "github.com/sirupsen/logrus"
  "gopkg.in/yaml.v2"
  "io/ioutil"
  "sync"
  "encoding/json"
  "github.com/asaskevich/govalidator"
  "errors"
  "net/http"
  homedir "github.com/mitchellh/go-homedir"
  "k8s.io/client-go/rest"
  "k8s.io/client-go/kubernetes"
  "k8s.io/client-go/tools/clientcmd"
  "path/filepath"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ruleset struct {
  NotificationProfiles map[string]string `yaml:"notification_profiles"`
  MonitorSets         []MonitorSet  `yaml:"rulesets"`
}

type MonitorSet struct {
  ObjectType          string        `yaml:"type"`
  Annotations         []Annotation  `yaml:"match_annotations"`
  BoundObjects        []string      `yaml:"bound_objects,omitempty"`
  Monitors            []Monitor     `yaml:"monitors"`
}

type Annotation struct {
  Name                string        `yaml:"name"`
  Value               string        `yaml:"value"`
}


type Thresholds struct {
  Ok                  *json.Number  `yaml:"ok"`
  Critical            *json.Number  `yaml:"critical"`
  Warning             *json.Number  `yaml:"warning"`
  Unknown             *json.Number  `yaml:"unknown"`
  CriticalRecovery    *json.Number  `yaml:"critical_recovery"`
  WarningRecovery     *json.Number  `yaml:"warning_recovery"`
}

type Monitor struct {
  Name                string        `yaml:"name"`
  Type                string        `yaml:"type"`
  Query               string        `yaml:"query"`
  Message             string        `yaml:"message"`
  Tags                []string      `yaml:"tags"`
  NoDataTimeframe     int           `yaml:"no_data_timeframe"`
  NotifyAudit         bool          `yaml:"notify_audit"`
  NotifyNoData        bool          `yaml:"notify_no_data"`
  RenotifyInterval    int           `yaml:"renotify_interval"`
  NewHostDelay        int           `yaml:"new_host_delay"`
  EvaluationDelay     int           `yaml:"evaluation_delay"`
  Timeout             int           `yaml:"timeout"`
  EscalationMessage   string        `yaml:"escalation_message"`
  Thresholds          Thresholds    `yaml:"thresholds"`
  RequireFullWindow   bool          `yaml:"require_full_window"`
  Locked              bool          `yaml:"locked"`
}


type Event struct {
  Key          string
  EventType    string
  Namespace    string
  ResourceType string
}


type Config struct {
  DatadogApiKey string
  DatadogAppKey string
  DryRun bool
  ClusterName string
  OwnerTag string
  MonitorDefinitionsPath string
  Rulesets *ruleset
  KubeClient kubernetes.Interface
}


func (config *Config) GetMatchingMonitors(annotations map[string]string, objectType string) *[]Monitor {
  var validMonitors []Monitor

  for _, mSet := range *config.getMatchingRulesets(annotations, objectType) {
     validMonitors = append(validMonitors, mSet.Monitors...)
  }
  return &validMonitors
}


func (config *Config) getMatchingRulesets(annotations map[string]string, objectType string) *[]MonitorSet {
  var validMSets []MonitorSet

  for _, monitorSet := range config.Rulesets.MonitorSets {
    if monitorSet.ObjectType == objectType {
      var hasAllAnnotations = false

      for _, annotation := range monitorSet.Annotations {
        val, found := annotations[annotation.Name]
        if found && val == annotation.Value {
          hasAllAnnotations = true
        } else {
          hasAllAnnotations = false
          log.Infof("Annotation %s with value %s does not exist, exiting.", annotation.Name, annotation.Value)
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


func (config *Config) GetBoundMonitors(namespace string, objectType string) *[]Monitor {
  var linkedMonitors []Monitor

  // get info about the namespace the object resides in
  ns, err := config.KubeClient.CoreV1().Namespaces().Get(namespace,metav1.GetOptions{})

  if err != nil {
    log.Errorf("Error getting namespace %s: %+v", namespace, err)
  } else {
    mSets := config.getMatchingRulesets(ns.Annotations,"binding")
    for _, mSet := range *mSets {
      if contains(mSet.BoundObjects,objectType) {
        // object is linked to the ruleset
        linkedMonitors = append(linkedMonitors, mSet.Monitors...)
      }
    }
  }
  return &linkedMonitors
}


var instance *Config
var once sync.Once

func New() *Config {
  once.Do(func() {
    instance = &Config {
      DatadogApiKey: getEnv("DD_API_KEY", ""),
      DatadogAppKey: getEnv("DD_APP_KEY", ""),
      DryRun: envAsBool("DRY_RUN", false),
      ClusterName: getEnv("CLUSTER_NAME", ""),
      OwnerTag: getEnv("OWNER","dd-manager"),
      MonitorDefinitionsPath: getEnv("DEFINITIONS_PATH", "conf.yml"),
      Rulesets: loadMonitorDefinitions(getEnv("DEFINITIONS_PATH", "conf.yml")),
      KubeClient: getKubeClient(),
    }
  })
  return instance;
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
  }

  err = yaml.Unmarshal(yml, rSet)
  if err != nil {
    log.Fatalf("Error unmarshalling config file %s: %v", path, err)
  }
  return rSet
}


func loadFromPath(path string) ([]byte, error) {
  if fPath, _ := govalidator.IsFilePath(path); fPath {
    // path is local
    return ioutil.ReadFile(path)
  }

  if govalidator.IsURL(path) {
    // path is a url
    response, _ := http.Get(path)
    return ioutil.ReadAll(response.Body)
  }
  return nil, errors.New("Definitions is not a valid path or URL.")
}


func getEnv(key string, defaultVal string) string {
  if value, exists := os.LookupEnv(key); exists {
    return value
  }
  log.Info(fmt.Sprintf("Using default value %s for %s", defaultVal, key))
  return defaultVal
}


func envAsBool(key string, defaultVal bool) bool {
  val := getEnv(key, "")
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
    localConfig, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
    clientset, _ := kubernetes.NewForConfig(localConfig)
    return clientset
  } else {
    // in cluster
    clientset, _ := kubernetes.NewForConfig(config)
    return clientset
  }
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
