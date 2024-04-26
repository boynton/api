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
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/boynton/api/model"
	"github.com/boynton/data"
)

const IndentAmount = "    "

// ASTs don't have a preferred namespace, but IDL files do. When going back to IDL, getting the preferred namespace is desirable.
// The algorithm here is to prefer the first service's namespace, if present, or the first non-smithy, non-aws namespace encountered.
func (ast *AST) NamespaceAndServiceVersion() (string, string, string) {
	var namespace, name, version string
	for _, k := range ast.Shapes.Keys() {
		v := ast.GetShape(k)
		if strings.HasPrefix(k, "smithy.") || strings.HasPrefix(k, "aws.") {
			continue
		}
		i := strings.Index(k, "#")
		if i >= 0 {
			namespace = k[:i]
		}
		if v.Type == "service" {
			version = v.Version
			name = k[i+1:]
			break
		}
	}
	return namespace, name, version
}

func (ast *AST) IDLForOperationShape(shapeId string, decorator *model.Decorator) string {
	shape := ast.GetShape(shapeId)
	w := &IdlWriter{
		ast:       ast,
		namespace: shapeIdNamespace(shapeId),
		version:   ast.AssemblyVersion(),
		decorator: decorator,
	}
	w.Begin()
	emitted := make(map[string]bool, 0)
	w.EmitOperationShape(shapeId, shape, emitted)
	return w.End()
}

func (ast *AST) IDLForTypeShape(shapeId string, decorator *model.Decorator) string {
	shape := ast.GetShape(shapeId)
	w := &IdlWriter{
		ast:       ast,
		namespace: shapeIdNamespace(shapeId),
		version:   ast.AssemblyVersion(),
		decorator: decorator,
	}
	w.Begin()
	w.EmitShape(shapeId, shape)
	return w.End()
}

func (ast *AST) IDLForResourceShape(shapeId string, decorator *model.Decorator) string {
	shape := ast.GetShape(shapeId)
	w := &IdlWriter{
		ast:       ast,
		namespace: shapeIdNamespace(shapeId),
		version:   ast.AssemblyVersion(),
		decorator: decorator,
	}
	w.Begin()
	w.EmitResourceShape(shapeId, shape)
	return w.End()
}

// Generate Smithy IDL to describe the Smithy model for a specified namespace
func (ast *AST) IDL(ns string) string {
	w := &IdlWriter{
		ast:       ast,
		namespace: ns,
		version:   ast.AssemblyVersion(),
	}

	w.Begin()
	w.Emit("$version: \"%d\"\n", w.version)
	emitted := make(map[string]bool, 0)

	if ast.Metadata != nil && ast.Metadata.Length() > 0 {
		w.Emit("\n")
		for _, k := range ast.Metadata.Keys() {
			v := ast.Metadata.Get(k)
			w.Emit("metadata %s = %s", k, data.Pretty(v))
		}
	}
	w.Emit("\nnamespace %s\n", ns)

	imports := ast.ExternalRefs(ns)
	if len(imports) > 0 {
		w.Emit("\n")
		for _, im := range imports {
			w.Emit("use %s\n", im)
		}
	}

	for _, nsk := range ast.Shapes.Keys() {
		shape := ast.GetShape(nsk)
		shapeAbsName := strings.Split(nsk, "#")
		shapeNs := shapeAbsName[0]
		shapeName := shapeAbsName[1]
		if shapeNs == ns {
			if shape.Type == "service" {
				w.Emit("\n")
				w.EmitServiceShape(shapeName, shape)
				break
			}
		}
	}
	for _, nsk := range ast.Shapes.Keys() {
		lst := strings.Split(nsk, "#")
		if lst[0] == ns {
			shape := ast.GetShape(nsk)
			k := lst[1]
			if shape.Type == "resource" {
				w.Emit("\n")
				w.EmitResourceShape(k, shape)
			}
		}
	}
	for _, nsk := range ast.Shapes.Keys() {
		lst := strings.Split(nsk, "#")
		if lst[0] == ns {
			shape := ast.GetShape(nsk)
			k := lst[1]
			if shape.Type == "operation" {
				w.Emit("\n")
				w.EmitOperationShape(k, shape, emitted)
			}
		}
	}
	for _, nsk := range ast.Shapes.Keys() {
		lst := strings.Split(nsk, "#")
		k := lst[1]
		if lst[0] == ns {
			if !emitted[k] {
				w.EmitShape(k, ast.GetShape(nsk))
			}
		}
	}
	for _, nsk := range ast.Shapes.Keys() {
		shape := ast.GetShape(nsk)
		if shape.Type == "operation" {
			lst := strings.Split(nsk, "#")
			if lst[0] == ns {
				if d := shape.Traits.Get("smithy.api#examples"); d != nil {
					if d.IsObject() {
						w.EmitExamplesTrait(nsk, d)
					}
				}
			}
		}
	}
	return w.End()
}

