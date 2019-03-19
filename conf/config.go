package conf

import (
	"k8s.io/api/apps/v1"
	"k8s.io/api/extensions/v1beta1"
	corev1 "k8s.io/api/core/v1"
  "os"
  "strconv"
  "fmt"
  log "github.com/sirupsen/logrus"
)


type MonitorSet struct {
  Object string `yaml:"object"`
  Annotations []Annotation `yaml:"match_annotations"`
  Monitors: []Monitor `yaml:"monitors"`
}

type Annotation struct {
  Name string `yaml:"name"`
  Value string `yaml:"value"`
}

type Thresholds struct {
  Ok int `yaml:"ok"`
  Critical int `yaml:"critical"`
  Warning int `yaml:"warning"`
  Unknown int `yaml:"unknown"`
  CriticalRecovery int `yaml:"critical_recovery"`
  WarningRecovery int `yaml:"warning_recovery"`
}

type Monitor struct {
  Name string `yaml:"name"`
  Type string `yaml:"type"`
  Query string `yaml:"query"`
  Message string `yaml:"message"`
  Tags []string `yaml:"tags"`
  NoDataTimeframe int `yaml:"no_data_timeframe"`
  NotifyAudit bool `yaml:"notify_audit"`
  NotifyNoData bool `yaml:"notify_no_data"`
  RenotifyInterval int `yaml:"renotify_interval"`
  NewHostDelay int `yaml:"new_host_delay"`
  EvaluationDelay int `yaml:"evaluation_delay"`
  Timeout int `yaml:"timeout"`
  EscalationMessage string `yaml:"escalation_message"`
  Thresholds Thresholds `yaml:"thresholds"`
  RequireFullWindow bool `yaml:"require_full_window"`
  Locked bool `yaml:"locked"`
}



type Config struct {
  DatadogApiKey string
  DatadogAppKey string
  DryRun bool
  ClusterName string
  MonitorDefinitions string
}


func New() *Config {
  return &Config {
    DatadogApiKey: getEnv("DD_API_KEY", ""),
    DatadogAppKey: getEnv("DD_APP_KEY", ""),
    DryRun: envAsBool("DRY_RUN", false),
    ClusterName: getEnv("CLUSTER_NAME", ""),
    MonitorDefinitions: getEnv("DEFINITIONS_PATH", "conf.yml"),
  }
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
