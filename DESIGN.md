# astro

## People
* Micah Huber
* Luke Reed
* Bader Boland

## Intent
The goal of this project is to automate the management of Datadog monitors for Kubernetes clusters.  Given the dynamic nature of distributed Kubernetes systems, monitoring must frequently adapt to match the state of clusters.  Currently, no solution exists to address this need automatically.  Existing tools rely on manually configuring monitoring state which introduces toil to SRE teams.  Additionally, monitoring is often an after-thought and seldom gets adjusted promptly to serve new or changing workloads.  This introduces risk to availability and performance assurance because monitors may not be present or accurate to trigger changes in KPIs (Key Performance Indicators).  The result can be breaches in SLAs because they weren't detected and noisy pagers contributing to pager fatigue.

A tool to automatically manage monitors helps manage these risks.  In general, the desired state of monitors for a cluster can be defined and repeated for each workload that is deployed.  This ensures consistent alerting with little human intervention.


## Key Elements
Key elements for this project include:

* Automated management of the lifecycle of datadog monitors for workloads running in Kubernetes.  Given configuration parameters, the utility will automatically manage defined monitors for all relevant objects within the Kubernetes cluster.  As objects change, monitors are updated to reflect that state.

* Correlation between logically bound objects.  For example, since a namespace is a logical boundary, the tool will have the ability to manage monitors for all objects within the namespace.

* Templating of values from Kubernetes objects into managed monitors.  Any data from a managed kubernetes object can be inserted into a managed monitor.  This makes more informative alerts and can make monitors more context specific.


## Scope

### In Scope:
* Automated management of Datadog Monitors pertaining to Kubernetes
* Automated management of synthetic transactions pertaining to Kubernetes

### Out of Scope:
* Management of Datadog monitors for objects that reside outside of a Kubernetes Cluster
* Support for monitoring systems other than Datadog
* Management of metric collection
* Management of APM configuration
* Management of Datadog agent


## Architecture
The main components of this project include a single binary controller that runs in the Kubernetes cluster and a configuration provided to the binary.

### Binary
The application contains the following core components:
* Config.  Configuration for the application.  It contains information about monitors to apply to objects, api keys, etc.  The main source of information is from environment variables and a configuration file.
* Controller.  The controller interacts with the Kubernetes api and handles registering and listening for updated events as well as calling appropriate handlers based on received events.
* Handlers.  Handlers receive events from the controller and respond to them.  They determine a desired monitoring state and create monitor objects that represent a datadog monitor.
* Utils.  Utilities for the application.  One important utility is the interaction with the datadog api to create, update, or destroy monitors.

### Configuration

Configuration is specified via a combination of environment variables and Custom Resources.  Environment variables specify things like API keys for the application. The custom resources mainly contain information about the monitoring state that is desired in the cluster.

#### ClusterAlertConfiguration

This is a cluster wide object. This defines a reusable set of endpoints to notify when an alert is triggered. Any configuration for these endpoints must already be configured in DataDog.

```yaml
kind: ClusterAlertConfiguration
metadata:
  name: alert-sample
spec:
  alwaysTarget: "@slack-general" # any time there is a message this target will be notified.
  noDataTarget: "@slack-nodata" # this target will be notified whenever the monitor is triggered from having no data.
  alertTarget: "@slack-alert" # this target will be notified when the monitor reaches the Alert threshold
  warningTarget: "@slack-warnings" # this target  will be notified when the monitor reaches the warning threshold
  notRecoveryTarget: "@slack-recovereD" # this target will be notified when either the warning or alert thresholds are met, or there is no data. But not when a monitor is recovered from.
```

#### ClusterMonitorSet

This is a cluster wide object. This defines a set of monitors to be created. They could be static or they could be templated out by all of the targets that match a selector.

```yaml
kind: ClusterMonitorSet
metadata:
  name: cluster-monitor-set
spec:
  targetReferences:
  - apiGroups:
    - ""
    resources:
    - pods
  selector:
    matchLabels:
      app: astro
  monitors:
  - name: "Pod Alert - {{ .ObjectMeta.Name }}"
    type: "metric alert"
    query: "max(last_10m):max:kubernetes.cpu.user.total{namespace:{{ .ObjectMeta.Namespace }},pod_name:{{ .ObjectMeta.Name }}} > 2"
    message: |-
      The CPU usage was too high for {{ .ObjectMeta.Name }}
    tags:
    - tag1
    options:
      no_data_timeframe: 60
      notify_audit: false
      notify_no_data: false
      renotify_interval: 5
      new_host_delay: 5
      evaluation_delay: 300
      timeout: 300
      escalation_message: ""
      threshold_count:
        critical: 0
      require_full_window: true
      locked: false
```

#### MonitorSet

This is a namespaced object. This is a namespace specific equivalent of `ClusterMonitorSet`

#### ClusterMonitorBinding

This is a cluster wide object. This is a binding between a `ClusterMonitorSet` and a `ClusterAlertConfiguration`.

```yaml
kind: ClusterMonitorBinding
metadata:
  name: cluster-monitor-binding
spec:
  monitorSetRef:
    apiGroup: astro.fairwinds.com
    name: cluster-monitor-set
    kind: ClusterMonitorSet
  alertConfigurationRef:
    apiGroup: astro.fairwinds.com
    name: alert-sample
    kind: ClusterAlertConfiguration
```

#### MonitorBinding

This is a namespaced object. This is the namespace specific equivalent of `ClusterMonitorBinding`. This is a binding between either a `ClusterMonitorSet` or a `MonitorSet` and a `ClusterAlertConfiguration`

```yaml
kind: MonitorBinding
metadata:
  name: monitor-binding
  namespace: default
spec:
  monitorSetRef:
    apiGroup: astro.fairwinds.com
    name: monitor-set
    kind: MonitorSet
    namespace: default
  alertConfigurationRef:
    apiGroup: astro.fairwinds.com
    name: alert-sample
    kind: ClusterAlertConfiguration
```

## Related Work

* [Datadog Terraform Provider](https://www.terraform.io/docs/providers/datadog/index.html).  The terraform provider can provision downtime, monitors, synthetics, and dashboards.  Using the provider is effective but it all must be managed manually.

## Possible objections

### Using the Terraform Datadog provider
Datadog providers a terraform provider that can be used to manage monitors.  This is especially beneficial when you already use terraform to manage existing infrastructure.  The disadvantage to this method is that all changes in state must be applied manually.  Using astro, manual intervention can be significantly reduced.
