package handler

import (
	"testing"

	"github.com/fairwindsops/dd-manager/pkg/config"
	"github.com/fairwindsops/dd-manager/pkg/datadog"
	"github.com/fairwindsops/dd-manager/pkg/kube"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateBoundResources(t *testing.T) {
	ctrl := gomock.NewController(t)
	kube.SetMock()
	kubeClient := kube.GetInstance()
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
	kubeClient.Client.CoreV1().Namespaces().Create(ns)
	depAnnotations := make(map[string]string, 0)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Annotations: depAnnotations,
		},
	}
	kubeClient.Client.AppsV1().Deployments(ns.Name).Create(dep)

	tags := []string{"dd-manager"}
	depTags := []string{"dd-manager", "dd-manager:object_type:deployment", "dd-manager:resource:bound/foo"}
	getTagsCall := ddMock.
		EXPECT().
		GetMonitorsByTags(tags)
	ddMock.
		EXPECT().
		GetMonitorsByTags(depTags)
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
	depAnnotations := make(map[string]string, 0)
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
