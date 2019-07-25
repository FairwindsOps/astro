package handler

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/fairwindsops/dd-manager/pkg/config"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeploymentChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	kubeClient, ddMock := setupTests(ctrl)
	defer ctrl.Finish()

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
	kubeClient, _ := setupTests(ctrl)
	defer ctrl.Finish()

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
