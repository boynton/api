/*
Copyright 2023 Lee R. Boynton

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
package model

import (
	"fmt"
	"strings"

	"github.com/boynton/data"
)

type DecoratorFunc func(string) string

type Decorator struct {
	BaseType DecoratorFunc
	UserType DecoratorFunc
}

const IndentAmount = "    "

// the generator for this tool's native format.
type ApiGenerator struct {
	BaseGenerator
	Decorator *Decorator
	indent    string
	ns        string
	name      string
}

func (gen *ApiGenerator) Generate(schema *Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.indent = "    "
	gen.ns = config.GetString("namespace")
	if gen.ns == "" {
		gen.ns = string(schema.ServiceNamespace())
	}
	gen.name = string(schema.ServiceName())
	gen.Begin()
	gen.GenerateSummary()
	gen.GenerateResources()
	gen.GenerateOperations()
	gen.GenerateExceptions()
	gen.GenerateTypes()
	s := gen.End()
	fname := gen.FileName(gen.name, ".api")
	err = gen.Write(s, fname, "")
	return err
}

func (gen *ApiGenerator) GenerateBlockComment(comment string, indent string) {
	if comment != "" {
		gen.Emit(FormatComment(indent, "// ", comment, 80, true))
	}
}

func (gen *ApiGenerator) GenerateSummary() {
	title := gen.name
	version := gen.Schema.Version
	gen.GenerateBlockComment(gen.Schema.Comment, "")
	if gen.ns != "" {
		gen.Emitf("namespace %s\n", gen.ns)
	}
	if gen.name != "" {
		gen.Emitf("service %s\n", title)
	}
	if version != "" {
		gen.Emitf("version %q\n", version)
	}
	//other metadata?
	gen.Emit("\n")
}

func (gen *ApiGenerator) GenerateResources() {
	for _, rez := range gen.Schema.Resources {
		gen.GenerateResource(rez)
		gen.Emit("\n")
	}
}

func (gen *ApiGenerator) GenerateOperations() {
	for _, op := range gen.Schema.Operations {
		gen.GenerateOperation(op)
		gen.Emit("\n")
	}
}

func (gen *ApiGenerator) GenerateExceptions() {
	for _, exc := range gen.Schema.Exceptions {
		gen.GenerateException(exc)
	}
}

func (gen *ApiGenerator) GenerateException(edef *OperationOutput) error {
	gen.GenerateBlockComment(edef.Comment, "")
	ename := gen.decorateReference(StripNamespace(edef.Id))
	gen.Emitf("exception %s (status=%d) {\n", ename, edef.HttpStatus)
	gen.GenerateOperationOutputFields(edef, "    ")
	gen.Emitf("}\n\n")
	return nil
}

func (gen *ApiGenerator) GenerateResource(rez *ResourceDef) error {
	gen.GenerateBlockComment(rez.Comment, "")
	rezId := StripNamespace(rez.Id)
	gen.Emitf("resource %s {\n", rezId)
	if rez.Create != "" {
		gen.Emitf("    create %s\n", gen.decorateReference(StripNamespace(rez.Create)))
	}
	if rez.Read != "" {
		gen.Emitf("    read %s\n", gen.decorateReference(StripNamespace(rez.Read)))
	}
	if rez.Update != "" {
		gen.Emitf("    update %s\n", gen.decorateReference(StripNamespace(rez.Update)))
	}
	if rez.Delete != "" {
		gen.Emitf("    delete %s\n", gen.decorateReference(StripNamespace(rez.Delete)))
	}
	if rez.List != "" {
		gen.Emitf("    list %s\n", gen.decorateReference(StripNamespace(rez.List)))
	}
	if len(rez.Operations) > 0 {
		var ops []string
		for _, op := range rez.Operations {
			s := gen.decorateReference(StripNamespace(op))
			ops = append(ops, s)
		}
		gen.Emitf("    operations [%s]\n", strings.Join(ops, ", "))
	}
	if len(rez.CollectionOperations) > 0 {
		var ops []string
		for _, op := range rez.CollectionOperations {
			s := gen.decorateReference(StripNamespace(op))
			ops = append(ops, s)
		}
		gen.Emitf("    collectionOperations [%s]\n", strings.Join(ops, ", "))
	}
	gen.Emit("}\n")
	return nil
}

func (gen *ApiGenerator) GenerateOperation(op *OperationDef) error {
	gen.GenerateBlockComment(op.Comment, "")
	gen.Emitf("operation %s (method=%s, url=%q) {\n", StripNamespace(op.Id), op.HttpMethod, op.HttpUri)
	gen.GenerateOperationInput(op)
	gen.GenerateOperationOutput(op)
	gen.GenerateOperationExceptionRefs(op)
	gen.GenerateOperationExamples(op)
	gen.Emit("}\n")
	return nil
}

func (gen *ApiGenerator) GenerateOperationInput(op *OperationDef) {
	in := op.Input
	if in != nil {
		indent := "        "
		commentHeaders := false
		for _, f := range in.Fields {
			if f.Comment != "" {
				if len(f.Comment) > 60 || strings.Index(f.Comment, "\n") >= 0 {
					commentHeaders = true
				}
			}
		}
		inname := ""
		if op.Input.Id != (op.Id+"Input") && op.Input.Id != "" {
			inname = "(name=" + StripNamespace(op.Input.Id) + ") "
		}
		gen.Emitf("    input %s{\n", inname)
		firstPad := ""
		for _, f := range in.Fields {
			var opts []string
			if f.Required {
				opts = append(opts, "required")
			} else if f.Default != nil {
				opts = append(opts, fmt.Sprintf("default=%s", data.JsonEncode(f.Default)))
			}
			if f.HttpPayload {
				opts = append(opts, "payload")
			} else if f.HttpPath {
				opts = append(opts, "path")
			} else if f.HttpQuery != "" {
				opts = append(opts, fmt.Sprintf("query=%q", f.HttpQuery))
			} else if f.HttpHeader != "" {
				opts = append(opts, fmt.Sprintf("header=%q", f.HttpHeader))
			}
			if f.MinValue != nil {
				opts = append(opts, fmt.Sprintf("minvalue=%v", f.MinValue))
			}
			if f.MaxValue != nil {
				opts = append(opts, fmt.Sprintf("maxvalue=%v", f.MaxValue))
			}
			if f.MinSize != 0 {
				opts = append(opts, fmt.Sprintf("minsize=%d", f.MinSize))
			}
			if f.MaxSize != 0 {
				opts = append(opts, fmt.Sprintf("maxsize=%d", f.MaxSize))
			}
			sopts := ""
			if len(opts) > 0 {
				sopts = " (" + strings.Join(opts, ", ") + ")"
			}
			comm := ""
			pcomm := ""
			if f.Comment != "" {
				if commentHeaders {
					//if len(f.Comment) > 60 || strings.Index(f.Comment, "\n") >= 0 {
					pcomm = FormatComment(indent, "// ", f.Comment, 72, false)
				} else {
					comm = " // " + f.Comment
				}
			}
			tname := gen.decorateReference(StripNamespace(f.Type))
			if commentHeaders {

				gen.Emitf("%s%s%s%s %s%s%s\n", firstPad, pcomm, indent, f.Name, tname, sopts, comm)
			} else {
				gen.Emitf("%s%s %s%s%s\n", indent, f.Name, tname, sopts, comm)
			}
			firstPad = "\n"
		}
		gen.Emit("    }\n")
	}
}

func (gen *ApiGenerator) decorateReference(tname string) string {
	if gen.Decorator != nil {
		//user defined types:
		switch tname {
		case "Int32", "String", "Int16", "Int8", "Int64", "Float64", "Float32", "Decimal", "Integer":
			return gen.Decorator.BaseType(tname)
		case "Timestamp":
			return gen.Decorator.BaseType(tname)
		}
		return gen.Decorator.UserType(tname)
	}
	return tname
}

func (gen *ApiGenerator) GenerateOperationOutputFields(out *OperationOutput, indent string) {
	commentHeaders := false
	for _, f := range out.Fields {
		if f.Comment != "" {
			if len(f.Comment) > 60 || strings.Index(f.Comment, "\n") >= 0 {
				commentHeaders = true
			}
		}
	}
	firstPad := ""
	for _, f := range out.Fields {
		var opts []string
		if f.HttpPayload {
			opts = append(opts, "payload")
		}
		if f.HttpHeader != "" {
			opts = append(opts, fmt.Sprintf("header=%q", f.HttpHeader))
		}
		sopts := ""
		if len(opts) > 0 {
			sopts = " (" + strings.Join(opts, ", ") + ")"
		}
		comm := ""
		pcomm := ""
		if f.Comment != "" {
			if commentHeaders {
				//if len(f.Comment) > 60 || strings.Index(f.Comment, "\n") >= 0 {
				pcomm = FormatComment(indent, "// ", f.Comment, 72, false)
			} else {
				comm = " // " + f.Comment
			}
		}
		tname := gen.decorateReference(StripNamespace(f.Type))
		if commentHeaders {
			gen.Emitf("%s%s%s%s %s%s%s\n", firstPad, pcomm, indent, f.Name, tname, sopts, comm)
		} else {
			gen.Emitf("%s%s %s%s%s\n", indent, f.Name, tname, sopts, comm)
		}
		firstPad = "\n"
	}
}

func (gen *ApiGenerator) GenerateOperationOutput(op *OperationDef) {
	if op.Output != nil {
		opts := fmt.Sprintf("(status=%d", op.Output.HttpStatus)
		if op.Output.Id != "" && op.Output.Id != (op.Id+"Output") {
			opts = opts + ", name=" + gen.decorateReference(StripNamespace(op.Output.Id))
		}
		opts = opts + ") "
		gen.Emitf("    output %s{\n", opts)
		gen.GenerateOperationOutputFields(op.Output, "        ")
		gen.Emit("    }\n")
	}
}

func (gen *ApiGenerator) GenerateOperationExceptionRefs(op *OperationDef) {
	if len(op.Exceptions) > 0 {
		exceptions := make([]string, 0)
		for _, errid := range op.Exceptions {
			errname := gen.decorateReference(StripNamespace(errid))
			exceptions = append(exceptions, errname)
		}
		gen.Emitf("    exceptions [%s]\n", strings.Join(exceptions, ", "))
	}
}

func (gen *ApiGenerator) GenerateOperationExamples(op *OperationDef) {
	if len(op.Examples) > 0 {
		gen.Emitf("    examples [\n")
		for _, ex := range op.Examples {
			gen.Emitf("        {\n")
			gen.Emitf("            title: %q\n", ex.Title)
			if ex.Input != nil {
				gen.Emitf("            input: %s\n", prettyData(ex.Input, "            "))
			}
			if ex.Output != nil {
				gen.Emitf("            output: %s\n", prettyData(ex.Output, "            "))
			}
			if ex.Error != nil {
				gen.Emitf("            error: {\n")
				gen.Emitf("                shapeId: %q\n", ex.Error.ShapeId)
				gen.Emitf("                output: %s\n", prettyData(ex.Error.Output, "                "))
				gen.Emitf("            }\n")
			}
			gen.Emitf("        }\n")
		}
		gen.Emitf("    ]\n")
	}
}

func prettyData(a any, indent string) string {
	nextIndent := indent + "    "
	if a == nil {
		return "null"
	}
	switch v := a.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case int32:
		return fmt.Sprintf("%d", v)
	case int:
		return fmt.Sprintf("%d", v)
	case []any:
		s := "[\n"
		i := 0
		last := len(v) - 1
		comma := ","
		for _, v := range v {
			if i == last {
				comma = ""
			}
			i++
			s = s + fmt.Sprintf("%s%s%s\n", nextIndent, prettyData(v, nextIndent), comma)
		}
		return s + indent + "]"
	case map[string]any:
		s := "{\n"
		i := 0
		last := len(v) - 1
		comma := ","
		for k, v := range v {
			if i == last {
				comma = ""
			}
			i++
			s = s + fmt.Sprintf("%s%q: %s%s\n", nextIndent, k, prettyData(v, nextIndent), comma)
		}
		return s + indent + "}"
	default:
		fmt.Println("a:", v)
		panic("prettyData: FIX ME")
	}
}

func (gen *ApiGenerator) GenerateFields(fields []*FieldDef, indent string) {
	commentHeaders := false
	for _, f := range fields {
		if f.Comment != "" {
			if len(f.Comment) > 60 || strings.Index(f.Comment, "\n") >= 0 {
				commentHeaders = true
			}
		}
	}
	for _, f := range fields {
		var opts []string
		if f.Required {
			opts = append(opts, "required")
		}
		sopts := ""
		if len(opts) > 0 {
			sopts = " (" + strings.Join(opts, ", ") + ")"
		}
		comm := ""
		pcomm := ""
		if f.Comment != "" {
			if commentHeaders {
				//if len(f.Comment) > 60 || strings.Index(f.Comment, "\n") >= 0 {
				pcomm = FormatComment(indent, "// ", f.Comment, 72, false)
			} else {
				comm = " // " + f.Comment
			}
		}
		tname := gen.decorateReference(StripNamespace(f.Type))
		if commentHeaders {
			gen.Emitf("\n%s%s%s %s%s%s\n", pcomm, indent, f.Name, tname, sopts, comm)
		} else {
			gen.Emitf("%s%s %s%s%s\n", indent, f.Name, tname, sopts, comm)
		}
	}
}

func (gen *ApiGenerator) GenerateExamples() {
	for _, op := range gen.Schema.Operations {
		for _, ex := range op.Examples {
			gen.GenerateExample(op, ex)
		}
	}
}
func (gen *ApiGenerator) GenerateExample(op *OperationDef, ex *OperationExample) {
	opName := StripNamespace(op.Id)
	gen.Emitf("example %s %s\n", opName, data.Pretty(ex))
}

func (gen *ApiGenerator) GenerateTypes() {
	for _, td := range gen.Schema.Types {
		gen.GenerateType(td)
		gen.Emit("\n")
	}
}

func (gen *ApiGenerator) GenerateType(td *TypeDef) error {
	gen.GenerateBlockComment(td.Comment, "")
	switch td.Base {
	case BaseType_String:
		sopts := ""
		var opts []string
		if td.Pattern != "" {
			opts = append(opts, fmt.Sprintf("pattern=%q", td.Pattern))
		}
		if len(opts) > 0 {
			sopts = " (" + strings.Join(opts, ", ") + ")"
		}
		gen.Emitf("type %s String%s\n", StripNamespace(td.Id), sopts)
	case BaseType_Struct:
		gen.Emitf("type %s Struct {\n", StripNamespace(td.Id))
		gen.GenerateFields(td.Fields, "    ")
		gen.Emitf("}\n")
	case BaseType_Union:
		gen.Emitf("type %s Union {\n", StripNamespace(td.Id))
		gen.GenerateFields(td.Fields, "    ")
		gen.Emitf("}\n")
	case BaseType_List:
		gen.Emitf("type %s List[%s]\n", StripNamespace(td.Id), gen.decorateReference(StripNamespace(td.Items)))
	case BaseType_Map:
		gen.Emitf("type %s Map[%s,%s]\n", StripNamespace(td.Id), gen.decorateReference(StripNamespace(td.Keys)), gen.decorateReference(StripNamespace(td.Items)))
	case BaseType_Enum:
		sopt := ""
		//for _, el := range td.Elements {
		//if el.Type != "" {
		//	panic("alternate enum types NYI")
		//}
		//}
		gen.Emitf("type %s Enum %s{\n", StripNamespace(td.Id), sopt)
		for _, el := range td.Elements {
			sopts := ""
			var opts []string
			if el.Value != "" && el.Value != string(el.Symbol) {
				opts = append(opts, fmt.Sprintf("value=%q", el.Value))
			}
			if len(opts) > 0 {
				sopts = " (" + strings.Join(opts, ", ") + ")"
			}
			gen.Emitf("    %s%s\n", el.Symbol, sopts)
		}
		gen.Emitf("}\n")
	case BaseType_Timestamp:
		sopts := ""
		gen.Emitf("type %s Timestamp%s\n", StripNamespace(td.Id), sopts)
	case BaseType_Int8, BaseType_Int16, BaseType_Int32, BaseType_Int64, BaseType_Float32, BaseType_Float64, BaseType_Integer, BaseType_Decimal:
		sopts := ""
		gen.Emitf("type %s %s%s\n", StripNamespace(td.Id), td.Base.String(), sopts)
	case BaseType_Bool:
		sopts := ""
		gen.Emitf("type %s Bool%s\n", StripNamespace(td.Id), sopts)
	default:
		gen.Emitf("type %s %s //FIX ME\n", StripNamespace(td.Id), td.Base)
	}
	return nil
}
