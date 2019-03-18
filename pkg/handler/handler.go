package handler


import (
  log "github.com/sirupsen/logrus"
)



type Handler interface {
	OnUpdate(obj interface{})
	OnDelete(obj interface{})
	OnCreate(obj iterface{})
}


type EventHandler struct {

}



func (handler *EventHandler) OnUpdate(obj interface{}) {
	log.Info("Handler got an OnUpdate event.")
}


func (handler *EventHandler) OnDelete(obj interface{}) {
	log.Info("Handler got an OnDelete event.")
}


func (handler *EventHandler) OnCreate(obj interface{}) {
	log.Info("Handler got an OnCreate event.")
}