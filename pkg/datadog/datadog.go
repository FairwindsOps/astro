// Copyright 2019 ReactiveOps
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

package datadog

import (
	"errors"
	"reflect"

	"github.com/reactiveops/dd-manager/pkg/config"
	log "github.com/sirupsen/logrus"
	"github.com/zorkian/go-datadog-api"
)

// ClientAPI defines the interface for the Datadog client, for testing purposes
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

var ddMonitorManagerInstance *DDMonitorManager

// GetInstance returns a singleton DDMonitorManager, creating it if necessary
func GetInstance() *DDMonitorManager {
	if ddMonitorManagerInstance == nil {
		config := config.GetInstance()
		ddMonitorManagerInstance = &DDMonitorManager{
			Datadog: datadog.NewClient(config.DatadogAPIKey, config.DatadogAppKey),
		}
	}
	return ddMonitorManagerInstance
}

// AddOrUpdate will create a monitor if it doesn't exist or update one if it does.
// It returns the Id of the monitor created or updated.
func (ddman *DDMonitorManager) AddOrUpdate(monitor *datadog.Monitor) (*datadog.Monitor, error) {
	log.Infof("Update templated monitor:%v", *monitor.Name)
	// check if monitor exists
	ddMonitor, err := ddman.GetProvisionedMonitor(monitor)
	if err != nil {
		//monitor doesn't exist
		provisioned, err := ddman.Datadog.CreateMonitor(monitor)
		if err != nil {
			log.Errorf("Error creating monitor %s: %s", *monitor.Name, err)
			return nil, err
		}
		return provisioned, nil
	}

	//monitor exists
	if reflect.DeepEqual(monitor, ddMonitor) {
		log.Infof("Monitor %d exists and is up to date.", ddMonitor.Id)
	} else {
		// monitor exists and needs updating.
		log.Infof("Monitor %d needs updating.", ddMonitor.Id)

		//TODO - do a deep merge of monitors
		updated := datadog.Monitor(*monitor)
		updated.Id = ddMonitor.Id

		err := ddman.Datadog.UpdateMonitor(&updated)
		if err != nil {
			log.Errorf("Could not update monitor %d: %s", ddMonitor.Id, err)
			return ddMonitor, err
		}
	}
	return ddMonitor, nil
}

// GetProvisionedMonitor returns a monitor with the same name from Datadog.
func (ddman *DDMonitorManager) GetProvisionedMonitor(monitor *datadog.Monitor) (*datadog.Monitor, error) {
	monitors, err := ddman.GetProvisionedMonitors()
	if err != nil {
		log.Errorf("Error getting monitors: %v", err)
		return nil, err
	}

	for _, ddMonitor := range monitors {
		if *ddMonitor.Name == *monitor.Name {
			return &ddMonitor, nil
		}
	}
	return nil, errors.New("monitor does not exist")
}

// GetProvisionedMonitors returns a collection of monitors managed by dd-manager.
func (ddman *DDMonitorManager) GetProvisionedMonitors() ([]datadog.Monitor, error) {
	return ddman.Datadog.GetMonitorsByTags([]string{config.GetInstance().OwnerTag})
}

// DeleteMonitor deletes a monitor
func (ddman *DDMonitorManager) DeleteMonitor(monitor *datadog.Monitor) error {
	ddMonitor, err := ddman.GetProvisionedMonitor(monitor)
	if err != nil {
		return ddman.Datadog.DeleteMonitor(*ddMonitor.Id)
	}
	return nil
}

// DeleteMonitors deletes monitors containing the specified tags.
func (ddman *DDMonitorManager) DeleteMonitors(tags []string) error {
	monitors, err := ddman.Datadog.GetMonitorsByTags(tags)

	log.Infof("Deleting %d monitors.", len(monitors))
	if err != nil {
		log.Errorf("Error getting monitors: %v", err)
		return err
	}

	for _, ddMonitor := range monitors {
		log.Infof("Deleting monitor with id %d", *ddMonitor.Id)
		ddman.Datadog.DeleteMonitor(*ddMonitor.Id)
	}
	return nil
}

// DeleteExtinctMonitors gathers monitors configured with all tags in variable tags;  If any are not present in variable monitors they get deleted.
func DeleteExtinctMonitors(monitors []string, tags []string) error {
	ddMan := GetInstance()
	existing, err := ddMan.Datadog.GetMonitorsByTags(tags)
	if err != nil {
		log.Infof("Error getting monitors: %v", err)
		return err
	}

	for _, monitor := range existing {
		if !contains(monitors, monitor) {
			// monitor should no longer exist
			log.Infof("Found monitor %s that shouldn't exist.", *monitor.Name)
			err = ddMan.Datadog.DeleteMonitor(*monitor.Id)
			if err != nil {
				log.Warnf("Error deleting extinct monitor %d: %v", *monitor.Id, err)
				return err
			}
		}
	}
	return nil
}

// contains returns a boolean indicating whether a collection of strings contains a monitor with the name of monitor item.
func contains(collection []string, item datadog.Monitor) bool {
	for _, name := range collection {
		if name == *item.Name {
			return true
		}
	}
	return false
}
