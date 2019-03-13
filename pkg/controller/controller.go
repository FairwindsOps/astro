package controller

import (
  log "github.com/sirupsen/logrus"
  "github.com/mjhuber/dd-manager/pkg/util"
  "github.com/mjhuber/dd-manager/conf"
  "time"
  "k8s.io/client-go/util/workqueue"
  "k8s.io/client-go/tools/cache"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/api/apps/v1"
  "k8s.io/apimachinery/pkg/util/wait"
  "os"
  "os/signal"
  "k8s.io/client-go/kubernetes"
  "k8s.io/apimachinery/pkg/runtime"
  "k8s.io/apimachinery/pkg/watch"
  "syscall"
)



type KubeResourceWatcher struct {
  kubeClient  kubernetes.Interface
  informer    cache.SharedIndexInformer
  wq          workqueue.RateLimitingInterface
}


func (watcher *KubeResourceWatcher) Watch(term <-chan struct{}) {
  go watcher.informer.Run(term)
  wait.Until(watcher.waitForEvents, time.Second, term)
}

func (watcher *KubeResourceWatcher) waitForEvents() {
  // just keep running forever
  for {

  }
}





func Run(cfg *conf.Config) {
  log.Info("Starting controller.")
  kubeClient := util.GetKubeClient()

  informer := cache.NewSharedIndexInformer(
    &cache.ListWatch{
      ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
        return kubeClient.AppsV1().Deployments("").List(options)
      },
      WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
        return kubeClient.AppsV1().Deployments("").Watch(options)
      },
    },
    &v1.Deployment{},
    0,
    cache.Indexers{},
  )

  watcher := createController(kubeClient, informer, "Deployment")
  term := make(chan struct{})
  defer close(term)

  go watcher.Watch(term)



  // create a channel to respond to SIGTERMs
  signals := make(chan os.Signal, 1)
  signal.Notify(signals, syscall.SIGTERM)
  signal.Notify(signals, syscall.SIGINT)
  <-signals
}




func createController(kubeClient kubernetes.Interface, informer cache.SharedIndexInformer, resource string) *KubeResourceWatcher {
  wq := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

  informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) {
      log.Infof("%s/%s has been added.", resource, cache.MetaNamespaceKeyFunc(obj))
    },
    DeleteFunc: func(obj interface{}) {
      log.Infof("%s/%s has been deleted.", resource, cache.MetaNamespaceKeyFunc(obj))
    },
    UpdateFunc: func(obj interface{}) {
      log.Infof("%s/%s has been updated.", resource, cache.MetaNamespaceKeyFunc(obj))
    },
  })

  return &KubeResourceWatcher{
    kubeClient: kubeClient,
    informer:     informer,
    wq:           wq,
  }
}
