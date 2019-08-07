package handler

import (
	"testing"

	"github.com/fairwindsops/dd-manager/pkg/config"
	"github.com/fairwindsops/dd-manager/pkg/datadog"
	"github.com/fairwindsops/dd-manager/pkg/kube"
	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeploymentChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	kubeClient := kube.SetAndGetMock()
	ddMock := datadog.GetMock(ctrl)
	defer ctrl.Finish()

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	kubeClient.Client.CoreV1().Namespaces().Create(ns)

	annotations := make(map[string]string, 1)
	annotations["dd-manager/owner"] = "dd-manager"
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Annotations: annotations,
		},
	}
	kubeClient.Client.AppsV1().Deployments("foo").Create(dep)
	event := config.Event{
		EventType:    "create",
		Namespace:    "foo",
		ResourceType: "deployment",
	}

	tags := []string{"dd-manager"}
	getTagsCall := ddMock.
		EXPECT().
		GetMonitorsByTags(tags)
	ddMock.
		EXPECT().
		CreateMonitor(gomock.Any()).
		After(getTagsCall)

	OnDeploymentChanged(dep, event)
}

func TestDeploymentChangeNoMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	kubeClient := kube.SetAndGetMock()
	defer ctrl.Finish()

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	kubeClient.Client.CoreV1().Namespaces().Create(ns)

	annotations := make(map[string]string, 1)
	annotations["dd-manager/owner"] = "not-dd-manager"
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Annotations: annotations,
		},
	}
	kubeClient.Client.AppsV1().Deployments("foo").Create(dep)
	event := config.Event{
		EventType:    "create",
		Namespace:    "foo",
		ResourceType: "deployment",
	}

	// Don't expect any calls to Datadog

	OnDeploymentChanged(dep, event)
}
