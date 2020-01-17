package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	ddapi "github.com/zorkian/go-datadog-api"
)

// Modifiers is a collection of Modifier objects
type Modifiers struct {
	Items []Modifier
}

// Modifier is an object that transforms fields of a monitor
type Modifier struct {
	Name       string
	ModifyFunc func(*ddapi.Monitor, string, interface{})
}

type modifierMatch struct {
	AnnotationKey   string
	AnnotationValue string
}

var (
	regStr string = `^%s.astro.fairwinds.com\/(global|%s)\.(?P<params>.+)$`
)

// NewModifiers returns a collection of available modifiers
func newModifiers() *Modifiers {
	return &Modifiers{
		Items: []Modifier{
			Modifier{
				Name: "override",
				ModifyFunc: func(monitor *ddapi.Monitor, params string, val interface{}) {
					log.Infof("Overriding monitor %s field %s", *monitor.Name, params)
					setProperty(params, monitor, val)
				},
			},
			Modifier{
				Name: "ignore",
				ModifyFunc: func(monitor *ddapi.Monitor, params string, val interface{}) {
					log.Infof("Ignoring monitor %s", *monitor.Name)
					*monitor = ddapi.Monitor{}
				},
			},
		},
	}
}

// Run will run all Modifiers that match the monitor
func (m *Modifiers) Run(monitor *ddapi.Monitor, name string, annotations map[string]string) {
	for _, modifier := range m.Items {
		log.Infof("Check Modifier %s for monitor %s", modifier.Name, *monitor.Name)
		if ok, match := modifier.isMatch(name, annotations); ok {
			for _, matchedItem := range match {
				log.Infof("Running modifier %s on monitor %s", modifier.Name, name)
				params := modifier.GetParams(name, &matchedItem)
				modifier.ModifyFunc(monitor, *params, matchedItem.AnnotationValue)
			}
		}
	}
}

// IsMatch returns true if a modifier matches the provided annotations and monitorName
func (m *Modifier) isMatch(monitorName string, annotations map[string]string) (bool, []modifierMatch) {
	rStr := fmt.Sprintf(regStr, m.Name, monitorName)
	log.Infof("Monitor regex str is: %s", rStr)
	re := regexp.MustCompile(fmt.Sprintf(regStr, m.Name, monitorName))

	matches := []modifierMatch{}
	for k, v := range annotations {
		if re.MatchString(k) {
			matches = append(matches, modifierMatch{AnnotationKey: k, AnnotationValue: v})
		}
	}
	return len(matches) > 0, matches
}

// GetParams returns the param field of a regex string
func (m *Modifier) GetParams(monitorName string, matchDetails *modifierMatch) *string {
	re := regexp.MustCompile(fmt.Sprintf(regStr, m.Name, monitorName))
	if re.MatchString(matchDetails.AnnotationKey) {
		fields := re.SubexpNames()
		vals := re.FindStringSubmatch(matchDetails.AnnotationKey)
		for i := 0; i < len(fields); i++ {
			if fields[i] == "params" {
				return &vals[i]
			}
		}
	}
	return nil
}

// setProperty sets the value field obj to value val
func setProperty(name string, obj interface{}, val interface{}) {
	parts := strings.Split(name, ".")
	parent := reflect.ValueOf(obj)
	for i, field := range parts {
		current := getReflectedField(field, parent)
		if i == len(parts)-1 {
			// reached the final object - set the value
			v := reflect.Indirect(current)
			log.Infof("Setting value to %v, kind is %s", val, v.Kind())
			switch v.Kind() {
			case reflect.Int:
				num, ok := val.(int)
				if ok {
					v.SetInt(int64(num))
				}
			case reflect.String:
				fmt.Println("Its a String")
				str, ok := val.(string)
				if ok {
					v.SetString(str)
				}
			case reflect.Bool:
				b, ok := val.(bool)
				if ok {
					v.SetBool(b)
				}
			case reflect.Ptr:
				fieldType := v.Type()
				newVal := ptr(reflect.ValueOf(val))
				s := newVal.Convert(fieldType)
				v.Set(s)
			}
		} else {
			parent = current
		}
	}
}

func ptr(v reflect.Value) reflect.Value {
	pt := reflect.PtrTo(v.Type()) // create a *T type.
	pv := reflect.New(pt.Elem())  // create a reflect.Value of type *T.
	pv.Elem().Set(v)              // sets pv to point to underlying value of v.
	return pv
}

func getReflectedField(name string, v reflect.Value) reflect.Value {
	r := v.Elem()
	log.Infof("Type is: %s", r.Type())
	log.Infof("Kind is %s", r.Kind())

	for i := 0; i < r.NumField(); i++ {
		fName := r.Type().Field(i).Name
		tags := r.Type().Field(i).Tag
		if matches(fName, name, string(tags)) {
			// it's the field we want
			v = reflect.Indirect(v)
			return v.Field(i).Addr()
		}
	}
	return reflect.Value{}
}