func (ast *AST) ExternalRefs(ns string) []string {
	match := ns + "#"
	if ns == "" {
		match = ""
	}
	refs := make(map[string]bool, 0)
	for _, k := range ast.Shapes.Keys() {
		lst := strings.Split(k, "#")
		if ns == "" || lst[0] == ns {
			v := ast.GetShape(k)
			ast.noteExternalRefs(match, k, v, refs)
		}
	}
	var res []string
	for k := range refs {
		res = append(res, k)
	}
	return res
}

func (ast *AST) noteExternalTraitRefs(match string, traits *NodeValue, refs map[string]bool) {
	if traits != nil {
		for _, tk := range traits.Keys() {
			if !strings.HasPrefix(tk, "smithy.api#") && (match != "" && !strings.HasPrefix(tk, match)) {
				refs[tk] = true
			}
		}
	}
}

func (ast *AST) noteExternalRefs(match string, name string, shape *Shape, refs map[string]bool) {
	if name == "smithy.api#Document" {
		//force an alias to this to get emitted.
	} else if strings.HasPrefix(name, "smithy.api#") {
		return
	}
	if _, ok := refs[name]; ok {
		return
	}
	if match == "" || !strings.HasPrefix(name, match) {
		refs[name] = true
		if shape != nil {
			ast.noteExternalTraitRefs(match, shape.Traits, refs)
			switch shape.Type {
			case "map":
				ast.noteExternalRefs(match, shape.Key.Target, ast.GetShape(shape.Key.Target), refs)
				ast.noteExternalTraitRefs(match, shape.Key.Traits, refs)
				ast.noteExternalRefs(match, shape.Value.Target, ast.GetShape(shape.Value.Target), refs)
				ast.noteExternalTraitRefs(match, shape.Value.Traits, refs)
			case "list", "set":
				ast.noteExternalRefs(match, shape.Member.Target, ast.GetShape(shape.Member.Target), refs)
				ast.noteExternalTraitRefs(match, shape.Member.Traits, refs)
			case "structure", "union":
				if shape.Members != nil {
					for _, k := range shape.Members.Keys() {
						member := shape.Members.Get(k)
						ast.noteExternalRefs(match, member.Target, ast.GetShape(member.Target), refs)
						ast.noteExternalTraitRefs(match, member.Traits, refs)
					}
				}
			}
		}
	}
}

type IdlWriter struct {
	buf       bytes.Buffer
	writer    *bufio.Writer
	namespace string
	name      string
	version   int
	ast       *AST
	decorator *model.Decorator
}

func (w *IdlWriter) Begin() {
	w.buf.Reset()
	w.writer = bufio.NewWriter(&w.buf)
}

func (w *IdlWriter) stripNamespace(id string) string {
	n := strings.Index(id, "#")
	if n < 0 {
		return id
	}
	return id[n+1:]
	/*
		match := w.namespace + "#"
		if strings.HasPrefix(id, match) {
			return id[len(match):]
		}
		if strings.HasPrefix(id, "smithy.api") {
			n := strings.Index(id, "#")
			if n >= 0 {
				return id[n+1:]
			}
		}
		return id
	*/
}

func (w *IdlWriter) Emit(format string, args ...interface{}) {
	w.writer.WriteString(fmt.Sprintf(format, args...))
}

