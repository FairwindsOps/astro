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

package datadog

import (
	"errors"
	"reflect"
	"sync"

	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
	ddapi "github.com/zorkian/go-datadog-api"

	"github.com/fairwindsops/astro/pkg/config"
	"github.com/fairwindsops/astro/pkg/metrics"
)

// ClientAPI defines the interface for the Datadog client, for testing purposes
type ClientAPI interface {
	CreateMonitor(*ddapi.Monitor) (*ddapi.Monitor, error)
	DeleteMonitor(id int) error
	GetMonitorsByMonitorTags(tags []string) ([]ddapi.Monitor, error)
	MuteMonitorScope(id int, muteMonitorScope *ddapi.MuteMonitorScope) error
	UnmuteMonitor(id int) error
	UpdateMonitor(*ddapi.Monitor) error
}

// DDMonitorManager is a higher-level wrapper around the Datadog API
type DDMonitorManager struct {
	Datadog ClientAPI
	mux     sync.Mutex
}

var ddMonitorManagerInstance *DDMonitorManager

// GetInstance returns a singleton DDMonitorManager, creating it if necessary
func GetInstance() *DDMonitorManager {
	if ddMonitorManagerInstance == nil {
		conf := config.GetInstance()
		ddMonitorManagerInstance = &DDMonitorManager{
			Datadog: ddapi.NewClient(conf.DatadogAPIKey, conf.DatadogAppKey),
		}
	}
	return ddMonitorManagerInstance
}

// AddOrUpdate will create a monitor if it doesn't exist or update one if it does.
// It returns the Id of the monitor created or updated.
func (ddman *DDMonitorManager) AddOrUpdate(monitor *ddapi.Monitor) (*ddapi.Monitor, error) {
	log.Debugf("Update templated monitor: %v", *monitor.Name)
	ddman.mux.Lock()
	defer ddman.mux.Unlock()

	// check if monitor exists
	ddMonitor, err := ddman.GetProvisionedMonitor(monitor)
	if err != nil {
		//monitor doesn't exist
		log.Infof("Creating new monitor: %v", *monitor.Name)
		provisioned, err := ddman.Datadog.CreateMonitor(monitor)

		if err != nil {
			metrics.DatadogErrCounter.Inc()
			log.Errorf("Error creating monitor %s: %s", *monitor.Name, err)
			return nil, err
		}
		return provisioned, nil
	}

	merged, err := mergeMonitors(*monitor, *ddMonitor)
	if err != nil {
		return nil, err
	}
	//monitor exists
	if reflect.DeepEqual(*merged, *ddMonitor) {
		log.Infof("Monitor exists and is up to date: %v", *ddMonitor.Name)
	} else {
		// monitor exists and needs updating.
		log.Infof("Monitor needs updating: %v", *ddMonitor.Name)
		err := ddman.Datadog.UpdateMonitor(merged)
		if err != nil {
			metrics.DatadogErrCounter.Inc()
			log.Errorf("Could not update monitor: %v, error: %s", *ddMonitor.Name, err)
			return ddMonitor, err
		}
	}
	return ddMonitor, nil
}

// GetProvisionedMonitor returns a monitor with the same name from Datadog.
func (ddman *DDMonitorManager) GetProvisionedMonitor(monitor *ddapi.Monitor) (*ddapi.Monitor, error) {
	monitors, err := ddman.GetProvisionedMonitors()
	if err != nil {
		metrics.DatadogErrCounter.Inc()
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

// GetProvisionedMonitors returns a collection of monitors managed by astro.
func (ddman *DDMonitorManager) GetProvisionedMonitors() ([]ddapi.Monitor, error) {
	return ddman.Datadog.GetMonitorsByMonitorTags([]string{config.GetInstance().OwnerTag})
}

// DeleteMonitor deletes a monitor
func (ddman *DDMonitorManager) DeleteMonitor(monitor *ddapi.Monitor) error {
	ddMonitor, err := ddman.GetProvisionedMonitor(monitor)
	if err != nil {
		return ddman.Datadog.DeleteMonitor(*ddMonitor.Id)
	}
	return nil
}

// DeleteMonitors deletes monitors containing the specified tags.
func (ddman *DDMonitorManager) DeleteMonitors(tags []string) error {
	monitors, err := ddman.Datadog.GetMonitorsByMonitorTags(tags)
	if err != nil {
		metrics.DatadogErrCounter.Inc()
		return err
	}

	log.Infof("Deleting %d monitors.", len(monitors))

	for _, ddMonitor := range monitors {
		log.Infof("Deleting monitor with id %d", *ddMonitor.Id)
		err := ddman.Datadog.DeleteMonitor(*ddMonitor.Id)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteExtinctMonitors gathers monitors configured with all tags in variable tags;  If any are not present in variable monitors they get deleted.
func DeleteExtinctMonitors(monitors []string, tags []string) error {
	ddMan := GetInstance()
	existing, err := ddMan.Datadog.GetMonitorsByMonitorTags(tags)
	if err != nil {
		metrics.DatadogErrCounter.Inc()
		log.Infof("Error getting monitors: %v", err)
		return err
	}

	for _, monitor := range existing {
		if !contains(monitors, monitor) {
			// monitor should no longer exist
			log.Infof("Removing monitor: %v", *monitor.Name)
			err = ddMan.Datadog.DeleteMonitor(*monitor.Id)
			if err != nil {
				metrics.DatadogErrCounter.Inc()
				log.Warnf("Error deleting extinct monitor %d: %v", *monitor.Id, err)
				return err
			}
		}
	}
	return nil
}

// mergeMonitors fills in zero/nil values in our proposed monitor with values that already exist from the DD API
func mergeMonitors(newMon, baseMon ddapi.Monitor) (*ddapi.Monitor, error) {
	err := mergo.Merge(newMon.Options, baseMon.Options)
	if err != nil {
		return &ddapi.Monitor{}, err
	}
	creator := ddapi.Creator{}
	newMon.Creator = &creator
	err = mergo.Merge(newMon.Creator, baseMon.Creator)
	if err != nil {
		return &ddapi.Monitor{}, err
	}
	newMon.OverallState = baseMon.OverallState
	newMon.Id = baseMon.Id
	return &newMon, nil
}

// contains returns a boolean indicating whether a collection of strings contains a monitor with the name of monitor item.
func contains(collection []string, item ddapi.Monitor) bool {
	for _, name := range collection {
		if name == *item.Name {
			return true
		}
	}
	return false
}
