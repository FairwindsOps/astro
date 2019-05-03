package util


import (
  log "github.com/sirupsen/logrus"
  "github.com/reactiveops/dd-manager/conf"
  "github.com/zorkian/go-datadog-api"
  "errors"
)


func AddOrUpdate(config *conf.Config, monitor *conf.Monitor) (*int, error) {
  // check if monitor exists
  ddMonitor, err := GetProvisionedMonitor(config, monitor)
  if err != nil {
    //monitor doesn't exist
    provisioned, err := createMonitor(config, convertMonitor(monitor))
    if err != nil {
      log.Errorf("Error creating monitor %s: %s", monitor.Name, err)
      return nil, err
    }
    return provisioned.Id, nil
  }

  //monitor exists
  //TODO - does monitor need updating?
  log.Infof("Monitor %s exists.", ddMonitor.Id)
  return nil, nil
}


func GetProvisionedMonitor(config *conf.Config, monitor *conf.Monitor) (*datadog.Monitor, error) {
  monitors, err := GetProvisionedMonitors(config)
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


func GetProvisionedMonitors(config *conf.Config) ([]datadog.Monitor, error) {
  client := getDDClient(config)
  return client.GetMonitorsByTags([]string{config.OwnerTag})
}


func DeleteMonitor(config *conf.Config, monitor *conf.Monitor) error {
  client := getDDClient(config)
  ddMonitor, err := GetProvisionedMonitor(config, monitor)
  if err != nil {
    return client.DeleteMonitor(ddMonitor.Id)
  }
  return nil 
}


func createMonitor(config *conf.Config, monitor *datadog.Monitor) (*datadog.Monitor, error) {
  client := getDDClient(config)
  return client.CreateMonitor(monitor)
}


func UpdateMonitor(config *conf.Config, monitor *datadog.Monitor) error {
  //TODO - Update monitor
  return nil
}


func getDDClient(config *conf.Config) *datadog.Client {
  client := datadog.NewClient(config.DatadogApiKey, config.DatadogAppKey)
  return client
}


func convertMonitor(in *conf.Monitor) *datadog.Monitor {
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
