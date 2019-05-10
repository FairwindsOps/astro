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
| `DEFINITIONS_PATH` | The path to the monitor definitions configuration. | `N` | `conf.yml` |

### Configuration File
A configuration file is used to define your monitors.  These are organized as rulesets, which consist of the type of resource the ruleset applies to, annotations that must be present on the resource to be considered valid objects, and a set of monitors to manage for that resource.  Go templating syntax may be used in your monitors and values will be inserted from each kubernetes object that matches the ruleset.

