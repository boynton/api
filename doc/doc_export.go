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
package doc

import (
	"fmt"
	"strings"
	
	"github.com/boynton/api/model"
	"github.com/boynton/api/smithy"
	"github.com/boynton/data"
)

const IndentAmount = "    "

type DocFormat interface {
	FileExtension() string
	RenderHeader()
	RenderSummary()
	RenderFooter()
}

type Generator struct {
	model.BaseGenerator
	ns              string
	name            string
	detailGenerator string
	format          DocFormat
}

func (gen *Generator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.ns = string(schema.ServiceNamespace())
	gen.name = string(schema.ServiceName())
	switch config.GetString("format") {
	case "html":
		gen.format = &HtmlFormat{gen: gen}
	default:
		gen.format = &MarkdownFormat{gen: gen}
	}
	gen.detailGenerator = config.GetString("detail-generator") //should be either "smithy" or "api". FIX
	if gen.detailGenerator == "" {
		gen.detailGenerator = "smithy"
	}
	gen.Begin()
	gen.format.RenderHeader()
	gen.format.RenderSummary()
	//gen.GenerateResources()
	//gen.GenerateOperations()
	//gen.GenerateExceptions()
	//gen.GenerateTypes()
	gen.format.RenderFooter()
	s := gen.End()
	ext := gen.format.FileExtension()
	fname := gen.FileName(gen.name, ext)
	err = gen.Write(s, fname, "")
	return err
}

func (gen *Generator) ResourceIds() []model.AbsoluteIdentifier {
	var resources []model.AbsoluteIdentifier
	if gen.detailGenerator == "smithy" {
		ast, err := smithy.SmithyAST(gen.Schema, gen.Sort)
		if err != nil {
			return resources
		}
		for _, shapeId := range ast.Shapes.Keys() {
			shape := ast.GetShape(shapeId)
			if shape.Type == "resource" {
				resources = append(resources, model.AbsoluteIdentifier(shapeId))
			}
		}
	}
	return resources
}

