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
package common

import (
	"fmt"
	"strings"

	"github.com/boynton/api/model"
	"github.com/boynton/data"
)

const IndentAmount = "    "

// the generator for this tool's native format.
type ApiGenerator struct {
	BaseGenerator
	indent string
	ns     string
	name   string
}

func (gen *ApiGenerator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.indent = "    "
	gen.ns = string(schema.ServiceNamespace())
	gen.name = string(schema.ServiceName())
	gen.Begin()
	gen.GenerateSummary()
	gen.GenerateOperations()
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
	if gen.Schema.Version != "" {
		title = title + " v" + gen.Schema.Version
	}
	gen.GenerateBlockComment(gen.Schema.Comment, "")
	if gen.ns != "" {
		gen.Emitf("namespace %s\n", gen.ns)
	}
	if gen.name != "" {
		gen.Emitf("service %s\n", gen.name)
	}
	//other metadata?
	gen.Emit("\n")
}

func (gen *ApiGenerator) GenerateOperations() {
	for _, op := range gen.Schema.Operations {
		gen.GenerateOperation(op)
		gen.Emit("\n")
	}
}

func (gen *ApiGenerator) GenerateOperation(op *model.OperationDef) error {
	gen.GenerateBlockComment(op.Comment, "")
	rez := ""
	if op.Resource != "" {
		rez = ", resource=" + op.Resource
		if op.Lifecycle != "" {
			rez = rez + ", lifecycle=" + op.Lifecycle
		}
	}
	gen.Emitf("operation %s (method=%s, url=%q%s) {\n", StripNamespace(op.Id), op.HttpMethod, op.HttpUri, rez)
	gen.GenerateOperationInput(op)
	gen.GenerateOperationOutput(op)
	gen.GenerateOperationExceptions(op)
	gen.Emit("}\n")
	return nil
}

func (gen *ApiGenerator) GenerateOperationInput(op *model.OperationDef) {
	in := op.Input
	if in != nil {
		inname := ""
		if op.Input.Id != (op.Id+"Input") && op.Input.Id != "" {
			inname = "(name=" + StripNamespace(op.Input.Id) + ") "
		}
		gen.Emitf("    input %s{\n", inname)
		for _, f := range in.Fields {
			var opts []string
			if f.Required {
				opts = append(opts, "required")
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
			sopts := ""
			if len(opts) > 0 {
				sopts = " (" + strings.Join(opts, ", ") + ")"
			}
			comm := ""
			if f.Comment != "" {
				//format it?
				comm = " // " + f.Comment
			}
			gen.Emitf("        %s %s%s%s\n", f.Name, StripNamespace(f.Type), sopts, comm)
		}
		gen.Emit("    }\n")
	}
}

func (gen *ApiGenerator) GenerateOperationOutputFields(out *model.OperationOutput, indent string) {
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
		if f.Comment != "" {
			//format it?
			comm = " // " + f.Comment
		}
		gen.Emitf("    %s%s %s%s%s\n", indent, f.Name, StripNamespace(f.Type), sopts, comm)
	}
}

func (gen *ApiGenerator) GenerateOperationOutput(op *model.OperationDef) {
	if op.Output != nil {
		outname := ""
		if op.Output.Id != "" && op.Output.Id != (op.Id+"Output") {
			outname = "(name=" + StripNamespace(op.Output.Id) + ") "
		}
		gen.Emitf("    output %d %s{\n", op.Output.HttpStatus, outname)
		gen.GenerateOperationOutputFields(op.Output, "    ")
		gen.Emit("    }\n")
	}
}

func (gen *ApiGenerator) GenerateOperationExceptions(op *model.OperationDef) {
	if len(op.Exceptions) > 0 {
		for _, errdef := range op.Exceptions {
			defaultId := model.AbsoluteIdentifier(fmt.Sprintf("%sException%d", op.Id, errdef.HttpStatus))
			errname := ""
			if errdef.Id != "" && errdef.Id != defaultId {
				errname = "(name=" + StripNamespace(errdef.Id) + ") "
			}
			gen.Emitf("    exception %d %s{\n", errdef.HttpStatus, errname)
			gen.GenerateOperationOutputFields(errdef, "    ")
			gen.Emit("    }\n")
		}
	}
}

func (gen *ApiGenerator) GenerateFields(fields []*model.FieldDef, indent string) {
	forceCommentHeaders := false
	for _, f := range fields {
		if f.Comment != "" {
			if len(f.Comment) > 60 || strings.Index(f.Comment, "\n") >= 0 {
				forceCommentHeaders = true
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
			if forceCommentHeaders {
				//if len(f.Comment) > 60 || strings.Index(f.Comment, "\n") >= 0 {
				pcomm = FormatComment(indent, "// ", f.Comment, 72, false)
			} else {
				comm = " // " + f.Comment
			}
		}
		//if pcomm != "" {
		if forceCommentHeaders {
			gen.Emitf("\n%s%s%s %s%s%s\n", pcomm, indent, f.Name, StripNamespace(f.Type), sopts, comm)
		} else {
			gen.Emitf("%s%s %s%s%s\n", indent, f.Name, StripNamespace(f.Type), sopts, comm)
		}
	}
}

func (gen *ApiGenerator) GenerateTypes() {
	for _, td := range gen.Schema.Types {
		gen.GenerateType(td)
		gen.Emit("\n")
	}
}

func (gen *ApiGenerator) GenerateType(td *model.TypeDef) error {
	gen.GenerateBlockComment(td.Comment, "")
	switch td.Base {
	case model.String:
		sopts := ""
		var opts []string
		if td.Pattern != "" {
			opts = append(opts, fmt.Sprintf("pattern=%q", td.Pattern))
		}
		if len(opts) > 0 {
			sopts = " (" + strings.Join(opts, ", ") + ")"
		}
		gen.Emitf("type %s String%s\n", StripNamespace(td.Id), sopts)
	case model.Struct:
		gen.Emitf("type %s Struct {\n", StripNamespace(td.Id))
		gen.GenerateFields(td.Fields, "    ")
		gen.Emitf("}\n")
	case model.Union:
		gen.Emitf("type %s Union {\n", StripNamespace(td.Id))
		gen.GenerateFields(td.Fields, "    ")
		gen.Emitf("}\n")
	case model.List:
		gen.Emitf("type %s List[%s]\n", StripNamespace(td.Id), StripNamespace(td.Items))
	case model.Map:
		gen.Emitf("type %s Map[%s,%s]\n", StripNamespace(td.Id), StripNamespace(td.Keys), StripNamespace(td.Items))
	case model.Enum:
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
	case model.Timestamp:
		sopts := ""
		gen.Emitf("type %s Timestamp%s\n", StripNamespace(td.Id), sopts)
	case model.Int8, model.Int16, model.Int32, model.Int64, model.Float32, model.Float64, model.Integer, model.Decimal:
		sopts := ""
		gen.Emitf("type %s %s%s\n", StripNamespace(td.Id), td.Base.String(), sopts)
	default:
		gen.Emitf("type %s %s //FIX ME\n", StripNamespace(td.Id), td.Base)
	}
	return nil
}
