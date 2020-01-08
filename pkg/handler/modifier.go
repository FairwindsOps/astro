package handler

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	ddapi "github.com/zorkian/go-datadog-api"
	"k8s.io/klog"
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
func NewModifiers() *Modifiers {
	return &Modifiers{
		Items: []Modifier{
			Modifier{
				Name: "override",
				ModifyFunc: func(monitor *ddapi.Monitor, params string, val interface{}) {
					klog.Infof("Overriding monitor %s field %s", *monitor.Name, params)
					setProperty(params, monitor, val)
				},
			},
			Modifier{
				Name: "ignore",
				ModifyFunc: func(monitor *ddapi.Monitor, params string, val interface{}) {
					klog.Infof("Ignoring monitor %s", *monitor.Name)
					monitor = nil
				},
			},
		},
	}
}

// Run will run all Modifiers that match the monitor
func (m *Modifiers) Run(monitor *ddapi.Monitor, annotations map[string]string) {
	klog.Errorf("There are %d modifiers available", len(m.Items))
	for _, modifier := range m.Items {
		klog.Info("Check Modifier %s for monitor %s", modifier.Name, monitor.Name)
		if ok, match := modifier.isMatch(*monitor.Name, annotations); ok {
			klog.Errorf("Running modifier %s on monitor %s", modifier.Name, *monitor.Name)
			params := modifier.GetParams(*monitor.Name, match)
			modifier.ModifyFunc(monitor, *params, match.AnnotationValue)
		}
	}
}

// IsMatch returns true if a modifier matches the provided annotations and monitorName
func (m *Modifier) isMatch(monitorName string, annotations map[string]string) (bool, *modifierMatch) {
	re := regexp.MustCompile(fmt.Sprintf(regStr, m.Name, monitorName))

	for k, v := range annotations {
		if re.MatchString(k) {
			return true, &modifierMatch{
				AnnotationKey:   k,
				AnnotationValue: v,
			}
		}
	}
	return false, nil
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
			klog.Error("Found our field")
			klog.Errorf("Setting value to %v", val)

			v := reflect.Indirect(current)
			fmt.Printf("Kind is: %s\n", v.Kind())
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
