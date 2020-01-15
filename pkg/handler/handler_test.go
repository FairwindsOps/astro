package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ddapi "github.com/zorkian/go-datadog-api"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fairwindsops/astro/pkg/config"
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
	monitor := ddapi.Monitor{
		Name:    &nameTemplate,
		Query:   &queryTemplate,
		Message: &messageTemplate,
		Options: &ddapi.Options{
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
