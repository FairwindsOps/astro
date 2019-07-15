[![CircleCI](https://circleci.com/gh/reactiveops/dd-manager/tree/master.svg?style=svg&circle-token=77f1eb3b95b59a0372b19fdefbbd28ebfaa9d0c0)](https://circleci.com/gh/reactiveops/dd-manager/tree/master)
[![codecov](https://codecov.io/gh/reactiveops/dd-manager/branch/master/graph/badge.svg?token=6zutKJd2Gy)](https://codecov.io/gh/reactiveops/dd-manager)



# dd-manager
DD-Manager was designed to simplify datadog monitor administration.  This is an operator that emits datadog monitors based on kubernetes state.  The operator responds to changes of resources in your kubernetes cluster and will manage datadog monitors based on the configured state.

## Configuration
A combination of environment variables and a yaml file is used to configure the application.  An example configuration file is available at [here](conf.yml).

### Environment Variables
| Variable    | Descritpion                        | Required  | Default     |
|:------------|:----------------------------------:|:----------|:------------|
| `DD_API_KEY` | The datadog api key for your datadog account. | `Y` ||
| `DD_APP_KEY` | The datadog app key for your datadog account. | `Y` ||
| `OWNER`      | A unique name to designate as the owner.  This will be applied as a tag to identified managed monitors. | `N`| `dd-manager` |
| `DEFINITIONS_PATH` | The path to monitor definition configurations.  This can be a local path or a URL.  Multiple paths shoudl be separated by a `;` | `N` | `conf.yml` |
| `DRY_RUN` | when set to true monitors will not be managed in datadog. | `N` | `false` |

### Configuration File
A configuration file is used to define your monitors.  These are organized as rulesets, which consist of the type of resource the ruleset applies to, annotations that must be present on the resource to be considered valid objects, and a set of monitors to manage for that resource.  Go templating syntax may be used in your monitors and values will be inserted from each kubernetes object that matches the ruleset.  There is also a section called `cluster_variables` that you can use to define your own variables.  These variables can be inserted into the monitor templates.

```yaml
---
cluster_variables:
  var1: test
  var2: test2
rulesets:
- type: deployment
  match_annotations:
  - name: dd-manager/owner
    value: dd-manager
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
      tags: []
      options:
        no_data_timeframe: 60
        notify_audit: false
        notify_no_data: false
        renotify_interval: 5
        new_host_delay: 5
        evaluation_delay: 300
        timeout_h: 1
        escalation_message: ""
        thresholds:
          critical: 2
          warning: 1
          unknown: -1
          ok: 0
          critical_recovery: 0
          warning_recovery: 0
        include_tags: true
        require_full_window: true
        locked: false
```

* `cluster_variables`: (dict).  A collection of variables that can be used in monitors.  They can be used in monitors by prepending with `ClusterVariables`, eg `{{ ClusterVariables.var1 }}`.
* `rulesets`: (List).  A collection of rulesets.  A ruleset consists of a kubernetes resource type, annotations the resource must have to be considered valid, and a collection of monitors to manage for the resource.
  * `match_annotations`: (List).  A collection of name/value pairs pairs of annotations that must be present on the resource to manage it.
  * `bound_objects`: (List).  A collection of object types that are bound to this object.  For instance, if you have a ruleset for a namespace, you can bind other objects like deployments, services, etc. Then, when the bound objects in the namespace get updated, those rulesets apply to it.
  * `monitors`: (List).  A collection of monitors to manage for any resource that matches the rules defined.
    * `name`: Name of the datadog monitor.
    * `type`: The type of the monitor, chosen from:
      - `metric alert`
      - `service check`
      - `event alert`
      - `query alert`
      - `composite`
      - `log alert`
    * `query`: The monitor query to notify on.
    * `message`: A message included with in monitor notifications.
    * `tags`: A list of tags to add to your monitor.
    * `options`: A dict of options, consisting of the following:
      * `no_data_timeframe`: Number of minutes before a monitor will notify if data stops reporting.
      * `notify_audit`: boolean that indicates whether tagged users are notified if the monitor changes.
      * `notify_no_data`: boolean that indicates if the monitor notifies if data stops reporting.
      * `renotify_interval`: Number of minutes after the last notification a monitor will re-notify.
      * `new_host_delay`: Number of seconds to wait for a new host before evaluating the monitor status.
      * `evaluation_delay`: Number of seconds to delay evaluation.
      * `timeout_h`: Number of hours the before the monitor will automatically resolve if it's not reporting data.
      * `escalation_message`: Message to include with re-notifications.
      * `thresholds`: Map of thresholds for the alert.  Valid options are:
        - `ok`
        - `critical`
        - `warning`
        - `unknown`
        - `critical_recovery`
        - `warning_recovery`
      * `include_tags`: When true, notifications from this monitor automatically insert triggering tags into the title.
      * `require_full_window`: boolean indicating if a monitor needs a full window of data to be evaluated.
      * `locked`: boolean indicating if changes are only allowed from the creator or admins.

#### A Note on Templating
Since datadog uses a very similar templating language to go templating, to pass a template variable to datadog it must be "escaped" by inserting it as a template literal:

```
{{ "{{/is_alert}}" }}
```

## Contributing
PRs welcome! Check out the [Contributing Guidelines](CONTRIBUTING.md),
[Code of Conduct](CODE_OF_CONDUCT.md), and [Roadmap](ROADMAP.md) for more information.

## Further Information
A history of changes to this project can be viewed in the [Changelog](CHANGELOG.md)

If you'd like to learn more about dd-manager, or if you'd like to speak with
a Kubernetes expert, you can contact `info@reactiveops.com` or [visit our website](https://reactiveops.com)

## License
Apache License 2.0
