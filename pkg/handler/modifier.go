package handler

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	ddapi "github.com/zorkian/go-datadog-api"
)

type modifiers struct {
	Items []modifier
}

// Modifier is an object that transforms fields of a monitor
type Modifier struct {
	Name       string
	ModifyFunc func(*ddapi.Monitor, string, interface{})
}

// astro.fairwinds.com/<monitor-name>/<modifier>/<param>/<param>: value
// astro.fairwinds.com/global/<modifier>/<param>/<param>: value

func newModifiers() *modifiers {
	return &modifiers{
		Items: []modifier{
			Modifier{
				Name: "override",
				ModifyFunc: func(monitor *ddapi.Monitor, param string, val interface{}) {
					setProperty(param, monitor, val)
				},
			},
		},
	}
}

// IsMatch returns true if a modifier matches the provided annotations and monitorName
func (m *Modifier) IsMatch(monitorName string, annotations map[string]string) bool {
	var re = regexp.MustCompile(fmt.Sprintf(`(?m)^astro.fairwinds.com\/(?:global|%s)\/(?P<modifier>[a-zA-Z]+)(?P<opts>\/.+)*$`, monitorName))
	for k := range annotations {
		if re.MatchString(k) {
			return true
		}
	}
	return false
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
