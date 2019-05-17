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


package util


import (
  log "github.com/sirupsen/logrus"
  "github.com/reactiveops/dd-manager/pkg/config"
  "github.com/zorkian/go-datadog-api"
  "errors"
  "reflect"
)

// AddOrUpdate will create a monitor if it doesn't exist or update one if it does.
// It returns the Id of the monitor created or updated.
func AddOrUpdate(monitor *config.Monitor) (*int, error) {
  log.Infof("Update templated monitor:\n\n%+v", monitor)
  // check if monitor exists
  ddMonitor, err := GetProvisionedMonitor(monitor)
  if err != nil {
    //monitor doesn't exist
    provisioned, err := createMonitor(toDdMonitor(monitor))
    if err != nil {
      log.Errorf("Error creating monitor %s: %s", monitor.Name, err)
      return nil, err
    }
    return provisioned.Id, nil
  }

  //monitor exists
  if reflect.DeepEqual(monitor, toMonitor(ddMonitor)) {
    log.Infof("Monitor %d exists and is up to date.", ddMonitor.Id)
  } else {
    // monitor exists and needs updating.
    log.Infof("Monitor %d needs updating.", ddMonitor.Id)

    //TODO - do a deep merge of monitors
    updated := toDdMonitor(monitor)
    updated.Id = ddMonitor.Id

    err := updateMonitor(updated)
    if err != nil {
      log.Errorf("Could not update monitor %d: %s", ddMonitor.Id, err)
      return ddMonitor.Id, err
    }
  }
  return ddMonitor.Id, nil
}

// GetProvisionedMonitor returns a monitor with the same name from Datadog.
func GetProvisionedMonitor(monitor *config.Monitor) (*datadog.Monitor, error) {
  monitors, err := GetProvisionedMonitors()
  if err != nil {
    log.Errorf("Error getting monitors: %v", err)
    return nil, err
  }

  for _, ddMonitor := range monitors {
    if *ddMonitor.Name == monitor.Name {
      return &ddMonitor, nil
    }
  }
  return nil, errors.New("Monitor does not exist.")
}

// GetProvisionedMonitors returns a collection of monitors managed by dd-manager.
func GetProvisionedMonitors() ([]datadog.Monitor, error) {
  client := getDDClient()
  return client.GetMonitorsByTags([]string{config.New().OwnerTag})
}


func DeleteMonitor(monitor *config.Monitor) error {
  client := getDDClient()
  ddMonitor, err := GetProvisionedMonitor(monitor)
  if err != nil {
    return client.DeleteMonitor(*ddMonitor.Id)
  }
  return nil
}

// DeleteMonitors deletes monitors containing the specified tags.
func DeleteMonitors(tags []string) error {
  client := getDDClient()
  monitors, err := client.GetMonitorsByTags(tags)

  log.Infof("Deleting %d monitors.", len(monitors))
  if err != nil {
    log.Errorf("Error getting monitors: %v", err)
    return err
  }

  for _, ddMonitor := range monitors {
    log.Infof("Deleting monitor with id %d", *ddMonitor.Id)
    client.DeleteMonitor(*ddMonitor.Id)
  }
  return nil
}


func createMonitor(monitor *datadog.Monitor) (*datadog.Monitor, error) {
  client := getDDClient()
  return client.CreateMonitor(monitor)
}


func updateMonitor(monitor *datadog.Monitor) error {
  client := getDDClient()
  return client.UpdateMonitor(monitor)
}


func getDDClient() *datadog.Client {
  config := config.New()
  return datadog.NewClient(config.DatadogApiKey, config.DatadogAppKey)
}


func toDdMonitor(in *config.Monitor) *datadog.Monitor {
  monitor := datadog.Monitor {
    Type:     &in.Type,
    Query:    &in.Query,
    Name:     &in.Name,
    Message:  &in.Message,
    Tags:     in.Tags,
    Options:  &datadog.Options {
      NoDataTimeframe:  datadog.NoDataTimeframe(in.NoDataTimeframe),
      NotifyAudit:        &in.NotifyAudit,
      NotifyNoData:       &in.NotifyNoData,
      RenotifyInterval:   &in.RenotifyInterval,
      NewHostDelay:       &in.NewHostDelay,
      EvaluationDelay:    &in.EvaluationDelay,
      TimeoutH:           &in.Timeout,
      EscalationMessage:  &in.EscalationMessage,
      Thresholds:         &datadog.ThresholdCount {
        Ok:                 in.Thresholds.Ok,
        Critical:           in.Thresholds.Critical,
        Warning:            in.Thresholds.Warning,
        Unknown:            in.Thresholds.Unknown,
        CriticalRecovery:   in.Thresholds.CriticalRecovery,
        WarningRecovery:    in.Thresholds.WarningRecovery,
      },
      RequireFullWindow:  &in.RequireFullWindow,
      Locked:             &in.Locked,
    },
  }
  return &monitor
}


func toMonitor(in *datadog.Monitor) *config.Monitor {
  thresholds := config.Thresholds {
    Ok:               in.Options.Thresholds.Ok,
    Critical:         in.Options.Thresholds.Critical,
    Warning:          in.Options.Thresholds.Warning,
    Unknown:          in.Options.Thresholds.Unknown,
    CriticalRecovery: in.Options.Thresholds.CriticalRecovery,
    WarningRecovery:  in.Options.Thresholds.WarningRecovery,
  }

  monitor := config.Monitor {
    Name:               *in.Name,
    Type:               *in.Type,
    Query:              *in.Query,
    Message:            *in.Message,
    Tags:               in.Tags,
    NoDataTimeframe:    int(in.Options.NoDataTimeframe),
    NotifyAudit:        *in.Options.NotifyAudit,
    NotifyNoData:       *in.Options.NotifyNoData,
    RenotifyInterval:   *in.Options.RenotifyInterval,
    NewHostDelay:       *in.Options.NewHostDelay,
    EvaluationDelay:    *in.Options.EvaluationDelay,
    Timeout:            *in.Options.TimeoutH,
    EscalationMessage:  *in.Options.EscalationMessage,
    Thresholds:         thresholds,
    RequireFullWindow:  *in.Options.RequireFullWindow,
  }
  return &monitor
}
