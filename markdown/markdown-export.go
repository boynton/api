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
package markdown

import (
	"fmt"
	"strings"

	"github.com/boynton/data"
	"github.com/boynton/api/common"
	"github.com/boynton/api/model"
)

const IndentAmount = "    "

type Generator struct {
	common.BaseGenerator
	ns string
	name string
}

func (gen *Generator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.ns = string(schema.ServiceNamespace())
	gen.name = string(schema.ServiceName())
	gen.Begin()
	gen.GenerateSummary()
	gen.GenerateOperations()
	gen.GenerateTypes()
	s := gen.End()
	fname := gen.FileName(gen.name, ".md")
	err = gen.Write(s, fname, "")
	return err
}

func (gen *Generator) GenerateSummary() {
	gen.Emitf("\n# %s\n\n", gen.name)
	gen.Emit(common.FormatComment("", "", gen.Schema.Comment, 80, true))
	gen.Emit("\n")
	if gen.name != "" {
		gen.Emitf("- **name**: %q\n", gen.name)
	}
	if gen.Schema.Version != "" {
		gen.Emitf("- **version**: %q\n", gen.Schema.Version)
	}
	if gen.ns != "" {
		gen.Emitf("- **namespace**: %q\n", gen.ns)
	}
	if gen.Schema.Base != "" {
		gen.Emitf("- **base**: %q\n", gen.Schema.Base)
	}
	gen.Emit("\n### Operation Index\n\n")
	for _, op := range gen.Schema.Operations {
		sum := summarySignature(op)
		s := StripNamespace(op.Id)
		gen.Emitf("- [%s](#%s)\n", sum, strings.ToLower(s))
	}
	gen.Emit("\n### Type Index\n\n")
	for _, td := range gen.Schema.Types {
		s := StripNamespace(td.Id)
		gen.Emitf("- [%s](#%s) → _%s_\n", s, strings.ToLower(s), td.Base)
	}
	gen.Emit("\n")
}

func StripNamespace(target model.AbsoluteIdentifier) string {
	t := string(target)
	n := strings.Index(t, "#")
	if n < 0 {
		return t
	}
	return t[n+1:]
}

func ExplodeInputs(in *model.OperationInput) string {
	var types []string
	for _, f := range in.Fields {
		//types = append(types, string(f.Name) + " " + StripNamespace(f.Type))
		types = append(types, string(f.Name))
	}
	return strings.Join(types, ", ")
}

func ExplodeOutputs(out *model.OperationOutput) string {
	var types []string
	for _, f := range out.Fields {
		//types = append(types, string(f.Name) + " " + StripNamespace(f.Type))
		types = append(types, string(f.Name))
	}
	return strings.Join(types, ", ")
}

func summarySignature(op *model.OperationDef) string {
	in := ExplodeInputs(op.Input)
	out := ExplodeOutputs(op.Output)
	s := StripNamespace(op.Id)
	return fmt.Sprintf("%s(%s) → (%s)", s, in, out)
}

func (gen *Generator) GenerateOperations() {
	//this is a high level signature without types or exceptions
	gen.Emitf("## Operations\n\n")
	for _, op := range gen.Schema.Operations {
		opId := StripNamespace(op.Id)
		gen.Emitf("### %s\n\n", opId)
		gen.Emitf("- **Method**: %s\n", op.HttpMethod)
		gen.Emitf("- **URI**: %s\n", op.HttpUri)
		//gen.Emitf("- **%s**:\n", StripNamespace(op.Input.Id))
		if op.Input != nil {
			gen.Emitf("- **Input**:\n")
			for _, f := range op.Input.Fields {
				var opts []string
				if f.Required {
					opts = append(opts, "required")
				}
				if f.Default != nil {
					opts = append(opts, "default=" + fmt.Sprint(f.Default))
				}
				if f.HttpPayload {
					opts = append(opts, "httpPayload")
				}
				if f.HttpPath {
					opts = append(opts, "httpPath")
				}
				if f.HttpQuery != "" {
					opts = append(opts, "httpQuery="+string(f.HttpQuery))
				}
				if f.HttpHeader != "" {
					opts = append(opts, "header=\""+string(f.HttpHeader) + "\"")
				}
				com := ""
				if f.Comment != "" {
					com = " // %s"
				}
				sopts := ""
				if len(opts) > 0 {
					sopts = " (" + strings.Join(opts, ", ") + ")"
				}
				stype := StripNamespace(f.Type)
				if gen.Schema.IsBaseType(f.Type) {
					stype = "_" + stype + "_"
				} else {
					stype = "[" + stype + "](#" + strings.ToLower(stype) +")"
				}
				gen.Emitf("    - **%s**: %s %s%s\n", f.Name, stype, sopts, com)
			}
		}
		if op.Output.Fields != nil {
			gen.Emitf("- **Output** (httpStatus=%d):\n", op.Output.HttpStatus)
			for _, f := range op.Output.Fields {
				sopts := ""
				com := ""
				stype := StripNamespace(f.Type)
				if gen.Schema.IsBaseType(f.Type) {
					stype = "_" + stype + "_"
				} else {
					stype = "[" + stype + "](#" + strings.ToLower(stype) +")"
				}
				gen.Emitf("    - **%s**: %s %s%s\n", f.Name, stype, sopts, com)
			}
		}
		if op.Exceptions != nil {
			gen.Emitf("- **Exceptions**:\n")
			for _, e := range op.Exceptions {
				gen.Emitf("    - **%s** (httpStatus=%d):\n", StripNamespace(e.Id), e.HttpStatus)
				for _, f := range e.Fields {
					sopts := ""
					com := ""
					stype := StripNamespace(f.Type)
					if gen.Schema.IsBaseType(f.Type) {
						stype = "_" + stype + "_"
					} else {
						stype = "[" + stype + "](#" + strings.ToLower(stype) +")"
					}
					gen.Emitf("        - **%s**: %s %s%s\n", f.Name, stype, sopts, com)
				}
			}
		}
	}
	gen.Emit("\n")
}

func (gen *Generator) GenerateTypes() {
	gen.Emitf("## Types\n\n")
	for _, td := range gen.Schema.Types {
		s := StripNamespace(td.Id)
		gen.Emitf("\n### %s\n\n", s)
		//gen.Emitf("\n### %s\n\n```\noperation %s(%s) → (%s)%s\n```\n", s, s, in, out, errs)
		switch td.Base {
		case model.Struct:
			gen.Emitf("- **Type**: Structure\n")
		case model.Union:
			gen.Emitf("- **Type**: Union\n")
		case model.List:
			gen.Emitf("- **Type**: List\n")
		case model.Map:
			gen.Emitf("- **Type**: Map\n")
		case model.Enum:
			gen.Emitf("- **Type**: Enum\n")
		case model.String:
			gen.Emitf("- **Type**: String\n")
		case model.Timestamp:
			gen.Emitf("- **Type**: Timestamp\n")
		case model.Int8:
			gen.Emitf("- **Type**: Int8\n")
		case model.Int16:
			gen.Emitf("- **Type**: Int16\n")
		case model.Int32:
			gen.Emitf("- **Type**: Int32\n")
		case model.Int64:
			gen.Emitf("- **Type**: Int64\n")
		case model.Integer:
			gen.Emitf("- **Type**: Integer\n")
		case model.Decimal:
			gen.Emitf("- **Type**: Decimal\n")
		case model.Float32:
			gen.Emitf("- **Type**: Float32\n")
		case model.Float64:
			gen.Emitf("- **Type**: Float64\n")
		default:
			panic("Handle this type: " + fmt.Sprint(td.Base))
		}
	}
	gen.Emit("\n")
}

