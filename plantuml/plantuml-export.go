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
	suppressService    bool
	exceptionEntities map[model.AbsoluteIdentifier]bool
}

func (gen *Generator) Init(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.generateExceptions = config.GetBool("generate-exceptions")
	if !gen.generateExceptions {
		gen.exceptionEntities = make(map[model.AbsoluteIdentifier]bool, 0)
	}
	gen.suppressService = config.GetBool("suppress-service")
	gen.ns = string(schema.ServiceNamespace())
	gen.name = string(schema.ServiceName())
	return nil
}

func (gen *Generator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.generateExceptions = config.GetBool("generate-exceptions")
	if !gen.generateExceptions {
		gen.exceptionEntities = make(map[model.AbsoluteIdentifier]bool, 0)
	}
	gen.ns = string(schema.ServiceNamespace())
	gen.name = string(schema.ServiceName())
	gen.suppressService = config.GetBool("suppress-service")
	gen.Begin()
	gen.GenerateHeader()
	gen.GenerateService()
	gen.GenerateResources() //could use this metadata to determine whether Entity or Struct
	gen.GenerateOperations()
	gen.GenerateExceptions()
	gen.GenerateTypes()
	gen.GenerateFooter()
	s := gen.End()
	if gen.name == "" {
		gen.name = "model"
	}
	fname := gen.FileName(gen.name, ".puml")
	err = gen.Write(s, fname, "")
	return err
}

func (gen *Generator) GenerateHeader() {
	//	gen.Emitf("@startuml\nhide empty members\nset namespaceSeparator none\nskinparam linetype ortho\n\n")
	gen.Emitf("@startuml\nhide empty members\nset namespaceSeparator none\n\n")
}

func (gen *Generator) GenerateFooter() {
	gen.Emitf("@enduml\n")
}

func StripNamespace(target model.AbsoluteIdentifier) string {
	t := string(target)
	n := strings.Index(t, "#")
	if n < 0 {
		return t
	}
	return t[n+1:]
}

func (gen *Generator) ResourceOperations(rez *model.ResourceDef) []model.AbsoluteIdentifier {
	var ops []model.AbsoluteIdentifier
	if rez.Create != "" {
		ops = append(ops, rez.Create)
	}
	if rez.Read != "" {
		ops = append(ops, rez.Read)
	}
	if rez.Update != "" {
		ops = append(ops, rez.Update)
	}
	if rez.Delete != "" {
		ops = append(ops, rez.Delete)
	}
	if rez.List != "" {
		ops = append(ops, rez.List)
	}
	if rez.Operations != nil {
		for _, opId := range rez.Operations {
			ops = append(ops, opId)
		}
	}
	return ops
}

func (gen *Generator) GenerateResource(rez *model.ResourceDef) error {
	rezId := StripNamespace(rez.Id)
	connections := make(map[string]string, 0)
	//gen.Emitf("class %s<Resource> << (R,CadetBlue) >> {\n", rezId)
	gen.Emitf("class %s << (R,CadetBlue) >> {\n", rezId)
	if rez.Create != "" {
		dst := StripNamespace(rez.Create)
		gen.Emitf("    {field} <b>create</b>: %s\n", rezId)
		connections[dst] = rezId
	}
	if rez.Read != "" {
		dst := StripNamespace(rez.Read)
		gen.Emitf("    {field} <b>read</b>: %s\n", rezId)
		connections[dst] = rezId
	}
	if rez.Update != "" {
		dst := StripNamespace(rez.Update)
		gen.Emitf("    {field} <b>update</b>: %s\n", rezId)
		connections[dst] = rezId
	}
	if rez.Delete != "" {
		dst := StripNamespace(rez.Delete)
		gen.Emitf("    {field} <b>delete</b>: %s\n", rezId)
		connections[dst] = rezId
	}
	if rez.List != "" {
		dst := StripNamespace(rez.List)
		gen.Emitf("    {field} <b>list</b>: %s\n", rezId)
		connections[dst] = rezId
	}
	if rez.Operations != nil {
		var stripped []string
		for _, opId := range rez.Operations {
			strippedId := StripNamespace(opId)
			connections[strippedId] = rezId
			stripped = append(stripped, strippedId)
		}
		gen.Emitf("    {field} <b>operations</b>: [%s]\n", strings.Join(stripped, ","))
	}
	gen.Emitf("}\n")
	for dst, src := range connections {
		gen.Emitf("%s ..> %s\n", src, dst)
	}
	/*
	if gen.Schema.Id == "" {
		for _, op := range gen.Operations() {
			dst := StripNamespace(op.Id)
			if _, ok := connections[dst]; ok {
				panic("generate op from resource")
				gen.GenerateOperation(op)
			} else {
				fmt.Println(connections)
				panic("WTF?")
			}
		}
	} else {
		gen.GenerateOperations()
		//		for _, op := range gen.Operations() {
		//			gen.GenerateOperation(op)
		//		}
	}
	*/
	return nil
}

func (gen *Generator) GenerateResources() {	
	if len(gen.Schema.Resources) > 0 {
		for _, rez := range gen.Schema.Resources {
			gen.GenerateResource(rez)
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
		if e != nil {
			fref := StripNamespace(eid)
			if gen.generateExceptions {
				connections[fref] = opId
			} else {
				//note the payload. If it only is used by an exception, the omit it (the structure) also
				for _, f := range e.Fields {
					if f.HttpPayload {
						gen.exceptionEntities[f.Type] = true
					}
				}
			}
			gen.Emitf("    {field} %d: %s\n", e.HttpStatus, fref)
		}
	}
	gen.Emitf("}\n")
	for link, s := range connections {
		gen.Emitf("%s ..> %s\n", s, link)
	}
	gen.Emitf("\n")
	return nil
}

//if service has resources, iterate through those first, noting the ops that they handle
//then, iterate through operations, adding at top level those that have not been added
func (gen *Generator) GenerateService() {
	if gen.suppressService || gen.Schema.Id == "" {
		return
	}
	src := StripNamespace(gen.Schema.Id)
	//gen.Emitf("interface %s<Service>\n", src)
	gen.Emitf("interface %s\n", src)
	targets := make(map[string]string, 0)
	opTargets := make(map[model.AbsoluteIdentifier]string, 0)
	if len(gen.Schema.Resources) > 0 {
		if len(gen.Schema.Resources) > 0 {
			for _, rez := range gen.Schema.Resources {
				dst := StripNamespace(rez.Id)
				if _, ok := targets[dst]; !ok {
					targets[dst] = src
				}
				for _, oid := range gen.ResourceOperations(rez) {
					opTargets[oid] = StripNamespace(oid)
				}
			}
		}
	}
	for _, op := range gen.Operations() {
		if _, ok := opTargets[op.Id]; !ok {
			targets[StripNamespace(op.Id)] = src
		}
	}
	for dst, src := range targets {
		gen.Emitf("%s ..> %s\n", src, dst)
	}
}

func (gen *Generator) GenerateOperations() {
	if len(gen.Schema.Operations) > 0 {
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
	if gen.exceptionEntities != nil {
		if _, ok := gen.exceptionEntities[td.Id]; ok {
			return nil
		}
	}
    s := StripNamespace(td.Id)
	connections := make(map[string]string, 0)
	targets := make(map[string]string, 0)
    switch td.Base {
    case model.BaseType_Struct:
		gen.Emitf("class %s {\n", s)
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
	if gen.generateExceptions {
		lst := gen.Exceptions()
		if len(lst) > 0 {
			for _, edef := range lst {
				gen.GenerateException(edef)
			}
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

