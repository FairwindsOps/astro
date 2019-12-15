package handler

import (
	"reflect"

	ddapi "github.com/zorkian/go-datadog-api"
)

type modifiers struct {
	Items []modifier
}

type modifier struct {
	Name       string
	ModifyFunc func(*ddapi.Monitor, string, ...string)
}

// astro.fairwinds.com/<monitor-name>/<modifier>/<param>/<param>: value
// astro.fairwinds.com/global/<modifier>/<param>/<param>: value

func newModifiers() *modifiers {
	return &modifiers{
		Items: []modifier{
			modifier{
				Name: "override-field",
				ModifyFunc: func(monitor *ddapi.Monitor, val string, params ...string) {
					structVals := reflect.ValueOf(monitor).Elem()
					for i := 0; i < structVals.NumField(); i++ {
						if structVals.Type().Field(i).Name == params[0] {
							reflect.ValueOf(monitor).Elem().FieldByName(params[0]).SetString(val)
						}
					}
				},
			},
		},
	}
}
