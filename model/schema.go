/*
Copyright 2022 Lee R. Boynton

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

import(
	"fmt"
	"strings"
)


func (ident Identifier) Capitalized() string {
	s := string(ident)
	if s == "" {
		return s
    }
    return strings.ToUpper(s[0:1]) + s[1:]
}

func (ident Identifier) Uncapitalized() string {
	s := string(ident)
	if s == "" {
		return s
    }
    return strings.ToLower(s[0:1]) + s[1:]
}

type Schema struct {
	ServiceDef
	typeIndex map[AbsoluteIdentifier]*TypeDef
	opIndex map[AbsoluteIdentifier]*OperationDef
}

func BaseTypeByName(name string) BaseType {
	for i, n := range namesBaseType {
		if n == name {
			return BaseType(i)
		}
	}
	panic("Bad base type name: " + name)
}

func (schema *Schema) String() string {
	return Pretty(schema)
}

func (schema *Schema) ServiceName() Identifier {
	if schema.Id == "" {
		return ""
	}
	lst := strings.Split(string(schema.Id), "#")
	return Identifier(lst[1])
}

func (schema *Schema) ServiceNamespace() Namespace {
	if schema.Id == "" {
		return ""
	}
	lst := strings.Split(string(schema.Id), "#")
	return Namespace(lst[0])
}

func NewSchema() *Schema {
	s := &Schema{
		ServiceDef: ServiceDef{},
	}
	return s
}

func (schema *Schema) GetTypeDef(id AbsoluteIdentifier) *TypeDef {
	if schema.typeIndex == nil {
		schema.typeIndex = make(map[AbsoluteIdentifier]*TypeDef, 0)
		for _, td := range schema.Types {
			schema.typeIndex[td.Id] = td
		}
	}
	return schema.typeIndex[id]
}

func (schema *Schema) BaseType(id AbsoluteIdentifier) BaseType {
	td := schema.GetTypeDef(id)
	if td != nil {
		return td.Base
	}
	//not an explicit type, i.e. an operation input/output/error, all effectively structs
	return Struct
}

func (schema *Schema) ShapeNames() []string {
	return nil //fixme
}

func (schema *Schema) AddTypeDef(td *TypeDef) error {
	if schema.GetTypeDef(td.Id) != nil {
		return fmt.Errorf("Duplicate type definition (merge NYI): %s", td.Id)
	}
	schema.Types = append(schema.Types, td)
	schema.typeIndex[td.Id] = td
	return nil
}

func (schema *Schema) GetOperationDef(id AbsoluteIdentifier) *OperationDef {
	if schema.opIndex == nil {
		schema.opIndex = make(map[AbsoluteIdentifier]*OperationDef, 0)
		for _, op := range schema.Operations {
			schema.opIndex[op.Id] = op
		}
	}
	return schema.opIndex[id]
}

func (schema *Schema) AddOperationDef(op *OperationDef) error {
	if schema.GetOperationDef(op.Id) != nil {
		return fmt.Errorf("Duplicate operation definition (merge NYI): %s", op.Id)
	}
	schema.Operations = append(schema.Operations, op)
	schema.opIndex[op.Id] = op
	return nil
}

func (schema *Schema) Merge(another *Schema) error {
	if schema.Id == "" {
		*schema = *another
	} else {
		return fmt.Errorf("Merge two non-empty models NYI")
	}
	return nil
}

func SliceContainsString(ary []string, val string) bool {
	for _, s := range ary {
		if s == val {
			return true
		}
	}
	return false
}

func (schema *Schema) Filter(tags []string) {
	var root []AbsoluteIdentifier
	for _, td := range schema.Types {
		if td.Tags != nil {
			for _, t := range td.Tags {
				if SliceContainsString(tags, t) {
					root = append(root, td.Id)
				}
			}
		}
	}
	included := make(map[AbsoluteIdentifier]bool, 0)
	for _, k := range root {
		if _, ok := included[k]; !ok {
			schema.noteDependencies(included, k)
		}
	}
	var filtered []*TypeDef
	for name, _ := range included {
		if !strings.HasPrefix(string(name), "api#") {
			filtered = append(filtered, schema.GetTypeDef(name))
		}
	}
	schema.Types = filtered
}

func (schema *Schema) noteDependencies(included map[AbsoluteIdentifier]bool, id AbsoluteIdentifier) {
	if id == "" {
		return
	}
	
	if _, ok := included[id]; ok {
		return
	}
	included[id] = true
	td := schema.GetTypeDef(id)
	if td == nil {
		return
	}
	switch td.Base {
	case Struct, Union:
		for _, f := range td.Fields {
			schema.noteDependencies(included, f.Type)
		}
	case Array:
		//could be *any*, do we need to mark that?
	case Object:
		//could be *any*, do we need to mark that?
	case List:
		schema.noteDependencies(included, td.Items)
	case Map:
		schema.noteDependencies(included, td.Items)
		schema.noteDependencies(included, td.Keys)
	default:
		//base types have no dependencies
	}
}

func (schema *Schema) Validate() error {
	//todo fix
	return nil
}

func (od *OperationDef) OutputHttpPayloadName() string {
	if od.Output != nil {
		for _, o := range od.Output.Fields {
			if o.HttpPayload {
				return string(o.Name)
			}
		}
	}
	return ""
}
