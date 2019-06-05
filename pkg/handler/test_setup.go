package handler

import (
	"github.com/golang/mock/gomock"
	"github.com/reactiveops/dd-manager/pkg/datadog"
	"github.com/reactiveops/dd-manager/pkg/kube"
	mocks "github.com/reactiveops/dd-manager/pkg/mocks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"os"
)

func setupTests(ctrl *gomock.Controller) (*kube.KubeClient, *mocks.MockDatadogAPI) {
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

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	kubeClient.Client.CoreV1().Namespaces().Create(ns)

	return &kubeClient, ddMock
}