func (w *IdlWriter) EmitShape(name string, shape *Shape) {
	s := strings.ToLower(shape.Type)
	w.Emit("\n")
	switch s {
	case "boolean":
		w.EmitBooleanShape(name, shape)
	case "byte", "short", "integer", "long", "float", "double", "biginteger", "bigdecimal":
		w.EmitNumericShape(shape.Type, name, shape)
	case "blob":
		w.EmitBlobShape(name, shape)
	case "string":
		w.EmitStringShape(name, shape)
	case "timestamp":
		w.EmitTimestampShape(name, shape)
	case "list", "set":
		w.EmitCollectionShape(shape.Type, name, shape)
	case "map":
		w.EmitMapShape(name, shape)
	case "structure":
		w.EmitStructureShape(name, shape)
	case "union":
		w.EmitUnionShape(name, shape)
	case "enum", "intenum":
		w.EmitEnumShape(shape.Type, name, shape)
	case "resource":
		// already emitted
		//w.EmitResourceShape(name, shape)
	case "operation", "service":
		// already emitted
	default:
		panic("fix: shape " + name + " of type " + data.Pretty(shape))
	}
}

func (w *IdlWriter) EmitDocumentation(doc, indent string) {
	if doc != "" {
		s := FormatComment(indent, "/// ", doc, 128, false)
		w.Emit(s)
		//		w.Emit("%s@documentation(%q)\n", indent, doc)
	}
}

func (w *IdlWriter) EmitBooleanTrait(b bool, tname, indent string) {
	if b {
		w.Emit("%s@%s\n", indent, tname)
	}
}

func (w *IdlWriter) EmitStringTrait(v, tname, indent string) {
	if v != "" {
		if v == "-" { //hack
			w.Emit("%s@%s\n", indent, tname)
		} else {
			w.Emit("%s@%s(%q)\n", indent, tname, v)
		}
	}
}

func (w *IdlWriter) EmitLengthTrait(v interface{}, indent string) {
	if nv, ok := v.(*NodeValue); ok {
		min := nv.Get("min")
		max := nv.Get("max")
		if min != nil && max != nil {
			w.Emit("%s@length(min: %d, max: %d)\n", indent, data.AsInt(min), data.AsInt(max))
		} else if max != nil {
			w.Emit("%s@length(max: %d)\n", indent, data.AsInt(max))
		} else if min != nil {
			w.Emit("%s@length(min: %d)\n", indent, data.AsInt(min))
		}
	}
}

func (w *IdlWriter) EmitRangeTrait(v interface{}, indent string) {
	if r, ok := v.(*NodeValue); ok {
		min := r.Get("min")
		max := r.Get("max")
		if min != nil && max != nil {
			w.Emit("@range(min: %v, max: %v)\n", data.AsDecimal(min), data.AsDecimal(max))
		} else if max != nil {
			w.Emit("@range(max: %v)\n", data.AsDecimal(max))
		} else if min != nil {
			w.Emit("@range(min: %v)\n", data.AsDecimal(min))
		}
	}
}

func (w *IdlWriter) EmitTraitTrait(v interface{}) {
	l := data.AsMap(v)
	if l != nil {
		var lst []string
		selector := data.GetString(l, "selector")
		if selector != "" {
			lst = append(lst, fmt.Sprintf("selector: %q", selector))
		}
		conflicts := data.GetStringSlice(l, "conflicts")
		if conflicts != nil {
			s := "["
			for _, e := range conflicts {
				if s != "[" {
					s = s + ", "
				}
				s = s + e
			}
			s = s + "]"
			lst = append(lst, fmt.Sprintf("conflicts: %s", s))
		}
		structurallyExclusive := data.GetString(l, "structurallyExclusive")
		if structurallyExclusive != "" {
			lst = append(lst, fmt.Sprintf("selector: %q", structurallyExclusive))
		}
		if len(lst) > 0 {
			w.Emit("@trait(%s)\n", strings.Join(lst, ", "))
			return
		}
	}
	w.Emit("@trait\n")
}

func (w *IdlWriter) EmitTagsTrait(v interface{}, indent string) {
	if sa, ok := v.([]string); ok {
		w.Emit("@tags(%v)\n", listOfStrings("", "%q", sa))
	}
}

func (w *IdlWriter) EmitDeprecatedTrait(dep *NodeValue, indent string) {
	if dep != nil && dep.value != nil {
		s := indent + "@deprecated"
		hasMessage := false
		if dep.Has("message") {
			s = s + fmt.Sprintf("(message: %q", dep.GetString("message"))
			hasMessage = true
		}
		if dep.Has("since") {
			if hasMessage {
				s = s + fmt.Sprintf(", since: %q)", dep.GetString("since"))
			} else {
				s = s + fmt.Sprintf("(since: %q)", dep.GetString("since"))
			}
		} else {
			s = s + ")"
		}
		w.Emit(s + "\n")
	}
}

