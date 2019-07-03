package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDeploymentRulesets(t *testing.T) {
	cfg := &Config{
		MonitorDefinitionsPath: "../../conf.yml",
	}
	cfg.reloadRulesets()

	annotations := make(map[string]string, 3)
	annotations["dd-manager/owner"] = "dd-manager"
	annotations["app"] = "foo"
	annotations["bar"] = "baz"

	objectType := "deployment"

	mSets := cfg.getMatchingRulesets(annotations, objectType)
	assert.Equal(t, 1, len(*mSets))
	mSet := (*mSets)[0]
	assert.Equal(t, objectType, mSet.ObjectType)
	assert.Equal(t, 1, len(mSet.Monitors))
	assert.Equal(t, "Deployment Replica Alert - {{ .ObjectMeta.Name }}", *mSet.Monitors[0].Name)

	monitors := cfg.GetMatchingMonitors(annotations, objectType)
	assert.Equal(t, mSet.Monitors, *monitors)
}

func TestGetNamespaceRulesets(t *testing.T) {
	cfg := &Config{
		MonitorDefinitionsPath: "../../conf.yml",
	}
	cfg.reloadRulesets()

	annotations := make(map[string]string, 3)
	annotations["dd-manager/owner"] = "dd-manager"
	annotations["app"] = "foo"
	annotations["bar"] = "baz"

	objectType := "namespace"

	mSets := cfg.getMatchingRulesets(annotations, objectType)
	assert.Equal(t, 1, len(*mSets))
	mSet := (*mSets)[0]
	assert.Equal(t, objectType, mSet.ObjectType)
	assert.Equal(t, 1, len(mSet.Monitors))
	assert.Equal(t, "Namespaced Deployment Replica Alert - {{ .ObjectMeta.Name }}", *mSet.Monitors[0].Name)

	monitors := cfg.GetMatchingMonitors(annotations, objectType)
	assert.Equal(t, mSet.Monitors, *monitors)
}
