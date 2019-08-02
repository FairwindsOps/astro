package kube

import (
	"k8s.io/client-go/kubernetes/fake"
)

// SetMock sets the singleton's interface to use a fake ClientSet
func SetMock() {
	kubeClient := ClientInstance{
		Client: fake.NewSimpleClientset(),
	}
	SetInstance(kubeClient)
}

// SetInstance allows the user to set the kubeClient singleton
func SetInstance(kc ClientInstance) {
	kubeClient = &kc
}
