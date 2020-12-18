package handler

import (
	log "github.com/sirupsen/logrus"

	"github.com/fairwindsops/astro/pkg/config"
	"github.com/fairwindsops/astro/pkg/datadog"
	"github.com/fairwindsops/astro/pkg/metrics"
)

// StaticMonitorUpdate is a handler that should be called by the controller on a timer
func StaticMonitorUpdate(event config.Event) {
	var err error
	var record []string
	cfg := config.GetInstance()
	dd := datadog.GetInstance()
	monitors := cfg.GetStaticMonitors()
	for _, monitor := range *monitors {
		err = applyTemplate(nil, &monitor, &event)
		if err != nil {
			metrics.TemplateErrorCounter.Inc()
			log.Errorf("Error applying template for monitor %s: %v", *monitor.Name, err)
			return
		}
		log.Debugf("Reconcile static monitor %s", *monitor.Name)
		if !cfg.DryRun {
			_, err = dd.AddOrUpdate(&monitor)
			record = append(record, *monitor.Name)
			if err != nil {
				metrics.ErrorCounter.Inc()
				log.Errorf("Error adding/updating static monitor:%s", err)
			} else {
				metrics.ChangeCounter.WithLabelValues("static", "create_update").Inc()
			}
		} else {
			log.Info("Running as DryRun, skipping DataDog update")
		}
	}
	if !cfg.DryRun {
		// if there are any additional monitors, they should be removed.  This could happen if an object
		// was previously monitored and now no longer is.
		err = datadog.DeleteExtinctMonitors(record, []string{cfg.OwnerTag, "astro:object_type:static"})
		if err != nil {
			log.Errorf("Error deleting extinct static monitors:%s", err)
		}
	}
}
