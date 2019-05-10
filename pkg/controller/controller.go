package controller

import (
  log "github.com/sirupsen/logrus"
  "github.com/reactiveops/dd-manager/conf"
  "time"
  "k8s.io/client-go/util/workqueue"
  "k8s.io/client-go/tools/cache"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  "k8s.io/apimachinery/pkg/util/wait"
  "os"
  "os/signal"
  "k8s.io/client-go/kubernetes"
  "k8s.io/apimachinery/pkg/runtime"
  "k8s.io/apimachinery/pkg/watch"
  "syscall"
  rt "k8s.io/apimachinery/pkg/util/runtime"
  "fmt"
  handler "github.com/reactiveops/dd-manager/pkg/handler"
)



type KubeResourceWatcher struct {
  kubeClient  kubernetes.Interface
  informer    cache.SharedIndexInformer
  wq          workqueue.RateLimitingInterface
}


func (watcher *KubeResourceWatcher) Watch(term <-chan struct{}) {
  log.Infof("Starting watcher.")

  defer watcher.wq.ShutDown()
  defer rt.HandleCrash()

  go watcher.informer.Run(term)

  if !cache.WaitForCacheSync(term, watcher.HasSynced) {
    rt.HandleError(fmt.Errorf("Timeout waiting for cache sync."))
    return
  }

  log.Infof("Watcher synced.")
  wait.Until(watcher.waitForEvents, time.Second, term)
}


func (watcher *KubeResourceWatcher) waitForEvents() {
  // just keep running forever
  for watcher.next() {

  }
}


func (watcher *KubeResourceWatcher) HasSynced() bool {
  return watcher.informer.HasSynced()
}


func (watcher *KubeResourceWatcher) LastSyncResourceVersion() string {
  return watcher.informer.LastSyncResourceVersion()
}


func (watcher *KubeResourceWatcher) process(evt conf.Event) error {
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
  processErr := watcher.process(evt.(conf.Event))
  if processErr != nil {
    // limit the number of retries
    if watcher.wq.NumRequeues(evt) < 5 {
      log.Errorf("Error running queued item %s: %v", evt.(conf.Event).Key, processErr)
      log.Infof("Retry processing item %s", evt.(conf.Event).Key)
      watcher.wq.AddRateLimited(evt)
    } else {
      log.Errorf("Giving up trying to run queued item %s: %v", evt.(conf.Event).Key, processErr)
      watcher.wq.Forget(evt)
      rt.HandleError(processErr)
    }
  }
  return true
}



func NewController(cfg *conf.Config, kubeClient kubernetes.Interface) {
  log.Info("Starting controller.")
  //kubeClient := util.GetKubeClient()

  log.Infof("Creating watcher for Deployments.")
  DeploymentInformer := cache.NewSharedIndexInformer(
    &cache.ListWatch{
      ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
        return kubeClient.AppsV1().Deployments("").List(metav1.ListOptions{})
      },
      WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
        return kubeClient.AppsV1().Deployments("").Watch(metav1.ListOptions{})
      },
    },
    &v1.Deployment{},
    0,
    cache.Indexers{},
  )

  DeployWatcher := createController(kubeClient, DeploymentInformer, "deployment")
  dTerm := make(chan struct{})
  defer close(dTerm)
  go DeployWatcher.Watch(dTerm)



  log.Infof("Creating watcher for Namespaces.")
  NSInformer := cache.NewSharedIndexInformer(
    &cache.ListWatch{
      ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
        return kubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
      },
      WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
        return kubeClient.CoreV1().Namespaces().Watch(metav1.ListOptions{})
      },
    },
    &corev1.Namespace{},
    0,
    cache.Indexers{},
  )

  NSWatcher := createController(kubeClient, NSInformer, "namespace")
  nsTerm := make(chan struct{})
  defer close(nsTerm)
  go NSWatcher.Watch(nsTerm)


  // create a channel to respond to SIGTERMs
  signals := make(chan os.Signal, 1)
  signal.Notify(signals, syscall.SIGTERM)
  signal.Notify(signals, syscall.SIGINT)
  <-signals
}


func createController(kubeClient kubernetes.Interface, informer cache.SharedIndexInformer, resource string) *KubeResourceWatcher {
  log.Infof("Creating controller for resource type %s", resource)
  wq := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
  var err error
  var evt conf.Event

  informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) {
      evt.Key, err = cache.MetaNamespaceKeyFunc(obj)
      evt.EventType = "create"
      evt.ResourceType = resource
      log.Infof("%s/%s has been added.", resource, evt.Key)
      wq.Add(evt)
    },
    DeleteFunc: func(obj interface{}) {
      evt.Key, err = cache.MetaNamespaceKeyFunc(obj)
      evt.EventType = "delete"
      evt.ResourceType = resource
      log.Infof("%s/%s has been deleted.", resource, evt.Key)
      wq.Add(evt)
    },
    UpdateFunc: func(old interface{}, new interface{}) {
      evt.Key, err = cache.MetaNamespaceKeyFunc(new)
      evt.EventType = "update"
      evt.ResourceType = resource
      log.Infof("%s/%s has been updated.", resource, evt.Key)
      wq.Add(evt)
    },
  })

  return &KubeResourceWatcher{
    kubeClient: kubeClient,
    informer:     informer,
    wq:           wq,
  }
}
