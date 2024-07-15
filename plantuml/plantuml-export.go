/*
   Copyright 2024 Lee R. Boynton

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
package plantuml

import (
	"fmt"
	"strings"
	
	"github.com/boynton/api/model"
	"github.com/boynton/api/smithy"
	"github.com/boynton/data"
)

const IndentAmount = "    "

const Association = "--"
const Aggregation = "o"
const Composition = "*"
const ManySrc = "}"
const ExactlyOne = "||"
const ZeroOrManySrc = "}o"
const OneOrManySrc = "}|"
const ManyDst = "{"
const ZeroOrManyDst = "o{"
const OneOrManyDst = "|{"

type Generator struct {
	model.BaseGenerator
	ns              string
	name            string
	detailGenerator string
	generateExceptions bool
	entities map[string]string
}

func (gen *Generator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.generateExceptions = config.GetBool("generateExceptions")
	gen.ns = string(schema.ServiceNamespace())
	gen.name = string(schema.ServiceName())
	gen.entities = make(map[string]string, 0)
	for _, op := range gen.Operations() {
		//		if op.Resource != ""  && op.Lifecycle == "read" {
		if op.HttpMethod == "GET" {
			entityType := ""
			for _, out := range op.Output.Fields {
				if out.HttpPayload {
					entityType = StripNamespace(out.Type)
				}
			}
			idField := ""
			if op.Input != nil {
				for _, in := range op.Input.Fields {
					if in.HttpPath {
						idField = string(in.Name) 
						break //fixme: more than one pathparam for a resource id
					}
				}
			}
			if idField == "" {
				idField = "-"
			}
			gen.entities[entityType] = idField
		}
	}
	gen.Begin()
	gen.GenerateHeader()
	//FIX gen.GenerateResources() //could use this metadata to determine whether Entity or Struct
	gen.GenerateOperations()
	if gen.generateExceptions {
		gen.GenerateExceptions()
	}
	gen.GenerateTypes()
	gen.GenerateFooter()
	s := gen.End()
	fname := gen.FileName(gen.name, ".md")
	err = gen.Write(s, fname, "")
	return err
}

func (gen *Generator) ResourceIds() []model.AbsoluteIdentifier {
	var resources []model.AbsoluteIdentifier
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
	return resources
}

/*
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
*/

func (gen *Generator) GenerateHeader() {
	//	gen.Emitf("@startuml\nhide empty members\nset namespaceSeparator none\nskinparam linetype ortho\n\n")
	gen.Emitf("@startuml\nhide empty members\nset namespaceSeparator none\n\n")
}

func (gen *Generator) GenerateFooter() {
	gen.Emitf("@enduml\n")
}

func StripNamespace(target model.AbsoluteIdentifier) string {
	fmt.Printf("StripNamespace: %q\n", target)
	t := string(target)
	n := strings.Index(t, "#")
	if n < 0 {
		return t
	}
	return t[n+1:]
}


func (gen *Generator) GenerateResource(id model.AbsoluteIdentifier) error {
	rezId := StripNamespace(id)
	gen.Emitf("class %s<Resource> {\n", rezId)
	gen.Emitf("}\n")
	return nil
}

func (gen *Generator) GenerateResources() {
	resourceIds := gen.ResourceIds()
	if len(resourceIds) > 0 {
		for _, id := range resourceIds {
			gen.GenerateResource(id)
		}
	}
}

func (gen *Generator) GenerateOperation(op *model.OperationDef) error {
	opId := StripNamespace(op.Id)
	connections := make(map[string]string, 0)
	//	gen.Emitf("class %s<Operation> << (O,DarkSalmon) >> {\n", opId)
	gen.Emitf("class %s << (O,DarkSalmon) >> {\n", opId)
	gen.Emitf("<b>%s %s</b>\n", op.HttpMethod, op.HttpUri)
	//gen.Emitf("{static} %s %s\n", op.HttpMethod, op.HttpUri)
	if op.Input != nil {
		gen.Emitf(".. <i>inputs</i> ..\n")
		for _, f := range op.Input.Fields {
			where := ""
			if f.HttpHeader != "" {
				where = "header=\"" + f.HttpHeader + "\""
			} else if f.HttpQuery != "" {
				where = "query=\"" + string(f.HttpQuery)  + "\""
			} else if f.HttpPath {
				where = "path"
			} else if f.HttpPayload {
				//where = "payload"
			}
			fref, link := gen.GenerateTypeRef(f.Type)
			if len(fref) <= 1 {
				fmt.Printf("fref: %q, %q\n", fref, f.Type)
				panic("what?!")
			}
			if link != "" {
				connections[link] = opId
			}
			if where != "" {
				where = " <i>(" + where + ")</i>"
			}
			if f.Required {
				//exactly 1 to target
				gen.Emitf("    {field} <b>%s</b>: %s%s\n", f.Name, fref, where)
			} else {
				//0 or 1 to target
				gen.Emitf("    {field} %s: %s%s\n", f.Name, fref, where)
			}
		}
	}
	if op.Output != nil {
		gen.Emitf(".. <i>responses</i>..\n")
		if len(op.Output.Fields) == 0 {
			gen.Emitf("   {field} <b>%d: <no content></b>\n", op.Output.HttpStatus)
		} else {
			for _, f := range op.Output.Fields {
				if f.HttpPayload {
					fref, link := gen.GenerateTypeRef(f.Type)
					if link != "" {
						connections[link] = opId
					}
					gen.Emitf("   {field} <b>%d</b>: %s\n", op.Output.HttpStatus, fref)
				}
			}
			/*		for _, f := range op.Output.Fields {
			if !f.HttpPayload {
				fref, link := gen.GenerateTypeRef(f.Type)
				if link != "" {
					connections = append(connections, fmt.Sprintf("%s ..> %s", opId, link))
				}
				gen.Emitf("    {field} %s: %s (header=\n", f.Name, fref)
			}
		}
			*/
		}
	}
	
	for _, eid := range op.Exceptions {
		e := gen.Schema.GetExceptionDef(eid)
		fref := StripNamespace(eid)
		if gen.generateExceptions {
			connections[fref] = opId
		}
		gen.Emitf("    {field} %d: %s\n", e.HttpStatus, fref)
	}
	gen.Emitf("}\n")
	for link, s := range connections {
		gen.Emitf("%s ..> %s\n", s, link)
	}
	gen.Emitf("\n")
	return nil
}

