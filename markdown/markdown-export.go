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

	"github.com/boynton/api/model"
	"github.com/boynton/api/smithy"
	"github.com/boynton/data"
)

const IndentAmount = "    "

type Generator struct {
	model.BaseGenerator
	ns              string
	name            string
	detailGenerator string
	useHtmlPreTag       bool
}

func (gen *Generator) getDetailGenerator() model.Generator {
	var dec *model.Decorator
	if gen.useHtmlPreTag {
		dec = &model.Decorator{
			BaseType: func(s string) string {
				return fmt.Sprintf("<em><strong>%s</strong></em>", s)
			},
			UserType: func(s string) string {
				return fmt.Sprintf("<em><strong><a href=\"#%s\">%s</a></strong></em>", strings.ToLower(s), s)
			},
		}
	}
	switch gen.detailGenerator {
	case "api":
		g := new(model.ApiGenerator)
		g.Decorator = dec
		return g
	default: //smithy
		g := new(smithy.IdlGenerator)
		g.Sort = gen.Sort
		g.Decorator = dec
		return g
	}
}

func (gen *Generator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.ns = string(schema.ServiceNamespace())
	gen.name = string(schema.ServiceName())
	gen.detailGenerator = config.GetString("detail-generator") //should be either "smithy" or "
	gen.useHtmlPreTag = config.GetBool("use-html-pre-tag")
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
	gen.Emit(model.FormatComment("", "", gen.Schema.Comment, 80, true))
	gen.Emit("\n")
	if gen.name != "" {
		gen.Emitf("- **service**: %q\n", gen.name)
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
	for _, op := range gen.Operations() {
		sum := summarySignature(op)
		s := StripNamespace(op.Id)
		gen.Emitf("- [%s](#%s)\n", sum, strings.ToLower(s))
	}
	gen.Emit("\n### Type Index\n\n")
	for _, td := range gen.Types() {
		//check if a type has input or output trait, if so, omit it here.
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
	if in != nil {
		for _, f := range in.Fields {
			//types = append(types, string(f.Name) + " " + StripNamespace(f.Type))
			types = append(types, string(f.Name))
		}
		return strings.Join(types, ", ")
	}
	return ""
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

func (gen *Generator) generateApiOperation(op *model.OperationDef) string {
	g := gen.getDetailGenerator()
	conf := data.NewObject()
	err := g.Configure(gen.Schema, conf)
	if err != nil {
		return "Whoops: " + err.Error()
	}
	g.Begin()
	g.GenerateOperation(op)
	s := g.End()
	return s
}

func (gen *Generator) GenerateOperation(op *model.OperationDef) error {
	opId := StripNamespace(op.Id)
	gen.Emitf("### %s\n\n", opId)
	if true { //doesn't work with some quicklook markdown viewers, the <pre> block gets skipped altogether
		gen.Emitf("<pre>\n%s</pre>\n\n", gen.generateApiOperation(op))
	} else {
		gen.Emitf("```\n%s```\n\n", gen.generateApiOperation(op))
	}
	return nil
}

func (gen *Generator) GenerateOperations() {
	//this is a high level signature without types or exceptions
	gen.Emitf("## Operations\n\n")
	if len(gen.Schema.Operations) > 0 {
		for _, op := range gen.Operations() {
			gen.GenerateOperation(op)
		}
		gen.Emit("\n")
	}
}

func (gen *Generator) GenerateException(op *model.OperationOutput) error {
	return nil
}

func (gen *Generator) GenerateType(td *model.TypeDef) error {
	s := StripNamespace(td.Id)
	gen.Emitf("\n### %s\n\n", s)
	gen.Emitf("```\n%s```\n\n", gen.generateApiType(td))
	return nil
}

func (gen *Generator) generateApiType(op *model.TypeDef) string {
	g := gen.getDetailGenerator()
	conf := data.NewObject()
	err := g.Configure(gen.Schema, conf)
	if err != nil {
		return "Whoops: " + err.Error()
	}
	g.Begin()
	g.GenerateType(op)
	s := g.End()
	return s
}

func (gen *Generator) GenerateTypes() {
	tds := gen.Schema.Types
	if len(tds) > 0 {
		gen.Emitf("## Types\n\n")
		for _, td := range gen.Types() {
			gen.GenerateType(td)
		}
		//to do: generate exception types for operations, since Smithy does not inline them
		//
		gen.Emit("\n")
	}
}
