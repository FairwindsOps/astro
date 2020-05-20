// Copyright 2019 FairwindsOps Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	rt "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/fairwindsops/astro/pkg/config"
	"github.com/fairwindsops/astro/pkg/handler"
	"github.com/fairwindsops/astro/pkg/kube"
)

type KubeResourceController struct {
    Name string
}

// KubeResourceWatcher contains the informer that watches Kubernetes objects and the queue that processes updates.
type KubeResourceWatcher struct {
	kubeClient kubernetes.Interface
	informer   cache.SharedIndexInformer
	wq         workqueue.RateLimitingInterface
}

// Watch tells the KubeResourceWatcher to start waiting for events
func (watcher *KubeResourceWatcher) Watch(term <-chan struct{}) {
	log.Debugf("Starting watcher.")

	defer watcher.wq.ShutDown()
	defer rt.HandleCrash()

	go watcher.informer.Run(term)

	if !cache.WaitForCacheSync(term, watcher.HasSynced) {
		rt.HandleError(fmt.Errorf("timeout waiting for cache sync"))
		return
	}

	log.Debugf("Watcher synced.")
	wait.Until(watcher.waitForEvents, time.Second, term)
}

func (watcher *KubeResourceWatcher) waitForEvents() {
	// just keep running forever
	for watcher.next() {

	}
}

// HasSynced determines whether the informer has synced
func (watcher *KubeResourceWatcher) HasSynced() bool {
	return watcher.informer.HasSynced()
}

// LastSyncResourceVersion returns the last sync resource version
func (watcher *KubeResourceWatcher) LastSyncResourceVersion() string {
	return watcher.informer.LastSyncResourceVersion()
}

func (watcher *KubeResourceWatcher) process(evt config.Event) error {
	info, _, err := watcher.informer.GetIndexer().GetByKey(evt.Key)

	if err != nil {
		//TODO - need some better error handling here
		return err
	}

	handler.OnUpdate(info, evt)
	return nil
}

func (watcher *KubeResourceWatcher) next() bool {
	evt, err := watcher.wq.Get()

	if err {
		return false
	}

	defer watcher.wq.Done(evt)
	processErr := watcher.process(evt.(config.Event))
	if processErr != nil {
		// limit the number of retries
		if watcher.wq.NumRequeues(evt) < 5 {
			log.Errorf("Error running queued item %s: %v", evt.(config.Event).Key, processErr)
			log.Errorf("Retry processing item %s", evt.(config.Event).Key)
			watcher.wq.AddRateLimited(evt)
		} else {
			log.Errorf("Giving up trying to run queued item %s: %v", evt.(config.Event).Key, processErr)
			watcher.wq.Forget(evt)
			rt.HandleError(processErr)
		}
	}
	return true
}

// Run starts a controller for watching Kubernetes objects.
func (k *KubeResourceController) Run(ctx context.Context) {
	log.Debug("Starting controller.")
	kubeClient := kube.GetInstance()
	log.Debug("Creating watcher for Deployments.")
	DeploymentInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kubeClient.Client.AppsV1().Deployments("").List(metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kubeClient.Client.AppsV1().Deployments("").Watch(metav1.ListOptions{})
			},
		},
		&v1.Deployment{},
		0,
		cache.Indexers{},
	)
	DeployWatcher := k.createController(kubeClient.Client, DeploymentInformer, "deployment")
	dTerm := make(chan struct{})
	defer close(dTerm)
	go DeployWatcher.Watch(dTerm)

	log.Debug("Creating watcher for Namespaces.")
	NSInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kubeClient.Client.CoreV1().Namespaces().List(metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kubeClient.Client.CoreV1().Namespaces().Watch(metav1.ListOptions{})
			},
		},
		&corev1.Namespace{},
		0,
		cache.Indexers{},
	)

	NSWatcher := k.createController(kubeClient.Client, NSInformer, "namespace")
	nsTerm := make(chan struct{})
	defer close(nsTerm)
	go NSWatcher.Watch(nsTerm)

	select {
	case <-ctx.Done():
		log.Info("Shutting down k8s controller")
		return
	}
}

func (k *KubeResourceController) createController(kubeClient kubernetes.Interface, informer cache.SharedIndexInformer, resource string) *KubeResourceWatcher {
	log.Debugf("Creating controller for resource type %s", resource)
	wq := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			var evt config.Event
			var err error
			evt.Key, err = cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				log.Errorf("Error handling add event")
				return
			}
			objMeta := objectMeta(obj)
			evt.EventType = "create"
			evt.ResourceType = resource
			evt.Namespace = objectMeta(obj).Namespace
			evt.NewMeta = &objMeta
			evt.OldMeta = new(metav1.ObjectMeta)
			log.Debugf("%s/%s has been added.", resource, evt.Key)
			wq.Add(evt)
		},
		DeleteFunc: func(obj interface{}) {
			var evt config.Event
			var err error
			evt.Key, err = cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				log.Errorf("Error handling delete event")
				return
			}
			objMeta := objectMeta(obj)
			evt.EventType = "delete"
			evt.ResourceType = resource
			evt.Namespace = objectMeta(obj).Namespace
			evt.NewMeta = new(metav1.ObjectMeta)
			evt.OldMeta = &objMeta
			log.Debugf("%s/%s has been deleted.", resource, evt.Key)
			wq.Add(evt)
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			var evt config.Event
			var err error
			evt.Key, err = cache.MetaNamespaceKeyFunc(new)
			if err != nil {
				log.Errorf("Error handling update event")
				return
			}
			oldMeta := objectMeta(old)
			newMeta := objectMeta(new)
			evt.EventType = "update"
			evt.ResourceType = resource
			evt.Namespace = objectMeta(new).Namespace
			evt.OldMeta = &oldMeta
			evt.NewMeta = &newMeta
			log.Debugf("%s/%s has been updated.", resource, evt.Key)
			wq.Add(evt)
		},
	})

	return &KubeResourceWatcher{
		kubeClient: kubeClient,
		informer:   informer,
		wq:         wq,
	}
}

func objectMeta(obj interface{}) metav1.ObjectMeta {
	var meta metav1.ObjectMeta

	switch object := obj.(type) {
	case *corev1.Namespace:
		meta = object.ObjectMeta
	case *v1.Deployment:
		meta = object.ObjectMeta
	}
	return meta
}