func (w *IdlWriter) EmitHttpTrait(rv interface{}, indent string) {
	var method, uri string
	code := 0
	switch v := rv.(type) {
	case map[string]interface{}:
		method = data.GetString(v, "method")
		uri = data.GetString(v, "uri")
		code = data.GetInt(v, "code")
	case *NodeValue:
		method = v.GetString("method")
		uri = v.GetString("uri")
		code = v.GetInt("code", 0)
	default:
		panic("What?!")
	}
	s := fmt.Sprintf("method: %q, uri: %q", method, uri)
	if code != 0 {
		s = s + fmt.Sprintf(", code: %d", code)
	}
	w.Emit("@http(%s)\n", s)
}

func (w *IdlWriter) EmitHttpErrorTrait(rv interface{}, indent string) {
	w.Emit("@httpError(%v)\n", rv)
}

func (w *IdlWriter) EmitSimpleShape(shapeName, name string, shape *Shape) {
	w.Emit("%s %s%s\n", shapeName, w.stripNamespace(name), w.withMixins(shape.Mixins))
}

func (w *IdlWriter) EmitBooleanShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.EmitSimpleShape("boolean", w.stripNamespace(name), shape)
}

func (w *IdlWriter) EmitNumericShape(shapeName, name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.EmitSimpleShape(shapeName, w.stripNamespace(name), shape)
}

func (w *IdlWriter) EmitStringShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.EmitSimpleShape(shape.Type, w.stripNamespace(name), shape)
}

func (w *IdlWriter) forResource(rezName string) string {
	return ""
}

func (w *IdlWriter) withMixins(mixins []*ShapeRef) string {
	if len(mixins) > 0 {
		var mixinNames []string
		for _, ref := range mixins {
			mixinNames = append(mixinNames, w.stripNamespace(ref.Target))
		}
		return fmt.Sprintf(" with [%s]", strings.Join(mixinNames, ", "))
	}
	return ""
}

func (w *IdlWriter) EmitTimestampShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("timestamp %s%s\n", w.stripNamespace(name), w.withMixins(shape.Mixins))
}

func (w *IdlWriter) EmitBlobShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("blob %s%s\n", w.stripNamespace(name), w.withMixins(shape.Mixins))
}

func (w *IdlWriter) EmitCollectionShape(shapeName, name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("%s %s%s {\n", shapeName, w.stripNamespace(name), w.withMixins(shape.Mixins))
	tname := w.decorate(w.stripNamespace(shape.Member.Target))
	w.Emit("    member: %s\n", tname)
	w.Emit("}\n")
}

func (w *IdlWriter) EmitMapShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("map %s%s {\n    key: %s,\n    value: %s\n}\n", w.stripNamespace(name), w.withMixins(shape.Mixins), w.stripNamespace(shape.Key.Target), w.stripNamespace(shape.Value.Target))
}

func (w *IdlWriter) EmitUnionShape(name string, shape *Shape) {
	comma := ""
	if w.version < 2 {
		comma = ","
	}
	w.EmitTraits(shape.Traits, "")
	w.Emit("union %s%s {\n", w.stripNamespace(name), w.withMixins(shape.Mixins))
	for i, k := range shape.Members.Keys() {
		if i > 0 {
			w.Emit("\n")
		}
		v := shape.Members.Get(k)
		w.EmitTraits(v.Traits, IndentAmount)
		tname := w.decorate(w.stripNamespace(v.Target))
		w.Emit("%s%s: %s%s\n", IndentAmount, k, tname, comma)
	}
	w.Emit("}\n")
}

func (w *IdlWriter) EmitEnumShape(enumType string, name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("enum %s%s {\n", w.stripNamespace(name), w.withMixins(shape.Mixins))
	count := shape.Members.Length()
	for _, fname := range shape.Members.Keys() {
		mem := shape.Members.Get(fname)
		eqval := ""
		if val := mem.Traits.Get("smithy.api#enumValue"); val != nil {
			if enumType == "intEnum" {
				dval := data.AsInt(val)
				eqval = fmt.Sprintf(" = %d", dval)
			} else {
				sval := fmt.Sprintf("%s", val) //data.AsString(val)
				if sval != fname {
					eqval = fmt.Sprintf(" = %q", sval)
				}
			}
		}
		w.EmitTraits(mem.Traits, IndentAmount)
		w.Emit("%s%s%s", IndentAmount, fname, eqval)
		count--
		if count > 0 {
			w.Emit(",\n")
		} else {
			w.Emit("\n")
		}
	}
	w.Emit("}\n")
}

