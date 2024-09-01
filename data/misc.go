package data

import(
	"bytes"
    "encoding/json"
	"fmt"
	"strings"
)

func Pretty(obj any) string {
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

func JsonEncode(obj any) string {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(&obj); err != nil {
		return fmt.Sprint(obj)
	}
	return strings.TrimRight(string(buf.String()), " \t\n\v\f\r")
}

func JsonDecode(j string) (*Value, error) {
	var raw Value
	//var raw any
	err := json.Unmarshal([]byte(j), &raw)
	if err != nil {
		return nil, err
	}
	return &raw, nil
}

func JsonDecodeAs[T any](j string, target *T) error {
	return json.Unmarshal([]byte(j), target)
}
