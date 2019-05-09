package handler


import (
  log "github.com/sirupsen/logrus"
  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
  "github.com/reactiveops/dd-manager/conf"
  "strings"

  "bytes"
  "text/template"
  "fmt"
)



func OnUpdate(obj interface{}, event conf.Event) {
  log.Infof("Handler got an OnUpdate event of type %s", event.EventType)

  if event.EventType == "delete" {
    onDelete(event)
    return
  }

  switch t := obj.(type) {
  case *appsv1.Deployment:
    OnDeploymentChanged(obj.(*appsv1.Deployment), event)
  case *corev1.Namespace:
    OnNamespaceChanged(obj.(*corev1.Namespace), event)
  default:
    log.Warnf("Object has unknown type of %T", t)
  }
}


func onDelete(event conf.Event) {
  log.Info("OnDelete()")
  switch strings.ToLower(event.ResourceType) {
  case "namespace":
    OnNamespaceChanged(&corev1.Namespace{}, event)
  case "deployment":
    OnDeploymentChanged(&appsv1.Deployment{}, event)
  default:
    log.Warnf("object has unknown resource type %s", event.ResourceType)
  }
}


func applyTemplate(obj interface{}, monitor *conf.Monitor, event *conf.Event) {
  cfg := conf.New()
  var err error
  var tpl bytes.Buffer
  name, _ := template.New("name").Parse(monitor.Name)
  query, _ := template.New("query").Parse(monitor.Query)
  msg, _ := template.New("message").Parse(monitor.Message)
  em, _ := template.New("escalation_message").Parse(monitor.EscalationMessage)

  err = name.Execute(&tpl, obj)
  if err != nil {
    log.Errorf("Error templating name: %s", err)
  }
  monitor.Name = tpl.String()
  tpl.Reset()

  err = query.Execute(&tpl, obj)
  if err != nil {
    log.Errorf("Error templating query: %s", err)
  }
  monitor.Query = tpl.String()
  tpl.Reset()

  err = msg.Execute(&tpl, obj)
  if err != nil {
    log.Error("Error templating message: %s", err)
  }
  monitor.Message = tpl.String()
  tpl.Reset()

  err = em.Execute(&tpl, obj)
  if err != nil {
    log.Errorf("Error templating escalation message: %s", err)
  }
  monitor.EscalationMessage = tpl.String()

  // apply identifying tags
  monitor.Tags = append(monitor.Tags, cfg.OwnerTag, fmt.Sprintf("dd-manager:object_type:%s", event.ResourceType), fmt.Sprintf("dd-manager:resource:%s", event.Key))
}