func (w *IdlWriter) EmitTraits(traits *NodeValue, indent string) {
	//note: @documentation is an alternate for ("///"+comment), but then must be before other traits.
	if traits == nil {
		return
	}
	for _, k := range traits.Keys() {
		v := traits.Get(k)
		switch k {
		case "smithy.api#documentation":
			w.EmitDocumentation(v.AsString(), indent)
		}
	}
	for _, k := range traits.Keys() {
		v := traits.Get(k)
		switch k {
		case "smithy.api#documentation", "smithy.api#examples", "smithy.api#enumValue":
			//do nothing, handled elsewhere
		case "smithy.api#sensitive", "smithy.api#required", "smithy.api#readonly", "smithy.api#idempotent":
			w.EmitBooleanTrait(v.AsBool(), w.stripNamespace(k), indent)
		case "smithy.api#httpLabel", "smithy.api#httpPayload":
			w.EmitBooleanTrait(v.AsBool(), w.stripNamespace(k), indent)
		case "smithy.api#httpQuery", "smithy.api#httpHeader", "smithy.api#timestampFormat":
			w.EmitStringTrait(v.AsString(), w.stripNamespace(k), indent)
		case "smithy.api#deprecated":
			w.EmitDeprecatedTrait(v, indent)
		case "smithy.api#http", "smithy.api#httpError":
			/* emit nothing here, handled in subsequent pass */
		case "smithy.api#length":
			w.EmitLengthTrait(v, indent)
		case "smithy.api#range":
			w.EmitRangeTrait(v, indent)
		case "smithy.api#tags":
			w.EmitTagsTrait(v, indent)
		case "smithy.api#pattern", "smithy.api#error":
			w.EmitStringTrait(v.AsString(), w.stripNamespace(k), indent)
		case "aws.protocols#restJson1":
			w.Emit("%s@%s\n", indent, k) //FIXME for the non-default attributes
		case "smithy.api#paginated":
			w.EmitPaginatedTrait(v)
		case "smithy.api#trait":
			w.EmitTraitTrait(v)
		case "smithy.api#default":
			//fmt.Println("FIX ME: emit defaut trait")
		default:
			w.EmitCustomTrait(k, v, indent)
		}
	}
	for _, k := range traits.Keys() {
		v := traits.Get(k)
		switch k {
		case "smithy.api#http":
			w.EmitHttpTrait(v, indent)
		case "smithy.api#httpError":
			w.EmitHttpErrorTrait(v, indent)
		}
	}
}

func (w *IdlWriter) EmitCustomTrait(k string, v interface{}, indent string) {
	args := ""
	if m, ok := v.(*NodeValue); ok {
		if m.Length() > 0 {
			var lst []string
			for _, ak := range m.Keys() {
				av := m.Get(ak)
				lst = append(lst, fmt.Sprintf("%s: %s", ak, data.JsonEncode(av)))
			}
			args = "(\n    " + strings.Join(lst, ",\n    ") + ")"
		}
	}
	w.Emit("%s@%s%s\n", indent, w.stripNamespace(k), args)
}

func (w *IdlWriter) EmitPaginatedTrait(d interface{}) {
	if m, ok := d.(map[string]interface{}); ok {
		var args []string
		for k, v := range m {
			args = append(args, fmt.Sprintf("%s: %q", k, v))
		}
		if len(args) > 0 {
			w.Emit("@paginated(" + strings.Join(args, ", ") + ")\n")
		}
	}
}

func (w *IdlWriter) EmitExamplesTrait(opname string, raw interface{}) {
	switch dat := raw.(type) {
	case []map[string]interface{}:
		target := w.stripNamespace(opname)
		formatted := data.Pretty(dat)
		if strings.HasSuffix(formatted, "\n") {
			formatted = formatted[:len(formatted)-1]
		}
		w.Emit("apply "+w.stripNamespace(target)+" @examples(%s)\n", formatted)
	default:
		panic("FIX ME!")
	}
}

