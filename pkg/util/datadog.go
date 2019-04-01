package util


import (
  log "github.com/sirupsen/logrus"
  "github.com/reactiveops/dd-manager/conf"
  "github.com/zorkian/go-datadog-api"
  "errors"
)


func GetProvisionedMonitor(config conf.Config, monitor conf.Monitor) (*datadog.Monitor, error) {
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


func GetProvisionedMonitors(config conf.Config) ([]datadog.Monitor, error) {
  client := getDDClient(config)
  return client.GetMonitorsByTags([]string{config.OwnerTag})
}


func DeleteMonitor(config conf.Config, AlertId int) error {
  client := getDDClient(config)
  return client.DeleteMonitor(AlertId)
}


func UpdateMonitor(config conf.Config, monitor *datadog.Monitor) error {
  //TODO - Update monitor
  return nil
}


func getDDClient(config conf.Config) *datadog.Client {
  client := datadog.NewClient(config.DatadogApiKey, config.DatadogAppKey)
  return client
}
