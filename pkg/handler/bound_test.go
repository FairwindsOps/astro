package handler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fairwindsops/astro/pkg/config"
	"github.com/fairwindsops/astro/pkg/datadog"
	"github.com/fairwindsops/astro/pkg/kube"
)

func TestUpdateBoundResources(t *testing.T) {
	ctrl := gomock.NewController(t)
	kubeClient := kube.SetAndGetMock()
	ddMock := datadog.GetMock(ctrl)
	defer ctrl.Finish()

	nsAnnotations := make(map[string]string, 1)
	nsAnnotations["test"] = "yup"
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "bound",
			Annotations: nsAnnotations,
		},
	}
	_, err := kubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	assert.NoError(t, err)
	depAnnotations := make(map[string]string)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Annotations: depAnnotations,
		},
	}
	_, err = kubeClient.Client.AppsV1().Deployments(ns.Name).Create(context.TODO(), dep, metav1.CreateOptions{})
	assert.NoError(t, err)

	tags := []string{"astro"}
	depTags := []string{"astro", "astro:object_type:deployment", "astro:resource:bound/foo"}
	getTagsCall := ddMock.
		EXPECT().
		GetMonitorsByMonitorTags(tags)
	ddMock.
		EXPECT().
		GetMonitorsByMonitorTags(depTags)
	ddMock.
		EXPECT().
		CreateMonitor(gomock.Any()).
		After(getTagsCall)

	event := config.Event{
		EventType:    "create",
		Namespace:    "bound",
		ResourceType: "namespace",
	}
	OnNamespaceChanged(ns, event)
}

func TestSetupBoundEvent(t *testing.T) {
	depAnnotations := make(map[string]string)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-dep",
			Annotations: depAnnotations,
			Namespace:   "foo",
		},
	}

	event := setupBoundEvent(dep)
	assert.IsType(t, config.Event{}, event)
	assert.Equal(t, "foo/test-dep", event.Key)
	assert.Equal(t, "foo", event.Namespace)
	assert.Equal(t, "deployment", event.ResourceType)
	assert.Equal(t, "update", event.EventType)
}
