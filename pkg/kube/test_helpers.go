package kube

import (
	"k8s.io/client-go/kubernetes/fake"
)

// SetAndGetMock sets the singleton's interface to use a fake ClientSet
func SetAndGetMock() *ClientInstance {
	kc := ClientInstance{
		Client: fake.NewSimpleClientset(),
	}
	SetInstance(kc)
	return &kc
}

// SetInstance allows the user to set the kubeClient singleton
func SetInstance(kc ClientInstance) {
	kubeClient = &kc
}
