package conf

import (
  "os"
  "strconv"
  "fmt"
  log "github.com/sirupsen/logrus"
  "gopkg.in/yaml.v2"
  "io/ioutil"
  "sync"
)

type ruleset struct {
  NotificationProfiles map[string]string `yaml:"notification_profiles"`
  MonitorSets         []MonitorSet  `yaml:"rulesets"`
}

type MonitorSet struct {
  ObjectType          string        `yaml:"type"`
  Annotations         []Annotation  `yaml:"match_annotations"`
  Monitors            []Monitor     `yaml:"monitors"`
}

type Annotation struct {
  Name                string        `yaml:"name"`
  Value               string        `yaml:"value"`
}


type Thresholds struct {
  Ok                  int           `yaml:"ok"`
  Critical            int           `yaml:"critical"`
  Warning             int           `yaml:"warning"`
  Unknown             int           `yaml:"unknown"`
  CriticalRecovery    int           `yaml:"critical_recovery"`
  WarningRecovery     int           `yaml:"warning_recovery"`
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


type Config struct {
  DatadogApiKey string
  DatadogAppKey string
  DryRun bool
  ClusterName string
  OwnerTag string
  MonitorDefinitionsPath string
  Rulesets *ruleset
}


func (config *Config) GetMatchingMonitors(annotations map[string]string, objectType string) *[]Monitor {
  var validMonitors []Monitor

  for _, monitorSet := range config.Rulesets.MonitorSets {
    if monitorSet.ObjectType == objectType {
      var hasAllAnnotations = false

      for _, annotation := range monitorSet.Annotations {
          val, found := annotations[annotation.Name]
          if found == true && val == annotation.Value {
            log.Infof("Annotation %s with value %s exists.", annotation.Name, annotation.Value)
            hasAllAnnotations = true
          } else {
            log.Infof("Annotation %s with value %s does not exist, exiting.", annotation.Name, annotation.Value)
            break
          }
      }

      if hasAllAnnotations {
        // valid - add to the list of monitors
        validMonitors = append(validMonitors, monitorSet.Monitors...)
      }
    }
  }
  return &validMonitors
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
    }
  })
  return instance;
}


func loadMonitorDefinitions(path string) *ruleset {
  rSet := &ruleset{}
  yml, err := ioutil.ReadFile(path)
  if err != nil {
    log.Fatalf("Could not load config file %s: %v", path, err)
  }

  err = yaml.Unmarshal(yml, rSet)
  if err != nil {
    log.Fatalf("Error unmarshalling config file %s: %v", path, err)
  }
  return rSet
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
