package handler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fairwindsops/astro/pkg/config"
	"github.com/fairwindsops/astro/pkg/datadog"
	"github.com/fairwindsops/astro/pkg/kube"
)

func TestNamespaceChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	kubeClient := kube.SetAndGetMock()
	ddMock := datadog.GetMock(ctrl)
	defer ctrl.Finish()

	annotations := make(map[string]string, 1)
	annotations["astro/owner"] = "astro"
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "owned-namespace",
			Annotations: annotations,
		},
	}
	kubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	event := config.Event{
		EventType:    "create",
		Namespace:    "owned-namespace",
		ResourceType: "namespace",
	}

	tags := []string{"astro"}
	getTagsCall := ddMock.
		EXPECT().
		GetMonitorsByMonitorTags(tags)
	ddMock.
		EXPECT().
		CreateMonitor(gomock.Any()).
		After(getTagsCall)

	OnNamespaceChanged(ns, event)
}

func TestNamespaceChangeNoMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	kubeClient := kube.SetAndGetMock()
	defer ctrl.Finish()

	annotations := make(map[string]string, 1)
	annotations["astro/owner"] = "not-astro"
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "unowned-namespace",
			Annotations: annotations,
		},
	}
	kubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	event := config.Event{
		EventType:    "create",
		Namespace:    "owned-namespace",
		ResourceType: "namespace",
	}

	// Don't expect any calls to Datadog

	OnNamespaceChanged(ns, event)
}
