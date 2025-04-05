package tool

import (
	"reflect"
	"strings"
)

func ProcessParams(callback any, cb func(name string, t reflect.Type, desc string)) {
	callbackType := reflect.TypeOf(callback)

	if callbackType.Kind() == reflect.Func && callbackType.NumIn() > 0 {

		paramType := callbackType.In(0)

		if paramType.Kind() == reflect.Struct {
			for i := 0; i < paramType.NumField(); i++ {
				desc := ""
				field := paramType.Field(i)

				if d, ok := field.Tag.Lookup("description"); ok {
					desc = d
				}
				name := field.Name
				if nameTag, ok := field.Tag.Lookup("json"); ok {
					name = strings.Split(nameTag, ",")[0]
				}
				cb(name, field.Type, desc)
			}
		}
	}
}
