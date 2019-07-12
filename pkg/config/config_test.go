package config

import (
	"bytes"
	"os"
	"testing"

	"github.com/reactiveops/dd-manager/pkg/kube"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var annotationCases = map[string]map[string]string{
	"pass": {
		"dd-manager/owner": "dd-manager",
	},
	"fail": {
		"test": "fail",
	},
}

var typeCases = map[string]map[string]string{
	"deployment": {
		"title": "Deployment Replica Alert - {{ .ObjectMeta.Name }}",
	},
	"namespace": {
		"title": "Namespaced Deployment Replica Alert - {{ .ObjectMeta.Name }}",
	},
}

func mockKube() *kube.ClientInstance {
	kubeClient := kube.ClientInstance{
		Client: fake.NewSimpleClientset(),
	}
	kube.SetInstance(kubeClient)
	return &kubeClient
}

func getConf(confPath string) *Config {
	config := &Config{
		MonitorDefinitionsPath: confPath,
	}
	config.reloadRulesets()
	return config
}

var cfg = getConf("./conf.yml")

// Because we use sync.Once.Do() to load the config it's difficult to test the logic when DD_API_KEY or DD_APP_KEY env vars are not present
func TestGetInstance(t *testing.T) {
	os.Setenv("DD_API_KEY", "dummy")
	os.Setenv("DD_APP_KEY", "dummy")
	os.Setenv("CLUSTER_NAME", "dummy")
	os.Setenv("OWNER", "dummy")
	os.Setenv("DEFINITIONS_PATH", "./conf.yml")
	os.Setenv("DRY_RUN", "false")

	cfg := GetInstance()
	assert.Equal(t, "dummy", (*cfg).DatadogAPIKey)
	assert.Equal(t, "dummy", (*cfg).DatadogAppKey)
	assert.Equal(t, "dummy", (*cfg).ClusterName)
	assert.Equal(t, "dummy", (*cfg).OwnerTag)
	assert.Equal(t, false, (*cfg).DryRun)
	assert.Equal(t, "./conf.yml", (*cfg).MonitorDefinitionsPath)
}

func TestGetRulesetsValid(t *testing.T) {
	for objectType, items := range typeCases {
		annotations := annotationCases["pass"]
		mSets := cfg.getMatchingRulesets(annotations, objectType)
		assert.Equal(t, 1, len(*mSets))
		mSet := (*mSets)[0]
		assert.Equal(t, objectType, mSet.ObjectType)
		assert.Equal(t, 1, len(mSet.Monitors))
		assert.Equal(t, items["title"], *mSet.Monitors[0].Name)

		monitors := cfg.GetMatchingMonitors(annotations, objectType)
		assert.Equal(t, mSet.Monitors, *monitors)
	}
}

func TestGetRulesetsInvalid(t *testing.T) {
	for objectType := range typeCases {
		annotations := annotationCases["fail"]
		mSets := cfg.getMatchingRulesets(annotations, objectType)
		assert.Equal(t, 0, len(*mSets))

	}
}

func TestGetBoundMonitorsValid(t *testing.T) {
	kubeClient := mockKube()
	annotations := make(map[string]string, 1)
	annotations["test"] = "yup"

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "owned-namespace",
			Annotations: annotations,
		},
	}
	kubeClient.Client.CoreV1().Namespaces().Create(ns)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Annotations: annotations,
		},
	}
	kubeClient.Client.AppsV1().Deployments("foo").Create(dep)

	mSets := cfg.GetBoundMonitors("owned-namespace", "deployment")
	assert.Equal(t, 1, len(*mSets))
	assert.Contains(t, (*mSets)[0].Tags, "dd-manager:bound_object")
}

func TestGetBoundMonitorsInvalid(t *testing.T) {
	kubeClient := mockKube()
	annotations := make(map[string]string, 1)
	annotations["test"] = "nope"
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "owned-namespace",
			Annotations: annotations,
		},
	}
	kubeClient.Client.CoreV1().Namespaces().Create(ns)

	mSets := cfg.GetBoundMonitors("owned-namespace", "deployment")
	assert.Equal(t, 0, len(*mSets))

	deleteOptions := metav1.NewDeleteOptions(10)
	kubeClient.Client.CoreV1().Namespaces().Delete("owned-namespace", deleteOptions)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stdout)
	}()
	cfg.GetBoundMonitors("owned-namespace", "deployment")
	assert.Contains(t, buf.String(), "Error getting namespace")
}

func TestGenEnvAsInt(t *testing.T) {
	os.Setenv("testing", "1")
	presentEnv := envAsInt("testing", 0)
	notPresentEnv := envAsInt("testing1", 0)
	assert.Equal(t, 1, presentEnv)
	assert.Equal(t, 0, notPresentEnv)
}

func TestGetEnvAsBool(t *testing.T) {
	os.Setenv("present", "true")
	presentEnv := envAsBool("present", false)
	notPresentEnv := envAsBool("notpresent", false)
	assert.Equal(t, true, presentEnv)
	assert.Equal(t, false, notPresentEnv)
}

func TestLoadFromPathInvalid(t *testing.T) {
	var invalidHTTP = "http://fake.fake/config.yml"
	var invalidLocal = "./fake.yml"

	data, err := loadFromPath(invalidHTTP)
	assert.Error(t, err)
	assert.Empty(t, data)

	data, err = loadFromPath(invalidLocal)
	assert.Error(t, err)
	assert.Empty(t, data)
}
