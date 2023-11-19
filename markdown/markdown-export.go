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

func (gen *Generator) generateApiOperation(op *model.OperationDef) string {
	g := new(common.ApiGenerator)
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

func (gen *Generator) GenerateOperations() {
	//this is a high level signature without types or exceptions
	gen.Emitf("## Operations\n\n")
	if len(gen.Schema.Operations) > 0 {
		for _, op := range gen.Operations() {
			opId := StripNamespace(op.Id)
			gen.Emitf("### %s\n\n", opId)
			gen.Emitf("```\n%s```\n\n", gen.generateApiOperation(op))
		}
		gen.Emit("\n")
	}
}

func (gen *Generator) generateApiType(op *model.TypeDef) string {
	g := new(common.ApiGenerator)
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
			s := StripNamespace(td.Id)
			gen.Emitf("\n### %s\n\n", s)
			gen.Emitf("```\n%s```\n\n", gen.generateApiType(td))
		}
		gen.Emit("\n")
	}
}

