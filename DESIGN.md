# astro

## People
* Micah huber
* Luke Reed

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
Configuration is specified via a combination of environment variables and a configuration file.  Environment variables specify things like API keys for the application.  The configuration file mainly contains information about the monitoring state that is desired in the cluster.

Example Configuration File:
```yaml
---
rulesets:
- type: deployment
  match_annotations:
    - name: astro/owner
      value: astro
  monitors:
    - name: "Deployment Replica Alert - {{ .ObjectMeta.Name }}"
      type: metric alert
      query: "max(last_10m):max:kubernetes_state.deployment.replicas_available{kubernetescluster:foobar,namespace:{{ .ObjectMeta.Namespace }}} by {deployment} <= 0"
      message: |
        {{ "{{#is_alert}}" }}
        Available replicas is currently 0 for {{ .ObjectMeta.Name }}
        {{ "{{/is_alert}}" }}
        {{ "{{^is_alert}}" }}
        Available replicas is no longer 0 for {{ .ObjectMeta.Name }}
        {{ "{{/is_alert}}" }}
      tags:
        - astro
      no_data_timeframe: 60
      notify_audit: false
      notify_no_data: false
      renotify_interval: 5
      new_host_delay: 5
      evaluation_delay: 300
      timeout: 300
      escalation_message: ""
      thresholds:
        critical: 0
      require_full_window: true
      locked: false
```

## Related Work
* [Rodd](https://github.com/FairwindsOps/rodd).  Rodd is our current monitor management solution that takes a config file as input and creates terraform as output.  The main differentiators between rodd and astro are:
  * rodd requires manual updates to state, astro does not
  * rodd supports creating monitors for non-kubernetes items

  Rodd's features complement astro because it supports edge cases that astro does not (for example, creating monitors for things that aren't defined in a Kubernetes cluster).

* [Datadog Terraform Provider](https://www.terraform.io/docs/providers/datadog/index.html).  The terraform provider can provision downtime, monitors, synthetics, and dashboards.  Using the provider is effective but it all must be managed manually.

## Possible objections

### Using CRDs to Store Configuration
Use of a CRD to store configuration could be attractive because it would easily enable automatic updates to configuration.  However, one of the potential benefits of this project would be having a global configuration broad enough to apply to multiple clusters.  In this case, it is not desirable to have it live in the cluster and should be stored somewhere easily accessible for all clusters using it to access it.

### Using the Terraform Datadog provider
Datadog providers a terraform provider that can be used to manage monitors.  This is especially beneficial when you already use terraform to manage existing infrastructure.  The disadvantage to this method is that all changes in state must be applied manually.  Using astro, manual intervention can be significantly reduced.
