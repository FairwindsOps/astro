package controller


import (
  log "github.com/sirupsen/logrus"
  "github.com/mjhuber/dd-manager/conf"
  "github.com/zorkian/go-datadog-api"
  "errors"
)








func GetProvisionedMonitor(config conf.Config, monitor conf.Monitor) (*datadog.Monitor, error) {
  client := datadog.NewClient(config.DatadogApiKey, config.DatadogAppKey)
  monitors, err := client.GetMonitorsByTags([]string{config.OwnerTag})
  if err != nil {
    log.Errorf("Error getting monitors: %v", err)
    return nil, err
  }

  for _, ddMonitor := range monitors {
    if *ddMonitor.Name == monitor.Name {
      // monitor exists - return it
      return &ddMonitor, nil
    }
  }
  return nil, errors.New("Monitor does not exist.")
}
