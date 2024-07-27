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
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/boynton/api/model"
	"github.com/boynton/api/smithy"
	"github.com/boynton/api/plantuml"
	"github.com/boynton/api/httptrace"
	"github.com/boynton/data"
)

const IndentAmount = "    "

type Generator struct {
	model.BaseGenerator
	ns              string
	name            string
	detailGenerator string
	//	useHtmlPreTag   bool
	diagramsFolder     string
	showExamples    bool
}

func (gen *Generator) getDetailGenerator() model.Generator {
	var dec *model.Decorator
	/*
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
	*/
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
	//gen.useHtmlPreTag = config.GetBool("use-html-pre-tag")
	gen.diagramsFolder = config.GetString("diagrams-folder")
	gen.showExamples = config.GetBool("show-examples")
	gen.Begin()
	gen.GenerateSummary()
	gen.GenerateResources()
	gen.GenerateOperations()
	gen.GenerateExceptions()
	gen.GenerateTypes()
	s := gen.End()
	fname := gen.FileName(gen.name, ".md")
	err = gen.Write(s, fname, "")
	if err != nil {
		return err
	}
	if gen.diagramsFolder != "" {
		for _, rez := range gen.Schema.Resources {
			err = gen.ensureResourceDiagram(rez)
			if err != nil {
				return err
			}
		}
	}
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
	rezs := gen.Schema.Resources
	if len(rezs) > 0 {
		gen.Emitf("\n### Resource Index\n")
		for _, rez := range rezs {
			s := StripNamespace(rez.Id)
			gen.Emitf("- [%s](#%s)\n", s, strings.ToLower(s))
		}
		gen.Emitf("\n")
	}
	ops := gen.Operations()
	if len(ops) > 0 {
		gen.Emit("\n### Operation Index\n\n")
		for _, op := range ops {
			sum := summarySignature(op)
			s := StripNamespace(op.Id)
			gen.Emitf("- [%s](#%s)\n", sum, strings.ToLower(s))
		}
	}
	excs := gen.Exceptions()
	if len(excs) > 0 {
		gen.Emit("\n### Exception Index\n\n")
		for _, exc := range excs {
			s := StripNamespace(exc.Id)
			gen.Emitf("- [%s](#%s)\n", s, strings.ToLower(s))
		}
	}
	types := gen.Types()
	if len(types) > 0 {
		gen.Emit("\n### Type Index\n\n")
		for _, td := range types {
			//check if a type has input or output trait, if so, omit it here.
			s := StripNamespace(td.Id)
			gen.Emitf("- [%s](#%s) → _%s_\n", s, strings.ToLower(s), td.Base)
		}
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

func (gen *Generator) generateApiResource(op *model.ResourceDef) string {
	g := gen.getDetailGenerator()
	conf := data.NewObject()
	err := g.Configure(gen.Schema, conf)
	if err != nil {
		return "Whoops: " + err.Error()
	}
	g.Begin()
	g.GenerateResource(op)
	s := g.End()
	return s
}

func (gen *Generator) GenerateResource(rez *model.ResourceDef) error {
	rezId := StripNamespace(rez.Id)
	gen.Emitf("### %s\n\n", rezId)
	gen.Emitf("```\n%s```\n\n", gen.generateApiResource(rez))
	if gen.diagramsFolder != "" {
		gen.Emitf("![ER Diagram for %s](%s/%s.png)\n", rezId, gen.diagramsFolder, rezId)
	}
	return nil
}

func (gen *Generator) GenerateResources() {
	gen.Emitf("## Resources\n\n")
	if len(gen.Schema.Resources) > 0 {
		for _, rez := range gen.Schema.Resources {
			gen.GenerateResource(rez)
		}
		gen.Emit("\n")
	} else {
		gen.Emitf("(_No Resources defined in model_)\n\n")
	}
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

var exampleTrace string = `#
# Create Item 1
#
POST /items HTTP/1.1
Accept: application/json
Content-Type: application/json; charset=utf-8
Content-Length: 29

{
  "title": "Test Item 1"
}

HTTP/1.1 201 Created
Content-Length: 46
Content-Type: application/json; charset=utf-8
Date: Fri, 26 Jul 2024 09:53:47 GMT

{
  "id": "item1",
  "title": "Test Item 1"
}
`

func (gen *Generator) GenerateOperation(op *model.OperationDef) error {
	opId := StripNamespace(op.Id)
	gen.Emitf("### %s\n\n", opId)
	gen.Emitf("```\n%s```\n\n", gen.generateApiOperation(op))
	if gen.showExamples && len(op.Examples) > 0 {
		gen.Emitf("\n#### %s Examples\n\n", opId)
		hgen := new(httptrace.Generator)
		hgen.Configure(gen.Schema, nil)
		for _, ex := range op.Examples {
			snippet, err := hgen.EmitHttpTrace(op, ex)
			if err != nil {
				return err
			}
			gen.Emitf("```\n%s\n```\n", snippet)
		}
	}
	return nil
}

func (gen *Generator) GenerateOperations() {
	gen.Emitf("## Operations\n\n")
	if len(gen.Schema.Operations) > 0 {
		for _, op := range gen.Operations() {
			gen.GenerateOperation(op)
		}
		gen.Emit("\n")
	} else {
		gen.Emitf("(_No Operations defined in model_)\n\n")
	}
}

func (gen *Generator) generateApiException(op *model.OperationOutput) string {
	g := gen.getDetailGenerator()
	conf := data.NewObject()
	err := g.Configure(gen.Schema, conf)
	if err != nil {
		return "Whoops: " + err.Error()
	}
	g.Begin()
	g.GenerateException(op)
	s := g.End()
	return s
}

func (gen *Generator) GenerateException(out *model.OperationOutput) error {
	outId := StripNamespace(out.Id)
	gen.Emitf("### %s\n\n", outId)
	gen.Emitf("```\n%s```\n\n", gen.generateApiException(out))
	return nil
}

func (gen *Generator) GenerateExceptions() {
	//this is a high level signature without types or exceptions
	if len(gen.Schema.Exceptions) > 0 {
		gen.Emitf("## Exceptions\n\n")
		for _, op := range gen.Exceptions() {
			gen.GenerateException(op)
		}
		gen.Emit("\n")
	}
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
	gen.Emitf("## Types\n\n")
	if len(tds) > 0 {
		for _, td := range gen.Types() {
			gen.GenerateType(td)
		}
		//to do: generate exception types for operations, since Smithy does not inline them
		//
		gen.Emit("\n")
	} else {
		gen.Emitf("(_No Types defined in model_)\n\n")
	}
}

func (gen *Generator) ensureResourceDiagram(rez *model.ResourceDef) error {
	if gen.OutDir == "" {
		return nil
	}
	rezId := StripNamespace(rez.Id)
	rezSchema := gen.Schema.ShakeResourceTree(rez.Id, true)
	imgdir := ""
	if gen.diagramsFolder != "." {
		imgdir = gen.diagramsFolder
		gen.EnsureDir(filepath.Join(gen.OutDir, imgdir))
	}
	pumlPath := fmt.Sprintf("%s/%s.puml", imgdir, rezId)
	pumlConf := data.NewObject()
	pumlConf.Put("force", true)
	pumlConf.Put("outdir", gen.OutDir)
	pumlConf.Put("generate-exceptions", gen.Config.GetBool("generate-exceptions"))
	pumlConf.Put("suppress-service", true)
	//pumlConf.Put("generate-exceptions", true)
	pumlGen := new(plantuml.Generator)
	err := pumlGen.Init(rezSchema, pumlConf)
	if err != nil {
		return err
	}
	pumlGen.Begin()
	pumlGen.GenerateHeader()
	pumlGen.GenerateResource(rez)
	pumlGen.GenerateExceptions()
	pumlGen.GenerateTypes()
	pumlGen.GenerateFooter()
	s := pumlGen.End()
	err = pumlGen.Write(s, pumlPath, "")
	if err != nil {
		return err
	}
	cmd := exec.Command("plantuml", filepath.Join(gen.OutDir, pumlPath))
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("Cannot exec `plantuml`. Please install it and try again")
	}
	return err
}
