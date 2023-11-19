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
	"strings"

	"github.com/boynton/data"
	"github.com/boynton/api/model"
)

type SummaryGenerator struct {
	BaseGenerator
	indent string
	ns string
	name string
}

func (gen *SummaryGenerator) Generate(schema *model.Schema, config *data.Object) error {
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
	fname := gen.FileName(gen.name, ".txt")
	err = gen.Write(s, fname, "")
	return err
}

func (gen *SummaryGenerator) GenerateSummary() {
	anything := false
	if gen.Schema.Comment != "" {
		gen.Emit(FormatComment("", "// ", gen.Schema.Comment, 80, true))
		anything = true
	}
	if gen.ns != "" {
		gen.Emitf("namespace %s\n", gen.ns)
		anything = true
	}
	if gen.name != "" {
		gen.Emitf("service %s\n", gen.name)
		anything = true
	}
	if gen.Schema.Version != "" {
		gen.Emitf("version %s\n", gen.Schema.Version)
		anything = true
	}
	//other metadata?
	if anything {
		gen.Emit("\n")
	}
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

func (gen *SummaryGenerator) GenerateOperation(op *model.OperationDef) error {
	in := ExplodeInputs(op.Input)
	out := ExplodeOutputs(op.Output)
	errs := ""
	gen.Emitf("operation %s(%s) → (%s)%s\n", StripNamespace(op.Id), in, out, errs)
	return nil
}

func (gen *SummaryGenerator) GenerateOperations() {
	//this is a high level signature without types or exceptions
	ops := gen.Operations()
	if len(ops) > 0 {
		for _, op := range ops {
			gen.GenerateOperation(op)
		}
		gen.Emit("\n")
	}
}

func (gen *SummaryGenerator) GenerateType(td *model.TypeDef) error {
	switch td.Base {
	case model.Struct, model.Union:
		var lst []string
		for _, fd := range td.Fields {
			lst = append(lst, string(fd.Name))
		}
		s := ""
		if len(lst) > 0 {
			s = "{" + strings.Join(lst, ", ") + "}"
		}
		gen.Emitf("type %s %s %s\n", StripNamespace(td.Id), td.Base, s)
	case model.List:
		gen.Emitf("type %s List[%s]\n", StripNamespace(td.Id), StripNamespace(td.Items))
	case model.Map:
		gen.Emitf("type %s Map[%s → %s]\n", StripNamespace(td.Id), StripNamespace(td.Keys), StripNamespace(td.Items))
	default:
		gen.Emitf("type %s %s\n", StripNamespace(td.Id), td.Base)
	}
	return nil
}

func (gen *SummaryGenerator) GenerateTypes() {
	tds := gen.Types()
	if len(tds) > 0 {
		for _, td := range tds {
			gen.GenerateType(td)
		}
		gen.Emit("\n")
	}
}
