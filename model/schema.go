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

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	//	"github.com/boynton/data"
)

// Q: do I want to *require* a service? I think not. I use codegen for types all the time.
type Schema struct {
	ServiceDef
	Namespace Namespace `json:"-"`
	typeIndex map[AbsoluteIdentifier]*TypeDef
	opIndex   map[AbsoluteIdentifier]*OperationDef
	excIndex  map[AbsoluteIdentifier]*OperationOutput
	rezIndex  map[AbsoluteIdentifier]*ResourceDef
	//Metadata *data.Object `json:"metadata,omitempty"`
}

func Load(paths []string, tags []string) (*Schema, error) {
	if len(paths) != 1 {
		return nil, fmt.Errorf("Openapi import can aonly accept a single file")
	}
	path := paths[0]
	var schema *Schema
	var err error
	if strings.HasSuffix(path, ".json") {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("Cannot read API JSON file: %v\n", err)
		}
		err = json.Unmarshal(data, &schema)
		if err != nil {
			return nil, fmt.Errorf("Cannot parse API JSON file: %v\n", err)
		}
		if schema.Id == "" {
			return nil, fmt.Errorf("Cannot parse API JSON file: %v\n", err)
		}
		schema.Namespace = schema.ServiceNamespace()
	} else {
		schema, err = Parse(path)
		if err != nil {
			return nil, fmt.Errorf("Cannot parse API file: %v\n", err)
		}
	}
	//filter by tag?
	return schema, nil
}

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