func (w *IdlWriter) EmitStructureShape(name string, shape *Shape) {
	comma := ""
	if w.version < 2 {
		comma = ","
	}
	w.EmitTraits(shape.Traits, "")
	w.Emit("structure %s%s {\n", w.stripNamespace(name), w.withMixins(shape.Mixins))
	for i, k := range shape.Members.Keys() {
		if i > 0 {
			w.Emit("\n")
		}
		v := shape.Members.Get(k)
		w.EmitTraits(v.Traits, IndentAmount)
		tname := w.decorate(w.stripNamespace(v.Target))
		w.Emit("%s%s: %s%s\n", IndentAmount, k, tname, comma)
	}
	w.Emit("}\n")
}

func (w *IdlWriter) listOfShapeRefs(label string, format string, lst []*ShapeRef, absolute bool) string {
	multiline := true
	indent := "    "
	indentAmount := "    "
	s := ""
	if len(lst) > 0 {
		s = label + ": ["
		if multiline {
			s = s + "\n" + indent + indentAmount
		}
		for n, a := range lst {
			if n > 0 {
				if multiline {
					s = s + "\n" + indent + indentAmount
				} else {
					s = s + ", "
				}
			}
			target := a.Target
			if !absolute {
				target = w.stripNamespace(target)
			}
			tname := w.decorate(target)
			s = s + fmt.Sprintf(format, tname)
		}
		if multiline {
			s = s + "\n" + indentAmount
		}
		s = s + "]"
	}
	return s
}

func listOfStrings(label string, format string, lst []string) string {
	s := ""
	if len(lst) > 0 {
		if label != "" {
			s = label + ": "
		}
		s = s + "["
		for n, a := range lst {
			if n > 0 {
				s = s + ", "
			}
			s = s + fmt.Sprintf(format, a)
		}
		s = s + "]"
	}
	return s
}

func (w *IdlWriter) EmitServiceShape(name string, shape *Shape) {
	comma := ""
	if w.version < 2 {
		comma = ","
	}
	w.EmitTraits(shape.Traits, "")
	w.Emit("service %s%s {\n", name, w.withMixins(shape.Mixins))
	if shape.Version != "" {
		w.Emit("    version: %q%s\n", shape.Version, comma)
	}
	if len(shape.Operations) > 0 {
		w.Emit("    %s\n", w.listOfShapeRefs("operations", "%s", shape.Operations, false))
	}
	if len(shape.Resources) > 0 {
		w.Emit("    %s\n", w.listOfShapeRefs("resources", "%s", shape.Resources, false))
	}
	w.Emit("}\n")
}

func (w *IdlWriter) EmitResourceShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("resource %s%s {\n", w.stripNamespace(name), w.withMixins(shape.Mixins))
	if shape.Identifiers.Length() > 0 {
		w.Emit("    identifiers: { ")
		for i, k := range shape.Identifiers.Keys() {
			v := shape.Identifiers.Get(k)
			s := ""
			if i > 0 {
				s = ", "
			}
			w.Emit("%s%s: %s", s, k, w.decorate(w.stripNamespace(v.Target)))
		}
		w.Emit(" }\n")
	}
	if shape.Create != nil {
		tname := w.decorate(w.stripNamespace(shape.Create.Target))
		w.Emit("    create: %v\n", tname)
	}
	if shape.Put != nil {
		tname := w.decorate(w.stripNamespace(shape.Put.Target))
		w.Emit("    put: %v\n", tname)
	}
	if shape.Read != nil {
		tname := w.decorate(w.stripNamespace(shape.Read.Target))
		w.Emit("    read: %v\n", tname)
	}
	if shape.Update != nil {
		tname := w.decorate(w.stripNamespace(shape.Update.Target))
		w.Emit("    update: %v\n", tname)
	}
	if shape.Delete != nil {
		tname := w.decorate(w.stripNamespace(shape.Delete.Target))
		w.Emit("    delete: %v\n", tname)
	}
	if shape.List != nil {
		tname := w.decorate(w.stripNamespace(shape.List.Target))
		w.Emit("    list: %v\n", tname)
	}
	if len(shape.Operations) > 0 {
		var tmp []*ShapeRef
		for _, id := range shape.Operations {
			tmp = append(tmp, &ShapeRef{Target: w.stripNamespace(id.Target)})
		}
		w.Emit("    %s\n", w.listOfShapeRefs("operations", "%s", tmp, true))
	}
	if len(shape.CollectionOperations) > 0 {
		w.Emit("    %s\n", w.listOfShapeRefs("collectionOperations", "%s", shape.CollectionOperations, true))
	}
	w.Emit("}\n")
}

