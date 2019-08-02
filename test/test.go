package test

import (
	"os"

	// "github.com/fairwindsops/dd-manager/pkg/datadog"
	"github.com/fairwindsops/dd-manager/pkg/kube"
	mocks "github.com/fairwindsops/dd-manager/pkg/mocks"
	"github.com/golang/mock/gomock"
	"github.com/zorkian/go-datadog-api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type ClientAPI interface {
	GetMonitorsByTags(tags []string) ([]datadog.Monitor, error)
	CreateMonitor(*datadog.Monitor) (*datadog.Monitor, error)
	UpdateMonitor(*datadog.Monitor) error
	DeleteMonitor(id int) error
}

// DDMonitorManager is a higher-level wrapper around the Datadog API
type DDMonitorManager struct {
	Datadog ClientAPI
}

func SetupTests(ctrl *gomock.Controller) (*kube.ClientInstance, *mocks.MockClientAPI) {
	os.Setenv("DEFINITIONS_PATH", "../config/test_conf.yml")
	os.Setenv("DD_API_KEY", "test")
	os.Setenv("DD_APP_KEY", "test")

	kubeClient := kube.ClientInstance{
		Client: fake.NewSimpleClientset(),
	}
	kube.SetInstance(kubeClient)

	// ddMon := datadog.GetInstance()
	var ddMon = &DDMonitorManager{}
	ddMock := mocks.NewMockClientAPI(ctrl)
	ddMon.Datadog = ddMock

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	kubeClient.Client.CoreV1().Namespaces().Create(ns)

	return &kubeClient, ddMock
}