func (gen *Generator) GenerateOperations() {
	if len(gen.Schema.Operations) > 0 {
		s := StripNamespace(gen.Schema.Id)
		if s != "" {
			//gen.Emitf("class %s<Service> << (S,khaki) >>\n", s)
			gen.Emitf("interface %s<Service>\n", s)
			for _, op := range gen.Operations() {
				gen.Emitf("%s ..> %s\n", s, StripNamespace(op.Id))
			}
		}
		for _, op := range gen.Operations() {
			gen.GenerateOperation(op)
		}
		gen.Emit("\n")
	}
}

func (gen *Generator) GenerateException(exc *model.OperationOutput) error {
	eId := StripNamespace(exc.Id)
	connections := make(map[string]string, 0)
	gen.Emitf("class %s<Exception> {\n", eId)
	for _, f := range exc.Fields {
		if f.HttpPayload {
			fref, link := gen.GenerateTypeRef(f.Type)
			if link != "" {
				connections[link] = eId
			}
			gen.Emitf("   {field} <b>%d: %s</b>\n", exc.HttpStatus, fref)
		}
	}
	gen.Emitf("}\n")
	for link, s := range connections {
		gen.Emitf("%s ..> %s\n", s, link)
	}
	gen.Emitf("\n")
	return nil
}

func (gen *Generator) GenerateTypeRef(ref model.AbsoluteIdentifier) (string, string) {
	sref := StripNamespace(ref)
	if !gen.Schema.IsBaseType(ref) {
		td := gen.Schema.GetTypeDef(ref)
		if td != nil {
			switch td.Base {
			case model.BaseType_List:
				s, link := gen.GenerateTypeRef(td.Items)
				return fmt.Sprintf("List<%s>", s), link
			case model.BaseType_Struct, model.BaseType_Union:
				return sref, sref
			case model.BaseType_Enum:
				return sref, sref
			default:
				return td.Base.String(), ""
			}
		}
	}
    return sref, ""
}

func (gen *Generator) GenerateType(td *model.TypeDef) error {
    s := StripNamespace(td.Id)
	connections := make(map[string]string, 0)
	targets := make(map[string]string, 0)
	entityIdField := ""
	if eid, ok := gen.entities[s]; ok {
		entityIdField = eid
	}
    switch td.Base {
    case model.BaseType_Struct:
		if entityIdField != "" {
			gen.Emitf("class %s << (R,CadetBlue) >> {\n", s)
		} else {
			gen.Emitf("class %s {\n", s)
		}
		for _, f := range td.Fields {
            fname := string(f.Name)
            fref, link := gen.GenerateTypeRef(f.Type)
			if link != "" {
				connections[link] = s
				if strings.HasPrefix(fref, "List<") {
					if f.Required {
						//should really check the type for length traits. Even a required list can be empty
						targets[link] = Composition + Association + OneOrManyDst
					} else {
						targets[link] = Composition + Association + ManyDst
					}
					//targets[link] = "\"1\" --> \"*\""
				}
			}
			if f.Required {
				gen.Emitf("    {field} <b>%s</b>: %s\n", fname, fref)
			} else {
				gen.Emitf("    {field} %s: %s\n", fname, fref)
			}
		}
		gen.Emitf("}\n")
    case model.BaseType_Union:
		//gen.Emitf("class %s<Union> {\n", s)
		gen.Emitf("class %s << (U,CornflowerBlue) >>{\n", s)
		for _, f := range td.Fields {
            fname := string(f.Name)
            fref, link := gen.GenerateTypeRef(f.Type)
			if link != "" {
				connections[link] = s
				if strings.HasPrefix(fref, "List<") {
					targets[link] = Composition + Association + ManyDst
				}
			}
            gen.Emitf("    {field} <b>%s</b>: %s\n", fname, fref)
		}
		gen.Emitf("}\n")
	case model.BaseType_Enum:
		gen.Emitf("enum %s {\n", s)
		for _, el := range td.Elements {
            gen.Emitf("    {field} %s\n", el.Symbol)
		}
		gen.Emitf("}\n")
	}
	for link, s := range connections {
		//		gen.Emitf("%s --> %s\n", s, link)
		arrow := Composition + Association
		if a, ok := targets[link]; ok {
			arrow = a
		}
		gen.Emitf("%s %s %s\n", s, arrow, link)
	}
    return nil
}

func (gen *Generator) GenerateExceptions() {
	lst := gen.Exceptions()
	if len(lst) > 0 {
		for _, edef := range lst {
			gen.GenerateException(edef)
		}
	}
}

func (gen *Generator) GenerateTypes() {
	tds := gen.Schema.Types
	//emitted := make(map[string]bool, 0)
	
	if len(tds) > 0 {
		for _, td := range gen.Types() {
			gen.GenerateType(td)
		}
		gen.Emit("\n")
	}
}