func BaseTypeByName(name string) BaseType {
	var bt BaseType
	for i, n := range namesBaseType {
		if n == name {
			return BaseType(i)
		}
	}
	return bt
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
	//use schema.Namespace for generic use (i.e. no service)
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

func (schema *Schema) GetExceptionDef(id AbsoluteIdentifier) *OperationOutput {
	if schema.excIndex == nil {
		schema.excIndex = make(map[AbsoluteIdentifier]*OperationOutput, 0)
		for _, edef := range schema.Exceptions {
			schema.excIndex[edef.Id] = edef
		}
	}
	return schema.excIndex[id]
}

func (schema *Schema) AddExceptionDef(ed *OperationOutput) error {
	if schema.GetExceptionDef(ed.Id) != nil {
		return fmt.Errorf("Duplicate exception definition (merge NYI): %s", ed.Id)
	}
	schema.Exceptions = append(schema.Exceptions, ed)
	schema.excIndex[ed.Id] = ed
	return nil
}

func (schema *Schema) EnsureExceptionDef(e *OperationOutput) error {
	if schema.GetExceptionDef(e.Id) != nil {
		return nil
	}
	schema.Exceptions = append(schema.Exceptions, e)
	schema.excIndex[e.Id] = e
	return nil
}

func (td *TypeDef) String() string {
	return Pretty(td)
}

func (op *OperationDef) String() string {
	return Pretty(op)
}

func (in *OperationInput) String() string {
	return Pretty(in)
}

func (out *OperationOutput) String() string {
	return Pretty(out)
}

func (schema *Schema) BaseType(id AbsoluteIdentifier) BaseType {
	switch id {
	case "base#Blob":
		return BaseType_Blob
	case "base#Bool":
		return BaseType_Bool
	case "base#String":
		return BaseType_String
	case "base#Int8":
		return BaseType_Int8
	case "base#Int16":
		return BaseType_Int16
	case "base#Int32":
		return BaseType_Int32
	case "base#Int64":
		return BaseType_Int64
	case "base#Float32":
		return BaseType_Float32
	case "base#Float64":
		return BaseType_Float64
	case "base#Decimal":
		return BaseType_Decimal
	case "base#Integer":
		return BaseType_Integer
	case "base#Timestamp":
		return BaseType_Timestamp
	}
	td := schema.GetTypeDef(id)
	if td != nil {
		return td.Base
	}
	//not an explicit type, i.e. an operation input/output/error, all effectively structs
	return BaseType_Struct
}

func (schema *Schema) ShapeNames() []string {
	return nil //fixme
}

func (schema *Schema) ReplaceTypeDef(prev *TypeDef, next *TypeDef) error {
	ok := false
	for i, td := range schema.Types {
		if td == prev {
			schema.Types[i] = next
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("Schema.ReplaceTypeDef: no such type (%s)", prev.Id)
	}
	schema.typeIndex[next.Id] = next
	return nil
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

func (schema *Schema) AddResourceDef(rez *ResourceDef) error {
	if schema.GetResourceDef(rez.Id) != nil {
		return fmt.Errorf("Duplicate resource definition (merge NYI): %s", rez.Id)
	}
	schema.Resources = append(schema.Resources, rez)
	schema.rezIndex[rez.Id] = rez
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

func (td *TypeDef) Name() string {
	return StripNamespace(td.Id)
}

func (op *OperationDef) Name() string {
	return StripNamespace(op.Id)
}

func (oi *OperationInput) Name() string {
	return StripNamespace(oi.Id)
}

func (oo *OperationOutput) Name() string {
	return StripNamespace(oo.Id)
}

func StripNamespace(target AbsoluteIdentifier) string {
	t := string(target)
	n := strings.Index(t, "#")
	if n < 0 {
		return t
	}
	return t[n+1:]
}

func (schema *Schema) Namespaced(name string) AbsoluteIdentifier {
	for _, s := range namesBaseType {
		if name == s {
			return AbsoluteIdentifier("base#" + name)
		}
	}
	return AbsoluteIdentifier(string(schema.Namespace) + "#" + name)
}

func (schema *Schema) GetResourceDef(rezId AbsoluteIdentifier) *ResourceDef {
	if schema.rezIndex == nil {
		schema.rezIndex = make(map[AbsoluteIdentifier]*ResourceDef, 0)
		for _, rdef := range schema.Resources {
			schema.rezIndex[rdef.Id] = rdef
		}
	}
	return schema.rezIndex[rezId]
}

// filter out all but what the given resource depends on. A new Schema is returned
func (schema *Schema) ShakeResourceTree(rezId AbsoluteIdentifier, includeExceptions bool) *Schema {
	rez := schema.GetResourceDef(rezId)
	if rez == nil {
		return nil
	}
	newSchema := NewSchema()
	newSchema.Namespace = schema.Namespace
	newSchema.AddResourceDef(rez)
	for _, oid := range schema.ResourceOperations(rez) {
		op := schema.GetOperationDef(oid)
		newSchema.AddOperationDef(op)
		for _, tid := range schema.OperationTypes(op, includeExceptions) {
			if !schema.IsBaseType(tid) {
				td := schema.GetTypeDef(tid)
				newSchema.AddTypeDef(td)
			}
		}
		for _, eid := range op.Exceptions {
			e := schema.GetExceptionDef(eid)
			newSchema.AddExceptionDef(e)
		}
	}
	return newSchema
}

func (schema *Schema) ResourceOperations(rez *ResourceDef) []AbsoluteIdentifier {
	var ops []AbsoluteIdentifier
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

func (schema *Schema) typeReferences(tid AbsoluteIdentifier, referenced map[AbsoluteIdentifier]bool) {
	if !schema.IsBaseType(tid) {
		td := schema.GetTypeDef(tid)
		referenced[td.Id] = true
		if td.Items != "" {
			schema.typeReferences(td.Items, referenced)
		}
		if td.Keys != "" {
			schema.typeReferences(td.Keys, referenced)
		}
		if td.Items != "" {
			schema.typeReferences(td.Items, referenced)
		}
		if td.Fields != nil {
			for _, fd := range td.Fields {
				schema.fieldReferences(fd, referenced)
			}
		}
	}
}

func (schema *Schema) fieldReferences(fd *FieldDef, referenced map[AbsoluteIdentifier]bool) {
	referenced[fd.Type] = true
	if fd.Items != "" {
		schema.typeReferences(fd.Items, referenced)
	}
	if fd.Keys != "" {
		schema.typeReferences(fd.Keys, referenced)
	}
	if fd.Items != "" {
		schema.typeReferences(fd.Items, referenced)
	}
}

func (schema *Schema) inputReferences(fields []*OperationInputField, referenced map[AbsoluteIdentifier]bool) {
	for _, f := range fields {
		schema.typeReferences(f.Type, referenced)
		if f.Items != "" {
			schema.typeReferences(f.Items, referenced)
		}
		if f.Keys != "" {
			schema.typeReferences(f.Keys, referenced)
		}
		if f.Fields != nil {
			for _, fd := range f.Fields {
				schema.fieldReferences(fd, referenced)
			}
		}
	}
}

func (schema *Schema) outputReferences(fields []*OperationOutputField, referenced map[AbsoluteIdentifier]bool) {
	for _, f := range fields {
		schema.typeReferences(f.Type, referenced)
		if f.Items != "" {
			schema.typeReferences(f.Items, referenced)
		}
		if f.Keys != "" {
			schema.typeReferences(f.Keys, referenced)
		}
		if f.Fields != nil {
			for _, fd := range f.Fields {
				schema.fieldReferences(fd, referenced)
			}
		}
	}
}

func (schema *Schema) OperationTypes(op *OperationDef, includeExceptions bool) AbsoluteIdentifierList {
	referenced := make(map[AbsoluteIdentifier]bool, 0)
	if op.Input != nil {
		schema.inputReferences(op.Input.Fields, referenced)
	}
	if op.Output != nil {
		schema.outputReferences(op.Output.Fields, referenced)
	}
	if op.Exceptions != nil && includeExceptions {
		for _, eid := range op.Exceptions {
			e := schema.GetExceptionDef(eid)
			schema.outputReferences(e.Fields, referenced)
		}
	}
	var result AbsoluteIdentifierList
	for k := range referenced {
		result = append(result, k)
	}
	return result
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
	for name := range included {
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
	case BaseType_Struct, BaseType_Union:
		for _, f := range td.Fields {
			schema.noteDependencies(included, f.Type)
		}
		/*
			case Array:
					//could be *any*, do we need to mark that?
				case Object:
					//could be *any*, do we need to mark that?
		*/
	case BaseType_List:
		schema.noteDependencies(included, td.Items)
	case BaseType_Map:
		schema.noteDependencies(included, td.Items)
		schema.noteDependencies(included, td.Keys)
	default:
		//base types have no dependencies
	}
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

func (schema *Schema) IsStringType(id AbsoluteIdentifier) bool {
	bt := schema.BaseType(id)
	return bt == BaseType_String
}

func (schema *Schema) IsNumericType(id AbsoluteIdentifier) bool {
	bt := schema.BaseType(id)
	return bt == BaseType_Int32 || bt == BaseType_Int64 || bt == BaseType_Int16 || bt == BaseType_Int8 ||
		bt == BaseType_Float64 || bt == BaseType_Float32 //not decimal, it is an object
}

func (schema *Schema) IsBaseType(id AbsoluteIdentifier) bool {
	return strings.HasPrefix(string(id), "base#")
}

func IsSymbol(str string) bool {
	slen := len(str)
	s := []rune(str)
	if slen > 0 {
		if IsSymbolChar(s[0], true) {
			for i := 1; i < slen; i++ {
				if !IsSymbolChar(s[i], false) {
					return false
				}
			}
			return true
		}
	}
	return false
}
