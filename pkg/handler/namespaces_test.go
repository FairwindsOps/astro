package handler

import (
	"github.com/golang/mock/gomock"
	"github.com/fairwindsops/dd-manager/pkg/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNamespaceChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	kubeClient, ddMock := setupTests(ctrl)
	defer ctrl.Finish()

	annotations := make(map[string]string, 1)
	annotations["dd-manager/owner"] = "dd-manager"
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "owned-namespace",
			Annotations: annotations,
		},
	}
	kubeClient.Client.CoreV1().Namespaces().Create(ns)
	event := config.Event{
		EventType:    "create",
		Namespace:    "owned-namespace",
		ResourceType: "namespace",
	}

	tags := []string{"dd-manager"}
	getTagsCall := ddMock.
		EXPECT().
		GetMonitorsByTags(tags)
	ddMock.
		EXPECT().
		CreateMonitor(gomock.Any()).
		After(getTagsCall)

	OnNamespaceChanged(ns, event)
}

func TestNamespaceChangeNoMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	kubeClient, _ := setupTests(ctrl)
	defer ctrl.Finish()

	annotations := make(map[string]string, 1)
	annotations["dd-manager/owner"] = "not-dd-manager"
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "unowned-namespace",
			Annotations: annotations,
		},
	}
	kubeClient.Client.CoreV1().Namespaces().Create(ns)
	event := config.Event{
		EventType:    "create",
		Namespace:    "owned-namespace",
		ResourceType: "namespace",
	}

	// Don't expect any calls to Datadog

	OnNamespaceChanged(ns, event)
}
