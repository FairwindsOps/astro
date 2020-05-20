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

package controller

import (
	"context"
    "github.com/fairwindsops/astro/pkg/datadog"
    "github.com/fairwindsops/astro/pkg/metrics"
    log "github.com/sirupsen/logrus"
    "time"
	ddapi "github.com/zorkian/go-datadog-api"
	"github.com/fairwindsops/astro/pkg/config"
)

// StaticController controls static monitors
type StaticController struct {
    Monitors []ddapi.Monitor
}

func (s *StaticController) Run(ctx context.Context) {
    log.Debugf("Creating controller for resource type %s", "static")
    dd := datadog.GetInstance()
    for {
        log.Info("Searching for static monitors to update")
        time.Sleep(5 * time.Minute)

        cfg := config.GetInstance()
        for _, monitor := range *cfg.GetMatchingMonitors(map[string]string{}, "static",map[string][]config.Override{}) {
            log.Debugf("Reconcile monitor %s", *monitor.Name)
            if cfg.DryRun == false {
                _, err := dd.AddOrUpdate(&monitor)
				metrics.ChangeCounter.WithLabelValues("static", "create_update").Inc()
                if err != nil {
					metrics.ErrorCounter.Inc()
					log.Errorf("Error adding/updating monitor")
				}
            } else {
                log.Info("Running as DryRun, skipping DataDog update")
            }
        }
    }

    select {
	case <-ctx.Done():
		log.Info("Shutting down static controller")
		return
	}
}