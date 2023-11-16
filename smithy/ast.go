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
	//	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/boynton/data" //for Decimal
)

const UnspecifiedNamespace = "example"
const UnspecifiedVersion = "0.0"

type AST struct {
	Smithy   string            `json:"smithy"`
	Metadata *NodeValue       `json:"metadata,omitempty"`
	Shapes   *Map[*Shape] `json:"shapes,omitempty"`
}

type NodeValue struct {
	value interface{}
}
func NewNodeValue() *NodeValue {
//to do: use Map to preserve order of keys
	return &NodeValue{value:make(map[string]interface{}, 0)}
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
	}
	return false
}

func (node *NodeValue) Keys() []string {
	switch val := node.value.(type) {
	case map[string]interface{}:
		var keys []string
		for k, _ := range val {
			keys = append(keys, k)
		}
		return keys
	default:
		panic("Whoa, Keys()")
	}
}

func (node *NodeValue) Has(key string) bool {
	switch m := node.value.(type) {
	case map[string]interface{}:
		if _, ok := m[key]; ok {
			return true
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
	case *NodeValue:
		return m.Get(key)
	default:
		fmt.Println("Whoa:", m)
		panic("Whoa, Get()")
	}
}

func (node *NodeValue) AsString() string {
	if node == nil {
		return ""
	}
	switch s := node.value.(type) {
	case string:
		return s
	case *string:
		return *s
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
	if node.value != nil {
		switch n := node.value.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		case *data.Integer:
			return n.AsInt()
		case *data.Decimal:
			return n.AsInt()
		case *NodeValue:
			panic("double NodeValue wrapper, oops")
		}
		fmt.Println("asInt?", node)
		panic("Whoa GetInt!")
	}
	return 0
}

func (node *NodeValue) GetInt(key string, def int) int {
	n := node.Get(key)
	if n == nil {
		return def
	}
	return n.AsInt()
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
		fmt.Println("val:", node.value)
		panic("Whoa GetDecimal")
	}
	return nil
}

func (node *NodeValue) GetSlice(key string) []interface{} {
	if node == nil {
		return nil
	}
	switch m := node.value.(type) {
	case map[string]interface{}:
		if tmp, ok := m[key]; ok {
			if a, ok := tmp.([]interface{}); ok {
				return a
			}
		}
		return nil
	default:
		panic("Whoa, GetSlice()")
	}
}

func (node *NodeValue) GetStringSlice(key string) []string {
	fmt.Println("node:", node)
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
	default:
		fmt.Println(".Length on a NodeValue of:", node)
	}
	panic("whoops")
}

func (node *NodeValue) Put(key string, val interface{}) *NodeValue {
	switch m := node.value.(type) {
	case map[string]interface{}:
		m[key] = val
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

type Shape struct {
	Type   string       `json:"type"`

	//Service
	Version string `json:"version,omitempty"`

	//List and Set
	Member *Member `json:"member,omitempty"`

	//Map
	Key   *Member `json:"key,omitempty"`
	Value *Member `json:"value,omitempty"`

	//Structure and Union
	Members *Map[*Member]    `json:"members,omitempty"` //keys must be case-insensitively unique. For union, len(Members) > 0,
	Mixins  []*ShapeRef `json:"mixins,omitempty"`  //mixins for the shape

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
	Target string       `json:"target"`
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
	for k, _ := range m {
		nss = append(nss, k)
	}
	return nss
}

func (ast *AST) RequiresDocumentType() bool {
	included := make(map[string]bool, 0)
	for _, k := range ast.Shapes.Keys() {
		ast.noteDependencies(included, k)
	}
	if _, ok := included["smithy.api#Document"]; ok {
		return true
	}
	return false
}

func (ast *AST) noteDependenciesFromRef(included map[string]bool, ref *ShapeRef) {
	if ref != nil {
		ast.noteDependencies(included, ref.Target)
	}
}

func (ast *AST) noteDependencies(included map[string]bool, name string) {
	//note traits
	if name == "smithy.api#Document" {
		included[name] = true
		return
	}
	if name == "" || strings.HasPrefix(name, "smithy.api#") {
		return
	}
	if _, ok := included[name]; ok {
		return
	}
	included[name] = true
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
				ast.noteDependenciesFromRef(included, v)
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
			assembly.Merge(ast)
		case ".json":
			ast, err := LoadAST(path)
			if err != nil {
				return nil, err
			}
			assembly.Merge(ast)
		}
	}
	assembly.ExpandMixins()
	for _, k := range assembly.Shapes.Keys() {
		if tmp := assembly.GetShape(k); tmp != nil {
			if tmp.Type == "apply" {
				assembly.Apply(k, tmp.Traits)
				//assembly.Shapes.Delete(k)
			}
		}
	}
	return assembly, nil
}

