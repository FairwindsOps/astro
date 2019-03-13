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

type NotificationProfile struct {
  WarnAddresses string
  CriticalAddress string
}

type MatchRule struct {
  Field string
  Operand string
  Val string
}


type K8sMeta struct {
  ClusterName string
	Deployments []v1.Deployment
	Ingresses []v1beta1.Ingress
	Daemonsets []v1.DaemonSet
	Namespaces []corev1.Namespace
	Nodes []corev1.Node
}

type Config struct {
  Port int
  WebhookUri string
  DryRun bool
  Interval int
}


func New() *Config {
  return &Config {
    DatadogApiKey: getEnv("DD_API_KEY", ""),
    DatadogAppKey: getEnv("DD_APP_KEY", ""),
    DryRun: envAsBool("DRY_RUN", false),
    ClusterName: getEnv("CLUSTER_NAME", "")
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
