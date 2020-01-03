package handler

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	ddapi "github.com/zorkian/go-datadog-api"
)

type modifiers struct {
	Items []Modifier
}

// Modifier is an object that transforms fields of a monitor
type Modifier struct {
	Name       string
	ModifyFunc func(*ddapi.Monitor, string, interface{})
}

var (
	regStr string = `^%s.astro.fairwinds.com\/(global|%s)\.(?P<params>.+)$`
)

// astro.fairwinds.com/<monitor-name>/<modifier>/<param>/<param>: value
// astro.fairwinds.com/global/<modifier>/<param>/<param>: value
// standard: astro.fairwinds.com/override.dep-replica-alert.name

func newModifiers() *modifiers {
	return &modifiers{
		Items: []Modifier{
			Modifier{
				Name: "override",
				ModifyFunc: func(monitor *ddapi.Monitor, params string, val interface{}) {
					if len(params) > 0 {
						setProperty(params, monitor, val)
					}
				},
			},
			Modifier{
				Name: "ignore",
				ModifyFunc: func(monitor *ddapi.Monitor, params string, val interface{}) {
					monitor = nil
				},
			},
		},
	}
}

// IsMatch returns true if a modifier matches the provided annotations and monitorName
func (m *Modifier) IsMatch(monitorName string, annotations map[string]string) bool {
	re := regexp.MustCompile(fmt.Sprintf(regStr, m.Name, monitorName))

	for k := range annotations {
		if re.MatchString(k) {
			return true
		}
	}
	return false
}

// GetParams returns the param field of a regex string
func (m *Modifier) GetParams(monitorName string, annotationKey string) *string {
	re := regexp.MustCompile(fmt.Sprintf(regStr, m.Name, monitorName))
	if re.MatchString(annotationKey) {
		fields := re.SubexpNames()
		vals := re.FindStringSubmatch(annotationKey)
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
			switch v.Kind() {
			case reflect.Int:
				num, ok := val.(int)
				if ok {
					v.SetInt(int64(num))
				}
			case reflect.String:
				str, ok := val.(string)
				if ok {
					v.SetString(str)
				}
			case reflect.Bool:
				b, ok := val.(bool)
				if ok {
					v.SetBool(b)
				}
			}
		} else {
			parent = current
		}
	}
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