func (ast *AST) Apply(target string, traits *NodeValue) error {
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
				fmt.Println("field not found:", field, target, data.Pretty(shape))
				panic("whoops")
			}
			t := ensureMemberTraits(m)
			for _, k := range traits.Keys() {
				t.Put(k, traits.Get(k))
			}
		} else {
			t := ensureShapeTraits(shape)
			for _, k := range traits.Keys() {
				t.Put(k, traits.Get(k))
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
			if tmp := ast.GetShape(k); tmp != nil {
				return fmt.Errorf("Duplicate shape in assembly: %s\n", k)
			}
			ast.PutShape(k, src.GetShape(k))
		}
	}
	return nil
}

func (ast *AST) mergeConflict(k string, v1 interface{}, v2 interface{}) error {
	//todo: if values are identical, accept one of them
	//todo: concat list values
	return fmt.Errorf("Conflict when merging metadata in models: %s\n", k)
}

func (ast *AST) expandMixins(shapeId string) error {
	//destructive: every mixin is merged once
	shape := ast.Shapes.Get(shapeId)
	if shape == nil {
		return fmt.Errorf("Shape not available:", shapeId)
	}
	//for _, mixinRef := range shape.Mixins {
	if shape.Mixins != nil {
		last := len(shape.Mixins) - 1
		for i := last; i >= 0; i--  {
			mixinRef := shape.Mixins[i]
			mixinId := mixinRef.Target
			ast.expandMixins(mixinId) //this causes reverse order, not what we want
			mixin := ast.Shapes.Get(mixinId) //expanded
			if mixin.Members != nil {
				if shape.Type != "structure" {
					return fmt.Errorf("Target for mixin with members not a Structure:", shapeId)
				}
				newMembers := NewMap[*Member]()
				for _, memKey := range mixin.Members.Keys() {
					mem := mixin.Members.Get(memKey)
					newMembers.Put(memKey, mem)
				}
				for _, memKey := range shape.Members.Keys() {
					if !newMembers.Has(memKey) {
						newMembers.Put(memKey, shape.Members.Get(memKey))
					}
				}
				shape.Members = newMembers
			}
			//note: `@private @mixin(localTraits: [private])`, which is a way to not propage a trait on a mixin, is NYI
			if mixin.Traits != nil && mixin.Traits.Length() > 1 {
				newTraits := NewNodeValue()
				for _, trait := range mixin.Traits.Keys() {
					if trait != "smithy.api#mixin" && trait != "smithy.api#trait" {
						newTraits.Put(trait, mixin.Traits.Get(trait))
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
	return nil
}

func (ast *AST) ExpandMixins() error {
	for _, shapeId := range ast.Shapes.Keys() {
		ast.expandMixins(shapeId)
	}
	return nil
}

func (ast *AST) FilterDependencies(root []string) {
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
		ast.FilterDependencies(root)
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
	ast.FilterDependencies(root)
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
	if len(exclude) > 0 {
		fmt.Print("TBD: exclude by tag: ", exclude)
		panic("here")
	}
}

func containsString(ary []string, val string) bool {
	for _, s := range ary {
		if s == val {
			return true
		}
	}
	return false
}

