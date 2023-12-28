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
package common

import (
	"bytes"
	"strings"
)

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
	leadcol := 0
	s := FormatBlock(comment, prefix, indent, leadcol, maxcol)
	if extraPad {
		s = indent + prefix + "\n" + s + indent + prefix + "\n"
	}
	return s
}

func FormatBlock(src, prefix, indent string, leadcol, maxcol int) string {
	cols := maxcol - leadcol
	extraPad := false
	tab := ""
	pad := ""
	emptyPrefix := strings.Trim(prefix, " ")
	if extraPad {
		pad = tab + emptyPrefix + "\n"
	}
	lead := ""
	if leadcol > 0 {
		leadbytes := make([]byte, 0, leadcol)
		for i := 0; i < leadcol; i++ {
			leadbytes = append(leadbytes, ' ')
		}
		lead = string(leadbytes)
	}
	left := len(indent)
	if cols <= left && strings.Index(src, "\n") < 0 {
		if extraPad {
			return indent + emptyPrefix + "\n" + indent + prefix + src + "\n" + indent + emptyPrefix + "\n"
		}
		return indent + prefix + src + "\n"
	}
	tabbytes := make([]byte, 0, left)
	for i := 0; i < left; i++ {
		tabbytes = append(tabbytes, ' ')
	}
	tab = string(tabbytes)
	prefixlen := len(prefix)
	if strings.Index(src, "\n") >= 0 {
		lines := strings.Split(src, "\n")
		result := ""
		if extraPad {
			result = result + pad
		}
		for _, line := range lines {
			var splitlines []string
			for len(line) > maxcol {
				for i := maxcol; i >= 0; i-- {
					if len(line) <= maxcol || i == 0 {
						splitlines = append(splitlines, line)
						line = ""
						break
					}
					if line[i] == ' ' {
						splitlines = append(splitlines, line[:i])
						line = line[i+1:]
						break
					}
				}
			}
			if splitlines != nil {
				for _, l := range splitlines {
					result = result + tab + prefix + l + "\n"
				}
			} else {
				result = result + tab + prefix + line + "\n"
			}
		}
		if extraPad {
			result = result + pad
		}
		return result
	}
	var buf bytes.Buffer
	col := 0
	lines := 1
	tokens := strings.Split(src, " ")
	newline := true
	for _, tok := range tokens {
		toklen := len(tok)
		if col+toklen >= cols {
			buf.WriteString("\n")
			lines++
			buf.WriteString(lead)
			col = leadcol
			newline = true
		}
		if newline { //col == leadcol {
			newline = false
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
