package controller

import (
	"syscall"
	"testing"
	"time"

	"github.com/fairwindsops/dd-manager/pkg/kube"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func TestCreateDeploymentController(t *testing.T) {
	kubeClient := kube.SetAndGetMock()
	DeploymentInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kubeClient.Client.AppsV1().Deployments("").List(metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kubeClient.Client.AppsV1().Deployments("").Watch(metav1.ListOptions{})
			},
		},
		&appsv1.Deployment{},
		0,
		cache.Indexers{},
	)
	DeployWatcher := createController(kubeClient.Client, DeploymentInformer, "deployment")

	annotations := make(map[string]string, 1)
	annotations["test"] = "yup"
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "owned-namespace",
			Annotations: annotations,
		},
	}
	kubeClient.Client.CoreV1().Namespaces().Create(ns)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	kubeClient.Client.AppsV1().Deployments("owned-namespace").Create(deploy)

	assert.Implements(t, (*kubernetes.Interface)(nil), DeployWatcher.kubeClient, "")
	assert.Implements(t, (*cache.SharedIndexInformer)(nil), DeployWatcher.informer, "")
	assert.Implements(t, (*workqueue.RateLimitingInterface)(nil), DeployWatcher.wq, "")
}

func TestNewController(t *testing.T) {
	kube.SetAndGetMock()

	go func() {
		time.Sleep(1 * time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()
	NewController()
}