func (w *IdlWriter) decorate(tname string) string {
	if w.decorator != nil {
		switch tname {
		case "Boolean", "String", "Blob", "Timestamp", "Byte", "Short", "Integer", "Long", "Float", "Double", "BigDecimal", "BigInteger":
			return w.decorator.BaseType(tname)
		}
		return w.decorator.UserType(tname)
	}
	return tname
}

func (w *IdlWriter) EmitOperationShape(name string, shape *Shape, emitted map[string]bool) {
	var inputShape, outputShape *Shape
	var inputName, outputName string
	var inputEmitted, outputEmitted bool
	if shape.Input != nil {
		inputName = w.stripNamespace(shape.Input.Target)
		inputShape = w.ast.GetShape(shape.Input.Target)
	}
	if shape.Output != nil {
		outputName = w.stripNamespace(shape.Output.Target)
		outputShape = w.ast.GetShape(shape.Output.Target)
	}
	w.EmitTraits(shape.Traits, "")
	w.Emit("operation %s%s {\n", StripNamespace(name), w.withMixins(shape.Mixins))
	if w.version == 2 {
		if inputShape != nil {
			if b := inputShape.Traits.Get("smithy.api#input"); b != nil {
				inputTraits := "" //?
				inputMixins := w.withMixins(inputShape.Mixins)
				inputResource := "" //w.forResource(inputShape.Resource)
				w.Emit("%sinput := %s%s%s{\n", IndentAmount, inputTraits, inputMixins, inputResource)
				i2 := IndentAmount + IndentAmount
				for i, k := range inputShape.Members.Keys() {
					if i > 0 {
						w.Emit("\n")
					}
					v := inputShape.Members.Get(k)
					w.EmitTraits(v.Traits, i2)
					tname := w.decorate(w.stripNamespace(v.Target))
					s := ""
					if v.Traits.Has("smithy.api#default") {
						s = " = " + data.JsonEncode(v.Traits.Get("smithy.api#default"))
					}
					w.Emit("%s%s: %s%s\n", i2, k, tname, s)
				}
				w.Emit("%s}\n", IndentAmount)
				inputEmitted = true
			} else {
				w.Emit("%sinput: %s,\n", IndentAmount, w.decorate(w.stripNamespace(inputName)))
			}
		}
		if outputShape != nil {
			if b := outputShape.Traits.Get("smithy.api#output"); b != nil {
				w.Emit("\n%soutput := {\n", IndentAmount)
				i2 := IndentAmount + IndentAmount
				for i, k := range outputShape.Members.Keys() {
					if i > 0 {
						w.Emit("\n")
					}
					v := outputShape.Members.Get(k)
					w.EmitTraits(v.Traits, i2)
					tname := w.decorate(w.stripNamespace(v.Target))
					w.Emit("%s%s: %s\n", i2, k, tname)
				}
				w.Emit("%s}\n", IndentAmount)
				outputEmitted = true
			} else {
				w.Emit("%soutput: %s,\n", IndentAmount, w.decorate(w.stripNamespace(outputName)))
			}
		}
		if len(shape.Errors) > 0 {
			w.Emit("\n%s%s\n", IndentAmount, w.listOfShapeRefs("errors", "%s", shape.Errors, false))
		}
	} else {
		if shape.Input != nil {
			w.Emit("    input: %s,\n", inputName)
		}
		if shape.Output != nil {
			w.Emit("    output: %s,\n", outputName)
		}
		if len(shape.Errors) > 0 {
			w.Emit("    %s,\n", w.listOfShapeRefs("errors", "%s", shape.Errors, false))
		}
	}
	w.Emit("}\n")
	emitted[name] = true
	if inputShape != nil {
		if !inputEmitted {
			w.EmitShape(inputName, inputShape)
		}
		emitted[inputName] = true
	}
	if outputShape != nil {
		if !outputEmitted {
			w.EmitShape(outputName, outputShape)
		}
		emitted[outputName] = true
	}
	/* emit these with other types
	if len(shape.Errors) > 0 {
		for _, errId := range shape.Errors {
			k := errId.Target
			if !emitted[k] {
				w.EmitShape(k, w.ast.GetShape(k))
				emitted[k] = true
			}
		}
	}
	*/
}

func (w *IdlWriter) End() string {
	w.writer.Flush()
	return w.buf.String()
}
