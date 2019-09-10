package handler

import (
	"testing"

	"github.com/fairwindsops/astro/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/zorkian/go-datadog-api"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplyTemplate(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	nameTemplate := "Name {{ .ObjectMeta.Name }}"
	queryTemplate := "Query {{ .ObjectMeta.Name }}"
	messageTemplate := "Message {{ .ObjectMeta.Name }}"
	emTemplate := "EM {{ .ObjectMeta.Name }}"
	monitor := datadog.Monitor{
		Name:    &nameTemplate,
		Query:   &queryTemplate,
		Message: &messageTemplate,
		Options: &datadog.Options{
			EscalationMessage: &emTemplate,
		},
	}
	event := config.Event{
		Key:          "a",
		EventType:    "b",
		Namespace:    "c",
		ResourceType: "d",
	}
	err := applyTemplate(deployment, &monitor, &event)
	assert.Equal(t, nil, err, "Error should be nil")
	assert.Equal(t, "Name foo", *monitor.Name, "Name template should be filled")
	assert.Equal(t, "Query foo", *monitor.Query, "Query template should be filled")
	assert.Equal(t, "Message foo", *monitor.Message, "Message template should be filled")
	assert.Equal(t, "EM foo", *monitor.Options.EscalationMessage, "EM template should be filled")
}

func TestParseOverrides(t *testing.T) {
	annotations := map[string]string{
		"astro.fairwinds.com/override.dep-monitor.name": "Deployment Monitor Name Override",
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Annotations: annotations,
		},
	}
	overrides := parseOverrides(deployment)
	assert.Equal(t, len(overrides), 1)
	assert.IsType(t, map[string][]config.Override{}, overrides)
	for k := range overrides {
		assert.Equal(t, "dep-monitor", k)
		assert.Equal(t, []config.Override{{Field: "name", Value: "Deployment Monitor Name Override"}}, overrides[k])
	}
}