func (gen *Generator) getDetailGenerator() model.Generator {
	dec := model.Decorator{
		BaseType: func(s string) string {
			return fmt.Sprintf("<em><strong>%s</strong></em>", s)
		},
		UserType: func(s string) string {
			return fmt.Sprintf("<em><strong><a href=\"#%s\">%s</a></strong></em>", strings.ToLower(s), s)
		},
	}
	switch gen.detailGenerator {
	case "api":
		g := new(model.ApiGenerator)
		g.Decorator = &dec
		return g
	default: //smithy
		g := new(smithy.IdlGenerator)
		g.Sort = gen.Sort
		g.Decorator = &dec
		return g
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
	return fmt.Sprintf("<b>%s</b>(%s) → (%s)", s, in, out)
}

func (gen *Generator) generateApiResource(sg *smithy.IdlGenerator, id model.AbsoluteIdentifier) string {
	sg.Begin()
	sg.GenerateResource(string(id))
	s := sg.End()
	return s
}

func (gen *Generator) GenerateResource(sg *smithy.IdlGenerator, id model.AbsoluteIdentifier) error {
	rezId := StripNamespace(id)
	gen.Emitf("<h3 id=%q>%s</h3>\n", strings.ToLower(rezId), rezId)
	gen.Emitf("<pre class=\"mknohighlight\"><code>\n")
	gen.Emitf("%s\n\n", gen.generateApiResource(sg, id))
	gen.Emitf("</code></pre>\n")
	return nil
}

func (gen *Generator) GenerateResources() {
	if gen.detailGenerator == "smithy" {
		resourceIds := gen.ResourceIds()
		if len(resourceIds) > 0 {
			g := gen.getDetailGenerator()
			conf := data.NewObject()
			conf.Put("sort", gen.Sort)
			err := g.Configure(gen.Schema, conf)
			if err != nil {
				return
			}
			if sg, ok := g.(*smithy.IdlGenerator); ok {
				gen.Emitf("<h2 id=\"resources\">Resources</h2>\n")
				for _, id := range resourceIds {
					gen.GenerateResource(sg, id)
				}
				gen.Emit("\n")
			}
		}
	}
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
	gen.Emitf("<h3 id=%q>%s</h3>\n", strings.ToLower(opId), opId)
	gen.Emitf("<pre class=\"mknohighlight\"><code>\n")
	gen.Emitf("%s\n\n", gen.generateApiOperation(op))
	gen.Emitf("</code></pre>\n")
	return nil
}

func (gen *Generator) GenerateOperations() {
	//this is a high level signature without types or exceptions
	gen.Emitf("<h2 id=\"operations\">Operations</h2>\n")
	if len(gen.Schema.Operations) > 0 {
		for _, op := range gen.Operations() {
			gen.GenerateOperation(op)
		}
		gen.Emit("\n")
	}
}

func (gen *Generator) GenerateException(exc *model.OperationOutput) error {
	s := StripNamespace(exc.Id)
	gen.Emitf("<h3 id=%q>%s</h3>\n", strings.ToLower(s), s)
	gen.Emitf("<pre class=\"mknohighlight\"><code>\n")
	gen.Emitf("%s\n", gen.generateExceptionType(exc))
	gen.Emitf("</code></pre>\n")
	return nil
}

func (gen *Generator) generateExceptionType(exc *model.OperationOutput) string {
	g := gen.getDetailGenerator()
	conf := data.NewObject()
	err := g.Configure(gen.Schema, conf)
	if err != nil {
		return "Whoops: " + err.Error()
	}
	g.Begin()
	g.GenerateException(exc)
	s := g.End()
	return s
}

func (gen *Generator) GenerateType(td *model.TypeDef) error {
	s := StripNamespace(td.Id)
	gen.Emitf("<h3 id=%q>%s</h3>\n", strings.ToLower(s), s)
	gen.Emitf("<pre class=\"mknohighlight\"><code>\n")
	gen.Emitf("%s\n", gen.generateApiType(td))
	gen.Emitf("</code></pre>\n")
	return nil
}

func (gen *Generator) generateApiType(op *model.TypeDef) string {
	g := gen.getDetailGenerator()
	conf := data.NewObject()
	if g.Sorted() {
		conf.Put("sort", true) //!
	}
	err := g.Configure(gen.Schema, conf)
	if err != nil {
		return "Whoops: " + err.Error()
	}
	g.Begin()
	g.GenerateType(op)
	s := g.End()
	return s
}

func (gen *Generator) GenerateExceptions() {
	lst := gen.Exceptions()
	if len(lst) > 0 {
		gen.Emitf("<h2 id=\"exceptions\">Exceptions</h2>\n")
		for _, edef := range lst {
			gen.GenerateException(edef)
		}
	}
/*	emitted := make(map[model.AbsoluteIdentifier]*model.OperationOutput, 0)
	for _, op := range gen.Operations() {		
		for _, out := range op.Exceptions {
			if _, ok := emitted[out.Id]; ok {
				//duplicates?
			} else {
				if len(emitted) == 0 {
					gen.Emitf("<h2 id=\"exceptions\">Exceptions</h2>\n")
				}
				gen.GenerateException(out)
				emitted[out.Id] = out
			}
		}
		if len(emitted) > 0 {
			gen.Emit("\n")
		}
	}
*/	
}

func (gen *Generator) GenerateTypes() {
	tds := gen.Schema.Types
	//emitted := make(map[string]bool, 0)
	
	if len(tds) > 0 {
		gen.Emitf("<h2 id=\"types\">Types</h2>\n")
		for _, td := range gen.Types() {
			if !strings.HasPrefix(string(td.Id), "aws.protocols#") && !strings.HasPrefix(string(td.Id), "smithy.api#") {
				gen.GenerateType(td)
			}
		}
		gen.Emit("\n")
	}
}

