//to do: move to data
package model

import(
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func Pretty(obj interface{}) string {
	indentSize := "  "
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", indentSize)
	if err := enc.Encode(&obj); err != nil {
		return fmt.Sprint(obj)
	}
	return string(buf.String())
}

func JsonEncode(obj interface{}) string {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(&obj); err != nil {
		return fmt.Sprint(obj)
	}
	return strings.TrimRight(string(buf.String()), " \t\n\v\f\r")
}

/*
   func JsonDecode(j string) Value {
	reader := &Reader{
		Input:    bufio.NewReader(strings.NewReader(j)),
		Position: 0,
	}
	v, err := reader.Read()
	if err != nil {
		return NewError(NewString(err.Error()))
	}
	return v
}
*/

func JsonDecodeAs[T any](j string, target *T) error {
	return json.Unmarshal([]byte(j), target)
}


