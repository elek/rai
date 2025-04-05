package tool

import (
	"encoding/json"
	"fmt"
	"github.com/elek/rai/schema"
	"github.com/pkg/errors"
	"reflect"
)

func HandleTools(msgs []schema.Message, tools []schema.Tool) (rmsgs []schema.Message, handled bool, err error) {
	for _, msg := range msgs {
		if msg.ToolName == "" {
			continue
		}
		rmsgs = append(msgs, msg)
		for _, t := range tools {
			if msg.ToolName == t.Name {
				callbackType := reflect.TypeOf(t.Callback)

				if callbackType.Kind() != reflect.Func || callbackType.NumIn() != 1 {
					return rmsgs, false, fmt.Errorf("callback function must have one parameter")
				}

				paramType := callbackType.In(0)
				v := reflect.New(paramType)

				err := json.Unmarshal([]byte(msg.Content), v.Interface())
				if err != nil {
					return rmsgs, false, errors.WithStack(err)
				}
				fmt.Println("Executing tool callback", t.Name, v.Elem().Interface())
				result := reflect.ValueOf(t.Callback).Call([]reflect.Value{v.Elem()})
				if len(result) > 0 {
					responseMsg := schema.Message{
						Role:    "tool",
						Content: result[0].String(),
						ToolID:  msg.ToolID,
					}
					rmsgs = append(msgs, responseMsg)
					handled = true
				}
				break
			}
		}
	}
	return rmsgs, handled, nil
}
