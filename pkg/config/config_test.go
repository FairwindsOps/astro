package config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	ddapi "github.com/zorkian/go-datadog-api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var annotationCases = map[string]map[string]string{
	"pass": {
		"astro/owner": "astro",
	},
	"fail": {
		"test": "fail",
	},
}

var typeCases = map[string]map[string]string{
	"deployment": {
		"title": "Deployment Replica Alert - {{ .ObjectMeta.Name }}",
		"name":  "dep-replica-alert",
	},
	"namespace": {
		"title": "Namespaced Deployment Replica Alert - {{ .ObjectMeta.Name }}",
		"name":  "namespaced-replica-alert",
	},
}

func getConf(confPath []string) *Config {
	config := &Config{
		MonitorDefinitionsPath: confPath,
	}
	config.reloadRulesets()
	return config
}

var cfg = getConf([]string{"./test_conf.yml"})

// Because we use sync.Once.Do() to load the config it's difficult to test the logic when DD_API_KEY or DD_APP_KEY env vars are not present
func TestGetInstance(t *testing.T) {
	os.Setenv("DD_API_KEY", "dummy")
	os.Setenv("DD_APP_KEY", "dummy")
	os.Setenv("CLUSTER_NAME", "dummy")
	os.Setenv("OWNER", "dummy")
	os.Setenv("DEFINITIONS_PATH", "./test_conf.yml;./test_conf_variables.yml")
	os.Setenv("DRY_RUN", "false")

	cfg := GetInstance()
	assert.Equal(t, "dummy", (*cfg).DatadogAPIKey)
	assert.Equal(t, "dummy", (*cfg).DatadogAppKey)
	assert.Equal(t, "dummy", (*cfg).ClusterName)
	assert.Equal(t, "dummy", (*cfg).OwnerTag)
	assert.Equal(t, false, (*cfg).DryRun)
	assert.Equal(t, []string{"./test_conf.yml", "./test_conf_variables.yml"}, (*cfg).MonitorDefinitionsPath)
}

func TestGetClusterVariables(t *testing.T) {
	cfg := GetInstance()
	assert.Contains(t, cfg.Rulesets.ClusterVariables, "TEST_FOO")
	assert.Equal(t, cfg.Rulesets.ClusterVariables["TEST_FOO"], "BAR")
}

func TestGetRulesetsValid(t *testing.T) {
	annotations := annotationCases["pass"]
	overrides := map[string][]Override{
		"dep-replica-alert": {
			{
				Field: "threshold-critical",
				Value: "10.0",
			},
			{
				Field: "threshold-warning",
				Value: "5",
			},
		},
		"namespaced-replica-alert": {
			{
				Field: "threshold-critical",
				Value: "500",
			},
			{
				Field: "threshold-warning",
				Value: "100",
			},
		},
	}
	thresholds := map[string]map[string]json.Number{
		"dep-replica-alert": {
			"critical": json.Number("10.0"),
			"warning":  json.Number("5"),
		},
		"namespaced-replica-alert": {
			"critical": json.Number("500"),
			"warning":  json.Number("100"),
		},
	}

	for objectType, items := range typeCases {
		name := items["name"]
		title := items["title"]
		mSets := cfg.getMatchingRulesets(annotations, objectType, overrides)
		assert.Equal(t, 1, len(*mSets))
		mSet := (*mSets)[0]
		assert.Equal(t, objectType, mSet.ObjectType)
		assert.Equal(t, 1, len(mSet.Monitors))
		assert.Equal(t, title, *mSet.Monitors[name].Name)
		assert.Equal(t, thresholds[name]["critical"], *mSet.Monitors[name].Options.Thresholds.Critical)
		assert.Equal(t, thresholds[name]["warning"], *mSet.Monitors[name].Options.Thresholds.Warning)

		monitors := cfg.GetMatchingMonitors(annotations, objectType, overrides)
		var expected []ddapi.Monitor
		for _, value := range mSet.Monitors {
			expected = append(expected, value)
		}
		assert.Equal(t, expected, *monitors)
	}
}

func TestGetRulesetsInvalid(t *testing.T) {
	for objectType := range typeCases {
		annotations := annotationCases["fail"]
		overrides := make(map[string][]Override)
		mSets := cfg.getMatchingRulesets(annotations, objectType, overrides)
		assert.Equal(t, 0, len(*mSets))
	}
}

func TestGetBoundMonitorsValid(t *testing.T) {
	annotations := make(map[string]string, 1)
	annotations["test"] = "yup"

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "owned-namespace",
			Annotations: annotations,
		},
	}

	overrides := make(map[string][]Override)
	mSets := cfg.GetBoundMonitors(ns.Annotations, "deployment", overrides)
	assert.Equal(t, 1, len(*mSets))
	assert.Contains(t, (*mSets)[0].Tags, "astro:bound_object")
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
	var invalidHTTP = "https://fake.fake/config.yml"
	var invalidLocal = "./fake.yml"

	data, err := loadFromPath(invalidHTTP)
	assert.Error(t, err)
	assert.Empty(t, data)

	data, err = loadFromPath(invalidLocal)
	assert.Error(t, err)
	assert.Empty(t, data)
}
