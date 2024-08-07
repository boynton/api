/*
Copyright 2021 Lee R. Boynton

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
package smithy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/boynton/data" //for Decimal
)

const UnspecifiedNamespace = "example"
const UnspecifiedVersion = "0.0"

type AST struct {
	Smithy   string       `json:"smithy"`
	Metadata *NodeValue   `json:"metadata,omitempty"`
	Shapes   *Map[*Shape] `json:"shapes,omitempty"`
}

func jsonEncode(obj interface{}) string {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(&obj); err != nil {
		return fmt.Sprint(obj)
	}
	return strings.TrimRight(string(buf.String()), " \t\n\v\f\r")
}

func jsonDecode(j string) interface{} {
	var tmp interface{}
	err := json.Unmarshal([]byte(j), &tmp)
	if err != nil {
		return nil
	}
	return tmp
}

func clone(o interface{}) interface{} {
	return jsonDecode(jsonEncode(o))
}

type NodeValue struct {
	value interface{}
}

func NewNodeValue() *NodeValue {
	return &NodeValue{value: make(map[string]interface{}, 0)}
}

func AsNodeValue(v interface{}) *NodeValue {
	if nv, ok := v.(*NodeValue); ok {
		return nv
	}
	return &NodeValue{value: v}
}

func (node NodeValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(node.value)
}

func (node *NodeValue) UnmarshalJSON(b []byte) error {
	var v interface{}
	err := json.Unmarshal(b, &v)
	if err == nil {
		node.value = v
	}
	return err
}

func (node *NodeValue) Clone() *NodeValue {
	if node.value == nil {
		return &NodeValue{}
	}
	return &NodeValue{value: clone(node.value)}
}

func cloneNodeValue(node *NodeValue) *NodeValue {
	if node == nil {
		return nil
	}
	return node.Clone()
}

func (node *NodeValue) RawValue() interface{} {
	return node.value
}

func (node *NodeValue) String() string {
	return fmt.Sprint(node.value)
}

func (node *NodeValue) IsObject() bool {
	switch node.value.(type) {
	case map[string]interface{}:
		return true
	case *data.Object:
		return true
	}
	return false
}

func (node *NodeValue) Keys() []string {
	switch val := node.value.(type) {
	case *data.Object:
		return val.Keys()
	case map[string]interface{}:
		var keys []string
		for k := range val {
			keys = append(keys, k)
		}
		return keys
	default:
		panic("Whoa, Keys()")
	}
}

func (node *NodeValue) Has(key string) bool {
	if node != nil && node.value != nil {
		switch m := node.value.(type) {
		case map[string]interface{}:
			if _, ok := m[key]; ok {
				return true
			}
		case *data.Object:
			return m.Has(key)
		}
	}
	return false
}

func (node *NodeValue) Get(key string) *NodeValue {
	if node == nil {
		return nil
	}
	switch m := node.value.(type) {
	case map[string]interface{}:
		if tmp, ok := m[key]; ok {
			return AsNodeValue(tmp)
		}
		return nil
	case *data.Object:
		return AsNodeValue(m.Get(key))
	case *NodeValue:
		return m.Get(key)
	default:
		return nil
	}
}

func (node *NodeValue) AsString() string {
	if node == nil || node.value == nil {
		return ""
	}
	switch s := node.value.(type) {
	case string:
		return s
	}
	return ""
}

func (node *NodeValue) GetBool(key string) bool {
	return node.Get(key).AsBool()
}

func (node *NodeValue) AsBool() bool {
	if node == nil {
		return false
	}
	if node.value != nil {
		switch b := node.value.(type) {
		case bool:
			return b
		case *bool:
			return *b
		default:
			return true
		}
	}
	return false
}

func (node *NodeValue) GetString(key string) string {
	return node.Get(key).AsString()
}

func (node *NodeValue) AsInt() int {
	return int(node.AsInt64())
}

func (node *NodeValue) AsInt64() int64 {
	if node.value != nil {
		switch n := node.value.(type) {
		case int:
			return int64(n)
		case *int:
			return int64(*n)
		case int64:
			return n
		case *int64:
			return *n
		case float64:
			return int64(n)
		case *float64:
			return int64(*n)
		case *data.Integer:
			return n.AsInt64()
		case *data.Decimal:
			return n.AsInt64()
		case *NodeValue:
			panic("double NodeValue wrapper, oops")
		}
	}
	return 0
}

func Kind(v interface{}) string {
	return fmt.Sprintf("%v", reflect.ValueOf(v).Kind())
}

func (node *NodeValue) GetInt(key string, def int) int {
	n := node.Get(key)
	if n == nil {
		return def
	}
	return n.AsInt()
}

func (node *NodeValue) GetInt64(key string, def int64) int64 {
	n := node.Get(key)
	if n == nil {
		return def
	}
	return n.AsInt64()
}

func (node *NodeValue) GetDecimal(key string, def *data.Decimal) *data.Decimal {
	n := node.Get(key)
	if n == nil {
		return def
	}
	return n.AsDecimal()
}

func (node *NodeValue) AsDecimal() *data.Decimal {
	if node == nil {
		return nil
	}
	if node.value != nil {
		switch n := node.value.(type) {
		case *data.Decimal:
			return n
		case data.Decimal:
			return &n
		case float64:
			return data.DecimalFromFloat64(n)
		case *NodeValue:
			panic("ooops, double wrapoper")
		}
	}
	return nil
}

func (node *NodeValue) GetSlice(key string) []interface{} {
	n := node.Get(key)
	if n == nil {
		return nil
	}
	switch v := n.value.(type) {
	case []interface{}:
		return v
	default:
		panic("Whoa, GetSlice()")
	}
}

func (node *NodeValue) GetStringSlice(key string) []string {
	switch m := node.value.(type) {
	case map[string]interface{}:
		if tmp, ok := m[key]; ok {
			if a, ok := tmp.([]interface{}); ok {
				var vals []string
				for _, v := range a {
					switch s := v.(type) {
					case string:
						vals = append(vals, s)
					default:
						panic("Whoa, not string in slice")
					}
				}
				return vals
			}
		}
		return nil
	default:
		panic("Whoa, GetStringSlice()")
	}
}

func (node *NodeValue) Length() int {
	switch m := node.value.(type) {
	case map[string]interface{}:
		return len(m)
	case []interface{}:
		return len(m)
	case *data.Object:
		return len(m.Bindings())
	default:
		return -1
	}
}

func (node *NodeValue) Put(key string, val interface{}) *NodeValue {
	switch m := node.value.(type) {
	case map[string]interface{}:
		m[key] = val
	case *data.Object:
		m.Put(key, val)
	default:
		panic("Whoa, Put()")
	}
	return node
}

func (ast *AST) AssemblyVersion() int {
	if strings.HasPrefix(ast.Smithy, "1") {
		return 1
	}
	return 2
}

func (ast *AST) PutShape(id string, shape *Shape) {
	if ast.Shapes == nil {
		ast.Shapes = NewMap[*Shape]()
	}
	ast.Shapes.Put(id, shape)
}

func (ast *AST) GetShape(id string) *Shape {
	if ast.Shapes == nil {
		return nil
	}
	shape := ast.Shapes.Get(id)
	if shape == nil {
		if IsPreludeType(id) {
			fmt.Println("WHOA: id not present, yet is a prelude shape:", id)
			panic("here")
		}
	}
	return shape
}

func (ast *AST) ForAllShapes(visitor func(shapeId string, shape *Shape) error) error {
	for _, shapeId := range ast.Shapes.Keys() {
		shape := ast.GetShape(shapeId)
		err := visitor(shapeId, shape)
		if err != nil {
			return err
		}
	}
	return nil
}

func cloneShape(shape *Shape) *Shape {
	newShape := &Shape{
		Type:                 shape.Type,
		Version:              shape.Version,
		Member:               cloneMember(shape.Member),
		Members:              cloneMembers(shape.Members),
		Mixins:               cloneShapeRefs(shape.Mixins),
		Key:                  cloneMember(shape.Key),
		Value:                cloneMember(shape.Value),
		Create:               cloneShapeRef(shape.Create),
		Put:                  cloneShapeRef(shape.Put),
		Read:                 cloneShapeRef(shape.Read),
		Update:               cloneShapeRef(shape.Update),
		Delete:               cloneShapeRef(shape.Delete),
		List:                 cloneShapeRef(shape.List),
		CollectionOperations: cloneShapeRefs(shape.CollectionOperations),
		Operations:           cloneShapeRefs(shape.Operations),
		Resources:            cloneShapeRefs(shape.Resources),
		Input:                cloneShapeRef(shape.Input),
		Output:               cloneShapeRef(shape.Output),
		Errors:               cloneShapeRefs(shape.Errors),
		Traits:               cloneNodeValue(shape.Traits),
	}
	if shape.Identifiers != nil {
		m := NewMap[*ShapeRef]()
		for _, k := range shape.Identifiers.Keys() {
			sr := shape.Identifiers.Get(k)
			m.Put(k, cloneShapeRef(sr))
		}
		newShape.Identifiers = m
	}
	return newShape
}

func cloneShapeRef(ref *ShapeRef) *ShapeRef {
	if ref != nil {
		return &ShapeRef{
			Target: ref.Target,
		}
	}
	return nil
}

func cloneShapeRefs(refs []*ShapeRef) []*ShapeRef {
	var res []*ShapeRef
	for _, r := range refs {
		res = append(res, cloneShapeRef(r))
	}
	return res
}

func cloneMember(member *Member) *Member {
	if member != nil {
		return &Member{
			Target: member.Target,
			Traits: cloneNodeValue(member.Traits),
		}
	}
	return nil
}

func cloneMembers(members *Map[*Member]) *Map[*Member] {
	if members != nil {
		mems := NewMap[*Member]()
		for _, k := range members.Keys() {
			m := cloneMember(members.Get(k))
			mems.Put(k, m)
		}
		return mems
	}
	return nil
}

type Shape struct {
	Type string `json:"type"`

	//Service
	Version string `json:"version,omitempty"`

	//List and Set
	Member *Member `json:"member,omitempty"`

	//Map
	Key   *Member `json:"key,omitempty"`
	Value *Member `json:"value,omitempty"`

	//Structure and Union
	Members *Map[*Member] `json:"members,omitempty"` //keys must be case-insensitively unique. For union, len(Members) > 0,
	Mixins  []*ShapeRef   `json:"mixins,omitempty"`  //mixins for the shape

	//Resource
	Identifiers *Map[*ShapeRef] `json:"identifiers,omitempty"`

	Create               *ShapeRef   `json:"create,omitempty"`
	Put                  *ShapeRef   `json:"put,omitempty"`
	Read                 *ShapeRef   `json:"read,omitempty"`
	Update               *ShapeRef   `json:"update,omitempty"`
	Delete               *ShapeRef   `json:"delete,omitempty"`
	List                 *ShapeRef   `json:"list,omitempty"`
	CollectionOperations []*ShapeRef `json:"collectionOperations,omitempty"`

	//Resource and Service
	Operations []*ShapeRef `json:"operations,omitempty"`
	Resources  []*ShapeRef `json:"resources,omitempty"`

	//Operation
	Input  *ShapeRef   `json:"input,omitempty"`
	Output *ShapeRef   `json:"output,omitempty"`
	Errors []*ShapeRef `json:"errors,omitempty"`

	Traits *NodeValue `json:"traits,omitempty"` //service, resource, operation, apply
}

func (shape *Shape) GetTrait(id string) *NodeValue {
	if shape.Traits != nil {
		return shape.Traits.Get(id)
	}
	return nil
}

func (shape *Shape) GetStringTrait(id string) string {
	if shape.Traits != nil {
		return shape.Traits.GetString(id)
	}
	return ""
}

type ShapeRef struct {
	Target string `json:"target"`
}

type Member struct {
	Target string     `json:"target"`
	Traits *NodeValue `json:"traits,omitempty"`
}

func (mem *Member) GetStringTrait(id string) string {
	if mem.Traits != nil {
		return mem.Traits.GetString(id)
	}
	return ""
}

func shapeIdNamespace(id string) string {
	//name.space#entity$member
	lst := strings.Split(id, "#")
	return lst[0]
}

func (ast *AST) Validate() error {
	alreadyChecked := make(map[string]*Shape, 0)
	for _, id := range ast.Shapes.Keys() {
		err := ast.ValidateDefined(id, alreadyChecked)
		if err != nil {
			return err
		}
	}
	return nil
}

// check that all references are defined in this assembly
func (ast *AST) ValidateDefined(id string, alreadyChecked map[string]*Shape) error {
	if _, ok := alreadyChecked[id]; ok {
		return nil
	}
	if ast.isSmithyType(id) {
		return nil
	}
	shape := ast.Shapes.Get(id)
	if shape == nil {
		return fmt.Errorf("Shape not defined: %s", id)
	}
	alreadyChecked[id] = shape
	switch shape.Type {
	case "structure", "union":
		for _, fname := range shape.Members.Keys() {
			fval := shape.Members.Get(fname)
			ftype := fval.Target
			err := ast.ValidateDefined(ftype, alreadyChecked)
			if err != nil {
				return err
			}
		}
	case "list":
		err := ast.ValidateDefined(shape.Member.Target, alreadyChecked)
		if err != nil {
			return err
		}
	case "map":
		err := ast.ValidateDefined(shape.Key.Target, alreadyChecked)
		if err != nil {
			return err
		}
		err = ast.ValidateDefined(shape.Value.Target, alreadyChecked)
		if err != nil {
			return err
		}
	default:
		//ok
	}
	return nil
}

func (ast *AST) isSmithyType(name string) bool {
	return strings.HasPrefix(name, "smithy.api#")
}

func (ast *AST) Namespaces() []string {
	m := make(map[string]int, 0)
	if ast.Shapes != nil {
		for _, id := range ast.Shapes.Keys() {
			ns := shapeIdNamespace(id)
			if n, ok := m[ns]; ok {
				m[ns] = n + 1
			} else {
				m[ns] = 1
			}
		}
	}
	nss := make([]string, 0, len(m))
	for k := range m {
		nss = append(nss, k)
	}
	return nss
}

func (ast *AST) RequiresDocumentType() bool {
	included := NewMap[bool]()
	for _, k := range ast.Shapes.Keys() {
		ast.noteDependencies(included, k)
	}
	if included.Has("smithy.api#Document") {
		return true
	}
	return false
}

func (ast *AST) noteDependenciesFromRef(included *Map[bool], ref *ShapeRef) {
	if ref != nil {
		ast.noteDependencies(included, ref.Target)
	}
}

func (ast *AST) noteDependencies(included *Map[bool], name string) {
	//note traits
	if name == "smithy.api#Document" {
		included.Put(name, true)
		return
	}
	if name == "" || strings.HasPrefix(name, "smithy.api#") {
		return
	}
	if included.Has(name) {
		return
	}
	included.Put(name, true)
	shape := ast.GetShape(name)
	if shape == nil {
		return
	}
	if shape.Traits != nil {
		for _, tk := range shape.Traits.Keys() {
			ast.noteDependencies(included, tk)
		}
	}
	switch shape.Type {
	case "service":
		for _, o := range shape.Operations {
			ast.noteDependenciesFromRef(included, o)
		}
		for _, r := range shape.Resources {
			ast.noteDependenciesFromRef(included, r)
		}
	case "operation":
		ast.noteDependenciesFromRef(included, shape.Input)
		ast.noteDependenciesFromRef(included, shape.Output)
		for _, e := range shape.Errors {
			ast.noteDependenciesFromRef(included, e)
		}
	case "resource":
		if shape.Identifiers != nil {
			for _, k := range shape.Identifiers.Keys() {
				v := shape.Identifiers.Get(k)
				ast.noteDependencies(included, v.Target)
			}
		}
		for _, o := range shape.Operations {
			ast.noteDependenciesFromRef(included, o)
		}
		for _, r := range shape.Resources {
			ast.noteDependenciesFromRef(included, r)
		}
		ast.noteDependenciesFromRef(included, shape.Create)
		ast.noteDependenciesFromRef(included, shape.Put)
		ast.noteDependenciesFromRef(included, shape.Read)
		ast.noteDependenciesFromRef(included, shape.Update)
		ast.noteDependenciesFromRef(included, shape.Delete)
		ast.noteDependenciesFromRef(included, shape.List)
		for _, o := range shape.CollectionOperations {
			ast.noteDependenciesFromRef(included, o)
		}
	case "structure", "union":
		for _, n := range shape.Members.Keys() {
			m := shape.Members.Get(n)
			ast.noteDependencies(included, m.Target)
		}
	case "list", "set":
		ast.noteDependencies(included, shape.Member.Target)
	case "map":
		ast.noteDependencies(included, shape.Key.Target)
		ast.noteDependencies(included, shape.Value.Target)
	case "string", "integer", "long", "short", "byte", "float", "double", "boolean", "bigInteger", "bigDecimal", "blob", "timestamp":
		//smithy primitives
	}
}

func (ast *AST) ShapeNames() []string {
	var lst []string
	for _, k := range ast.Shapes.Keys() {
		lst = append(lst, k)
	}
	return lst
}

func LoadAST(path string) (*AST, error) {
	var ast *AST
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot read smithy AST file: %v\n", err)
	}
	err = json.Unmarshal(data, &ast)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse Smithy AST file: %v\n", err)
	}
	if ast.Smithy == "" {
		return nil, fmt.Errorf("Cannot parse Smithy AST file: %v\n", err)
	}
	return ast, nil
}

func Assemble(paths []string) (*AST, error) {
	assembly := &AST{}
	for _, path := range paths {
		ext := filepath.Ext(path)
		switch ext {
		case ".smithy":
			ast, err := Parse(path)
			if err != nil {
				return nil, err
			}
			err = assembly.Merge(ast)
			if err != nil {
				return nil, err
			}
		case ".json":
			ast, err := LoadAST(path)
			if err != nil {
				return nil, err
			}
			err = assembly.Merge(ast)
			if err != nil {
				return nil, err
			}
		}
	}
	assembly.ExpandMixins()
	for _, k := range assembly.Shapes.Keys() {
		if tmp := assembly.GetShape(k); tmp != nil {
			if tmp.Type == "apply" {
				return nil, fmt.Errorf("Cannot apply traits to %s: target shape not found", k)
			}
		}
	}
	return assembly, nil
}

func (ast *AST) Apply(sftarget string, traits *NodeValue) error {
	target := sftarget
	lst := strings.Split(target, "$")
	field := ""
	if len(lst) == 2 {
		target = lst[0]
		field = lst[1]
	}
	if shape := ast.GetShape(target); shape != nil {
		if field != "" {
			m := shape.Members.Get(field)
			if m == nil {
				m = &Member{}
				shape.Members.Put(field, m)
			}
			if m.Traits == nil {
				m.Traits = NewNodeValue()
			}
			for _, k := range traits.Keys() {
				m.Traits.Put(k, cloneNodeValue(traits.Get(k)))
			}
		} else {
			t := ensureShapeTraits(shape)
			for _, k := range traits.Keys() {
				t.Put(k, cloneNodeValue(traits.Get(k)))
			}
		}
		return nil
	}
	return fmt.Errorf("Cannot apply traits to %s: target shape not found", target)
}

func (ast *AST) Merge(src *AST) error {
	if ast == nil || (ast.Metadata == nil && ast.Shapes == nil) {
		*ast = *src
		return nil
	}
	if ast.Smithy != src.Smithy {
		if strings.HasPrefix(ast.Smithy, "1") && strings.HasPrefix(src.Smithy, "2") {
			ast.Smithy = src.Smithy
		} else {
			fmt.Println("//WARNING: smithy version mismatch:", ast.Smithy, "and", src.Smithy)
		}
	}
	if src.Metadata != nil {
		if ast.Metadata == nil {
			ast.Metadata = src.Metadata
		} else {
			for _, k := range src.Metadata.Keys() {
				v := src.Metadata.Get(k)
				prev := ast.Metadata.Get(k)
				if prev != nil {
					err := ast.mergeConflict(k, prev, v)
					if err != nil {
						return err
					}
				}
				ast.Metadata.Put(k, v)
			}
		}
	}
	if src.Shapes != nil {
		for _, k := range src.Shapes.Keys() {
			srcShape := src.GetShape(k)
			curShape := ast.GetShape(k)
			err := ast.mergeShape(k, curShape, srcShape)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (ast *AST) mergeTraits(t1 *NodeValue, t2 *NodeValue) (*NodeValue, error) {
	for _, tk := range t2.Keys() {
		tv := t2.Get(tk)
		if t1.Get(tk) == nil {
			if t1 == nil {
			}
			t1.Put(tk, tv)
		} else {
			//if lists, append
			//if identical, ignore
			//else err
			return nil, fmt.Errorf("Duplicate trait in assembly: %s\n", tk)
		}
	}
	return t1, nil
}

func (ast *AST) mergeShape(key string, shape1 *Shape, shape2 *Shape) error {
	//note: only shape1 can be nil
	if shape1 == nil {
		ast.PutShape(key, shape2)
		return nil
	}
	mergedTraits, err := ast.mergeTraits(ensureShapeTraits(shape1), ensureShapeTraits(shape2))
	if err != nil {
		return err
	}
	if shape1.Type == "apply" {
		shape2.Traits = mergedTraits
		ast.PutShape(key, shape2)
	} else if shape2.Type == "apply" {
		shape1.Traits = mergedTraits
		ast.PutShape(key, shape1)
	} else { //neither is apply, just merge if we can
		return fmt.Errorf("Duplicate shape in assembly: %s\n", key) //not yet
	}
	return nil
}

func (ast *AST) mergeConflict(k string, v1 interface{}, v2 interface{}) error {
	//todo: if values are identical, accept one of them
	//todo: concat list values
	return fmt.Errorf("Conflict when merging models (not yet handled): %s\n", k)
}

var mixinSeq int

func (ast *AST) expandMixins(shapeId string) (*Shape, error) {
	shape := ast.Shapes.Get(shapeId)
	if shape == nil {
		return nil, fmt.Errorf("Shape not available: %s", shapeId)
	}
	if shape.Mixins != nil {
		last := len(shape.Mixins) - 1
		for i := last; i >= 0; i-- {
			mixinRef := shape.Mixins[i]
			mixinId := mixinRef.Target
			mixin, err := ast.expandMixins(mixinId) //this causes reverse order, not what we want
			if err != nil {
				return nil, err
			}
			//mixin := ast.Shapes.Get(mixinId)
			if mixin == nil {
				panic("oops, should have deferred elision and apply at assembly time")
			}
			if mixin.Members != nil {
				if shape.Type != "structure" {
					return nil, fmt.Errorf("Target for mixin with members not a Structure: %s", shapeId)
				}
				newMembers := NewMap[*Member]()
				for _, memKey := range mixin.Members.Keys() {
					mem := cloneMember(mixin.Members.Get(memKey))
					newMembers.Put(memKey, mem)
				}
				for _, memKey := range shape.Members.Keys() {
					mem := shape.Members.Get(memKey)
					if !newMembers.Has(memKey) {
						newMembers.Put(memKey, mem)
					} else {
						newMem := newMembers.Get(memKey)
						for _, trait := range mem.Traits.Keys() {
							newMem.Traits.Put(trait, mem.Traits.Get(trait))
						}
					}
				}
				shape.Members = newMembers
			}
			//note: `@private @mixin(localTraits: [private])`, which is a way to not propagate a trait on a mixin, is NYI
			if mixin.Traits != nil && mixin.Traits.Length() > 1 {
				newTraits := NewNodeValue()
				for _, trait := range mixin.Traits.Keys() {
					if trait != "smithy.api#mixin" && trait != "smithy.api#trait" {
						newTraits.Put(trait, cloneNodeValue(mixin.Traits.Get(trait)))
					}
				}
				if shape.Traits != nil {
					for _, trait := range shape.Traits.Keys() {
						newTraits.Put(trait, shape.Traits.Get(trait))
					}
				}
				shape.Traits = newTraits
			}
		}
		shape.Mixins = nil
	}
	return shape, nil
}

func (ast *AST) ExpandMixins() error {
	newShapes := NewMap[*Shape]()
	for _, shapeId := range ast.Shapes.Keys() {
		newShape, err := ast.expandMixins(shapeId)
		if err != nil {
			return err
		}
		if !newShape.Traits.Has("smithy.api#mixin") {
			newShapes.Put(shapeId, newShape)
		}
	}
	ast.Shapes = newShapes
	return nil
}

func (ast *AST) FilterDependencies(root []string, exclude []string) {
	included := NewMap[bool]()
	for _, k := range root {
		if !included.Has(k) {
			ast.noteDependencies(included, k)
		}
	}
	filtered := NewMap[*Shape]()
	for _, name := range included.Keys() {
		knn := stripNamespace(name)
		if !containsString(exclude, name) && !containsString(exclude, knn) && !strings.HasPrefix(name, "smithy.api#") {
			filtered.Put(name, ast.GetShape(name))
		}
	}
	ast.Shapes = filtered
}

func (ast *AST) ServiceDependencies() (string, error) {
	var root []string
	ns := ""
	for _, k := range ast.Shapes.Keys() {
		if ns == "" {
			ns = k[:strings.Index(k, "#")]
		}
		shape := ast.Shapes.Get(k)
		if shape == nil {
			fmt.Println("whoops, shape not defined:", k)
			panic("here")
		} else {
			if shape.Type == "service" {
				root = append(root, k)
			}
		}
	}
	switch len(root) {
	case 0:
		return ns, nil
	case 1:
		ast.FilterDependencies(root, nil)
		return ns, nil
	default:
		return "", fmt.Errorf("Cannot handle more than one service in model")
	}
}

func (ast *AST) Filter(tags []string) {
	var root []string
	if len(tags) == 0 {
		//if no tags, don't filter
		return
	}
	var include []string
	var exclude []string
	for _, tag := range tags {
		if strings.HasPrefix(tag, "-") {
			exclude = append(exclude, tag[1:])
		} else {
			include = append(include, tag)
		}
	}

	for _, k := range ast.Shapes.Keys() {
		kNoNamespace := stripNamespace(k)
		if len(include) == 0 || containsString(include, k) || containsString(include, kNoNamespace) {
			root = append(root, k)
		}
		shape := ast.Shapes.Get(k)
		if shape == nil {
			panic("whoops, shape is nil")
		}
		if shape.Traits != nil {
			shapeTags := shape.Traits.GetStringSlice("smithy.api#tags")
			if shapeTags != nil {
				for _, t := range shapeTags {
					if containsString(include, t) {
						root = append(root, k)
					}
				}
			}
		}
	}
	ast.FilterDependencies(root, exclude)
	/*
		included := make(map[string]bool, 0)
		for _, k := range root {
			if _, ok := included[k]; !ok {
				ast.noteDependencies(included, k)
			}
		}
		filtered := NewMap[*Shape]()
		for name, _ := range included {
			if !strings.HasPrefix(name, "smithy.api#") {
				filtered.Put(name, ast.GetShape(name))
			}
		}
	*/
}

func containsString(ary []string, val string) bool {
	for _, s := range ary {
		if s == val {
			return true
		}
	}
	return false
}
