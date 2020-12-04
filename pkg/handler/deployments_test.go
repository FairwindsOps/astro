package handler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fairwindsops/astro/pkg/config"
	"github.com/fairwindsops/astro/pkg/datadog"
	"github.com/fairwindsops/astro/pkg/kube"
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
	kubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})

	annotations := make(map[string]string, 1)
	annotations["astro/owner"] = "astro"
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Annotations: annotations,
		},
	}
	kubeClient.Client.AppsV1().Deployments("foo").Create(context.TODO(), dep, metav1.CreateOptions{})
	event := config.Event{
		EventType:    "create",
		Namespace:    "foo",
		ResourceType: "deployment",
	}

	tags := []string{"astro"}
	getTagsCall := ddMock.
		EXPECT().
		GetMonitorsByMonitorTags(tags)
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
	kubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})

	annotations := make(map[string]string, 1)
	annotations["astro/owner"] = "not-astro"
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Annotations: annotations,
		},
	}
	kubeClient.Client.AppsV1().Deployments("foo").Create(context.TODO(), dep, metav1.CreateOptions{})
	event := config.Event{
		EventType:    "create",
		Namespace:    "foo",
		ResourceType: "deployment",
	}

	// Don't expect any calls to Datadog

	OnDeploymentChanged(dep, event)
}