// returns true if fieldName either matches the name of the field of the json/yaml tags match
func matches(fieldName string, desiredName string, tags string) bool {
	if strings.ToLower(fieldName) == strings.ToLower(desiredName) {
		return true
	}

	// check json/yaml field name
	var re = regexp.MustCompile(`(?m)(json|yaml):\"[a-zA-Z]+(,.+|\")`)
	return re.MatchString(desiredName)
}

/*
// TODO : this error when adding this:
kubectl annotate deploy/coredns override.astro.fairwinds.com/deploy-replica-alert.options.escalation_message="HelloWorld"

INFO[0078] Overriding monitor IOverrideYou field options.escalation_message
E0115 11:53:54.032175   76001 runtime.go:69] Observed a panic: &reflect.ValueError{Method:"reflect.Value.NumField", Kind:0x16} (reflect: call of reflect.Value.NumField on ptr Value)
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/runtime/runtime.go:76
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/runtime/runtime.go:65
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/runtime/runtime.go:51
/Users/huberm/.asdf/installs/golang/1.13.4/go/src/runtime/panic.go:679
/Users/huberm/.asdf/installs/golang/1.13.4/go/src/reflect/value.go:208
/Users/huberm/.asdf/installs/golang/1.13.4/go/src/reflect/value.go:1356
/Users/huberm/code/github.com/fairwinds/astro/pkg/config/modifier.go:148
/Users/huberm/code/github.com/fairwinds/astro/pkg/config/modifier.go:104
/Users/huberm/code/github.com/fairwinds/astro/pkg/config/modifier.go:41
/Users/huberm/code/github.com/fairwinds/astro/pkg/config/modifier.go:63
/Users/huberm/code/github.com/fairwinds/astro/pkg/config/config.go:108
/Users/huberm/code/github.com/fairwinds/astro/pkg/config/config.go:76
/Users/huberm/code/github.com/fairwinds/astro/pkg/handler/deployments.go:55
/Users/huberm/code/github.com/fairwinds/astro/pkg/handler/handler.go:44
/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:89
/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:101
/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:66
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/wait/wait.go:152
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/wait/wait.go:153
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/wait/wait.go:88
/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:61
/Users/huberm/.asdf/installs/golang/1.13.4/go/src/runtime/asm_amd64.s:1357
E0115 11:53:54.033778   76001 runtime.go:69] Observed a panic: &reflect.ValueError{Method:"reflect.Value.NumField", Kind:0x16} (reflect: call of reflect.Value.NumField on ptr Value)
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/runtime/runtime.go:76
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/runtime/runtime.go:65
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/runtime/runtime.go:51
/Users/huberm/.asdf/installs/golang/1.13.4/go/src/runtime/panic.go:679
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/runtime/runtime.go:58
/Users/huberm/.asdf/installs/golang/1.13.4/go/src/runtime/panic.go:679
/Users/huberm/.asdf/installs/golang/1.13.4/go/src/reflect/value.go:208
/Users/huberm/.asdf/installs/golang/1.13.4/go/src/reflect/value.go:1356
/Users/huberm/code/github.com/fairwinds/astro/pkg/config/modifier.go:148
/Users/huberm/code/github.com/fairwinds/astro/pkg/config/modifier.go:104
/Users/huberm/code/github.com/fairwinds/astro/pkg/config/modifier.go:41
/Users/huberm/code/github.com/fairwinds/astro/pkg/config/modifier.go:63
/Users/huberm/code/github.com/fairwinds/astro/pkg/config/config.go:108
/Users/huberm/code/github.com/fairwinds/astro/pkg/config/config.go:76
/Users/huberm/code/github.com/fairwinds/astro/pkg/handler/deployments.go:55
/Users/huberm/code/github.com/fairwinds/astro/pkg/handler/handler.go:44
/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:89
/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:101
/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:66
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/wait/wait.go:152
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/wait/wait.go:153
/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/wait/wait.go:88
/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:61
/Users/huberm/.asdf/installs/golang/1.13.4/go/src/runtime/asm_amd64.s:1357
panic: reflect: call of reflect.Value.NumField on ptr Value [recovered]
	panic: reflect: call of reflect.Value.NumField on ptr Value [recovered]
	panic: reflect: call of reflect.Value.NumField on ptr Value

goroutine 71 [running]:
k8s.io/apimachinery/pkg/util/runtime.HandleCrash(0x0, 0x0, 0x0)
	/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/runtime/runtime.go:58 +0x105
panic(0x1e612e0, 0xc0002e3800)
	/Users/huberm/.asdf/installs/golang/1.13.4/go/src/runtime/panic.go:679 +0x1b2
k8s.io/apimachinery/pkg/util/runtime.HandleCrash(0x0, 0x0, 0x0)
	/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/runtime/runtime.go:58 +0x105
panic(0x1e612e0, 0xc0002e3800)
	/Users/huberm/.asdf/installs/golang/1.13.4/go/src/runtime/panic.go:679 +0x1b2
reflect.flag.mustBe(...)
	/Users/huberm/.asdf/installs/golang/1.13.4/go/src/reflect/value.go:208
reflect.Value.NumField(0x1ffb340, 0xc00029a6e8, 0x196, 0x1ffb340)
	/Users/huberm/.asdf/installs/golang/1.13.4/go/src/reflect/value.go:1356 +0xbe
github.com/fairwindsops/astro/pkg/config.getReflectedField(0xc00074e3fa, 0x12, 0xc00035d380, 0xc00029a6e8, 0x16, 0xc00035d380, 0xc00029a6e8, 0x16)
	/Users/huberm/code/github.com/fairwinds/astro/pkg/config/modifier.go:148 +0xc1
github.com/fairwindsops/astro/pkg/config.setProperty(0xc00074e3f2, 0x1a, 0x1fd3820, 0xc00029a690, 0x1e0fa20, 0xc0007aef50)
	/Users/huberm/code/github.com/fairwinds/astro/pkg/config/modifier.go:104 +0x169
github.com/fairwindsops/astro/pkg/config.newModifiers.func1(0xc00029a690, 0xc00074e3f2, 0x1a, 0x1e0fa20, 0xc0007aef50)
	/Users/huberm/code/github.com/fairwinds/astro/pkg/config/modifier.go:41 +0x12c
github.com/fairwindsops/astro/pkg/config.(*Modifiers).Run(0xc0002e3100, 0xc00029a690, 0xc0007014c0, 0x14, 0xc00026b6b0)
	/Users/huberm/code/github.com/fairwinds/astro/pkg/config/modifier.go:63 +0x397
github.com/fairwindsops/astro/pkg/config.(*Config).getMatchingRulesets(0xc00024c620, 0xc00026b6b0, 0x200b7b8, 0xa, 0xc0008f4000)
	/Users/huberm/code/github.com/fairwinds/astro/pkg/config/config.go:108 +0x332
github.com/fairwindsops/astro/pkg/config.(*Config).GetMatchingMonitors(0xc00024c620, 0xc00026b6b0, 0x200b7b8, 0xa, 0x0)
	/Users/huberm/code/github.com/fairwinds/astro/pkg/config/config.go:76 +0x84
github.com/fairwindsops/astro/pkg/handler.OnDeploymentChanged(0xc0000dad80, 0xc0007a9820, 0x13, 0x2008e6e, 0x6, 0xc0006d87e0, 0xb, 0x200b7b8, 0xa)
	/Users/huberm/code/github.com/fairwinds/astro/pkg/handler/deployments.go:55 +0x272
github.com/fairwindsops/astro/pkg/handler.OnUpdate(0x1fe3e20, 0xc0000dad80, 0xc0007a9820, 0x13, 0x2008e6e, 0x6, 0xc0006d87e0, 0xb, 0x200b7b8, 0xa)
	/Users/huberm/code/github.com/fairwinds/astro/pkg/handler/handler.go:44 +0x19a
github.com/fairwindsops/astro/pkg/controller.(*KubeResourceWatcher).process(0xc0003b6930, 0xc0007a9820, 0x13, 0x2008e6e, 0x6, 0xc0006d87e0, 0xb, 0x200b7b8, 0xa, 0x0, ...)
	/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:89 +0xcf
github.com/fairwindsops/astro/pkg/controller.(*KubeResourceWatcher).next(0xc0003b6930, 0xc0003a3e00)
	/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:101 +0x17e
github.com/fairwindsops/astro/pkg/controller.(*KubeResourceWatcher).waitForEvents(0xc0003b6930)
	/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:66 +0x2b
k8s.io/apimachinery/pkg/util/wait.JitterUntil.func1(0xc0003a3fa0)
	/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/wait/wait.go:152 +0x5e
k8s.io/apimachinery/pkg/util/wait.JitterUntil(0xc000913fa0, 0x3b9aca00, 0x0, 0x1, 0xc000246240)
	/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/wait/wait.go:153 +0xf8
k8s.io/apimachinery/pkg/util/wait.Until(...)
	/Users/huberm/go/pkg/mod/k8s.io/apimachinery@v0.0.0-20190404173353-6a84e37a896d/pkg/util/wait/wait.go:88
github.com/fairwindsops/astro/pkg/controller.(*KubeResourceWatcher).Watch(0xc0003b6930, 0xc000246240)
	/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:61 +0x248
created by github.com/fairwindsops/astro/pkg/controller.New
	/Users/huberm/code/github.com/fairwinds/astro/pkg/controller/controller.go:138 +0x2d0
exit status 2
*/
