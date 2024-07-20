/*
Copyright 2021 Lee R. Boynton

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package smithy

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/boynton/data" //for Pretty
)

var Verbose bool

func Debug(args ...interface{}) {
	if Verbose {
		max := len(args) - 1
		for i := 0; i < max; i++ {
			fmt.Print(str(args[i]))
		}
		fmt.Println(str(args[max]))
	}
}

func pretty(arg any) string {
	return data.Pretty(arg)
}

func str(arg interface{}) string {
	return fmt.Sprintf("%v", arg)
}

func Capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}

func Uncapitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[0:1]) + s[1:]
}

func FormatComment(indent, prefix, comment string, maxcol int, extraPad bool) string {
	tab := ""
	pad := ""
	emptyPrefix := strings.Trim(prefix, " ")
	if extraPad {
		pad = tab + emptyPrefix + "\n"
	}
	left := len(indent)
	if maxcol <= left && strings.Index(comment, "\n") < 0 {
		if extraPad {
			return indent + emptyPrefix + "\n" + indent + prefix + comment + "\n" + indent + emptyPrefix + "\n"
		}
		return indent + prefix + comment + "\n"
	}
	tabbytes := make([]byte, 0, left)
	for i := 0; i < left; i++ {
		tabbytes = append(tabbytes, ' ')
	}
	tab = string(tabbytes)
	prefixlen := len(prefix)
	if strings.Index(comment, "\n") >= 0 {
		lines := strings.Split(comment, "\n")
		result := ""
		if extraPad {
			result = result + pad
		}
		for _, line := range lines {
			result = result + tab + prefix + line + "\n"
		}
		if extraPad {
			result = result + pad
		}
		return result
	}
	var buf bytes.Buffer
	col := 0
	lines := 1
	tokens := strings.Split(comment, " ")
	for _, tok := range tokens {
		toklen := len(tok)
		if col+toklen >= maxcol {
			buf.WriteString("\n")
			lines++
			col = 0
		}
		if col == 0 {
			buf.WriteString(tab)
			buf.WriteString(prefix)
			buf.WriteString(tok)
			col = left + prefixlen + toklen
		} else {
			buf.WriteString(" ")
			buf.WriteString(tok)
			col += toklen + 1
		}
	}
	buf.WriteString("\n")
	return pad + buf.String() + pad
}

func TrimSpace(s string) string {
	return TrimLeftSpace(TrimRightSpace(s))
}

func TrimRightSpace(s string) string {
	return strings.TrimRight(s, " \t\n\v\f\r")
}

func TrimLeftSpace(s string) string {
	return strings.TrimLeft(s, " \t\n\v\f\r")
}

func AsString(a any) string {
	if a == nil {
		return ""
	}
	switch v := a.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func AsFloat64(a any) float64 {
	if a == nil {
		return 0
	}
	switch v := a.(type) {
	case float64:
		return v
	default:
		return 0
	}
}

func AsInt(a any) int {
	if a == nil {
		return 0
	}
	switch v := a.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}

func AsSlice(a any) []any {
	if a == nil {
		return nil
	}
	switch v := a.(type) {
	case []any:
		return v
	}
	return nil
}

func AsMap(a any) map[string]any {
	if a == nil {
		return nil
	}
	switch v := a.(type) {
	case map[string]any:
		return v
	default:
		fmt.Println("not a map:", a)
		panic("here")
	}
}

func Get(a any, key string) any {
	m := AsMap(a)
	if a == nil {
		return nil
	}
	return m[key]
}

func GetString(a any, key string) string {
	v := Get(a, key)
	if v == nil {
		return ""
	}
	return AsString(v)
}

func GetMap(a any, key string) map[string]any {
	v := Get(a, key)
	if v == nil {
		return nil
	}
	return AsMap(v)
}

func Put(a any, key string, val any) any {
	m := AsMap(a)
	if a == nil {
		return nil
	}
	m[key] = val
	return a
}
