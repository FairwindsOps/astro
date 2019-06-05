package handler

import (
	"github.com/golang/mock/gomock"
	"github.com/reactiveops/dd-manager/pkg/config"
	"github.com/reactiveops/dd-manager/pkg/datadog"
	"github.com/reactiveops/dd-manager/pkg/kube"
	mocks "github.com/reactiveops/dd-manager/pkg/mocks"
	_ "github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"os"
	"testing"
)

func TestDeploymentChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	os.Setenv("DEFINITIONS_PATH", "../../conf.yml")
	os.Setenv("DD_API_KEY", "test")
	os.Setenv("DD_APP_KEY", "test")

	kubeClient := kube.KubeClient{
		Client: fake.NewSimpleClientset(),
	}
	kube.SetInstance(kubeClient)
	ddMon := datadog.GetInstance()
	ddMock := mocks.NewMockDatadogAPI(ctrl)
	ddMon.Datadog = ddMock

	annotations := make(map[string]string, 1)
	annotations["dd-manager/owner"] = "dd-manager"
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Annotations: annotations,
		},
	}
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	kubeClient.Client.CoreV1().Namespaces().Create(ns)
	kubeClient.Client.AppsV1().Deployments("foo").Create(dep)
	event := config.Event{
		EventType:    "create",
		Namespace:    "foo",
		ResourceType: "deployment",
	}

	tags := []string{"dd-manager"}
	ddMock.
		EXPECT().
		GetMonitorsByTags(tags)
	ddMock.
		EXPECT().
		CreateMonitor(gomock.Any())

	OnDeploymentChanged(dep, event)
}
