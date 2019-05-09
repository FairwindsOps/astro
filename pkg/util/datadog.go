package util


import (
  log "github.com/sirupsen/logrus"
  "github.com/reactiveops/dd-manager/conf"
  "github.com/zorkian/go-datadog-api"
  "errors"
  "reflect"
)


func AddOrUpdate(monitor *conf.Monitor) (*int, error) {
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


func GetProvisionedMonitor(monitor *conf.Monitor) (*datadog.Monitor, error) {
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


func GetProvisionedMonitors() ([]datadog.Monitor, error) {
  client := getDDClient()
  return client.GetMonitorsByTags([]string{conf.New().OwnerTag})
}


func DeleteMonitor(monitor *conf.Monitor) error {
  client := getDDClient()
  ddMonitor, err := GetProvisionedMonitor(monitor)
  if err != nil {
    return client.DeleteMonitor(*ddMonitor.Id)
  }
  return nil
}


//delete monitors containing the specified tags
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
  config := conf.New()
  return datadog.NewClient(config.DatadogApiKey, config.DatadogAppKey)
}


func toDdMonitor(in *conf.Monitor) *datadog.Monitor {
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


func toMonitor(in *datadog.Monitor) *conf.Monitor {
  thresholds := conf.Thresholds {
    Ok:               in.Options.Thresholds.Ok,
    Critical:         in.Options.Thresholds.Critical,
    Warning:          in.Options.Thresholds.Warning,
    Unknown:          in.Options.Thresholds.Unknown,
    CriticalRecovery: in.Options.Thresholds.CriticalRecovery,
    WarningRecovery:  in.Options.Thresholds.WarningRecovery,
  }

  monitor := conf.Monitor {
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
