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
package swagger

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/boynton/api/model"
	"github.com/boynton/data"
)

func Import(paths []string, tags []string, ns string) (*model.Schema, error) {
	if len(paths) != 1 {
		return nil, fmt.Errorf("Swagger import can aonly accept a single file")
	}
	path := paths[0]
	swagger, err := Load(path)
	if err != nil {
		return nil, err
	}
	file := filepath.Base(path)
	ext := filepath.Ext(path)
	name := file[:len(file)-len(ext)]
	return ImportSwagger(swagger, ns, name)
}

type Swagger struct {
	name      string
	namespace string
	raw       data.Object
	schema    *model.Schema
}

func (swagger *Swagger) String() string {
	return data.Pretty(swagger.raw)
}

func ImportSwagger(swagger *Swagger, ns string, name string) (*model.Schema, error) {
	if swagger.raw.GetString("swagger") != "2.0" {
		return nil, fmt.Errorf("Not a valid swagger document. Only version 2.0 is supported")
	}
	swagger.schema = model.NewSchema()
	err := swagger.ImportInfo(ns, name)
	if err != nil {
		return nil, err
	}
	err = swagger.ImportService()
	return swagger.schema, err
}

func (swagger *Swagger) ServiceName(s string) string {
	if !model.IsSymbol(s) {
		//from filename?
		s = "Untitled"
	}
	return s
}

func (swagger *Swagger) ImportInfo(ns, name string) error {
	//the name is fromthe filename. I find that most uses of swagger embed the version in this name
	swagger.schema.Namespace = model.Namespace(ns)
	if info := swagger.raw.GetObject("info"); info != nil {
		//name := info.GetString("title")
		//name := swagger.ServiceName(name)
		schema := swagger.schema
		schema.Id = model.AbsoluteIdentifier(string(schema.Namespace) + "#" + name)
		//schema.Version = info.GetString("version") //typically redundant
		schema.Comment = info.GetString("description")
		schema.Base = swagger.raw.GetString("basePath")
		license := info.GetObject("license")
		if license != nil {
			/*
				schema.Metadata = data.NewObject()
				schema.Metadata.Put("x_license_name", license.GetString("name"))
				schema.Metadata.Put("x_license_url", license.GetString("url"))
			*/
		}
		return nil
	}
	return nil
}

func (swagger *Swagger) ImportService() error {
	paths := swagger.raw.GetObject("paths")
	for _, path := range paths.Keys() {
		def := paths.GetObject(path)
		for _, method := range def.Keys() {
			switch method {
			case "post", "put", "get", "delete":
				swagger.ImportOperation(method, path, def.GetObject(method))
			}
		}
	}
	defs := swagger.raw.GetObject("definitions")
	for _, k := range defs.Keys() {
		def := defs.GetObject(k)
		otype := def.GetString("type")
		switch otype {
		case "integer", "number":
			return swagger.ImportNumber(k, def)
		case "string":
			if def.Has("enum") {
				err := swagger.ImportEnum(k, def)
				if err != nil {
					return err
				}
			} else {
				err := swagger.ImportString(k, def)
				if err != nil {
					return err
				}
			}
		case "array":
			err := swagger.ImportList(k, def)
			if err != nil {
				return err
			}
		case "object":
			err := swagger.ImportStruct(k, def)	
			if err != nil {
				return err
			}
		default:
			fmt.Println("whoops:", k, "->", model.Pretty(def))
			panic("here")
		}
	}
	return nil
}

func (swagger *Swagger) toCanonicalAbsoluteId(name string) model.AbsoluteIdentifier {
	return model.AbsoluteIdentifier(string(swagger.schema.Namespace) + "#" + name)
}

func (swagger *Swagger) toCanonicalTypeName(prop *data.Object) model.AbsoluteIdentifier {
	ref := prop.GetString("$ref")
	if ref != "" {
		if ref == "#/definitions/Timestamp" {
			return model.AbsoluteIdentifier("base#Timestamp")
		} else if ref == "#/definitions/Decimal" {
			return model.AbsoluteIdentifier("base#Decimal")
		}
		s := ref[len("#/definitions/"):]
		return swagger.toCanonicalAbsoluteId(s)
	}
	tname := prop.GetString("type")
	name := "?"
	switch tname {
	case "number", "integer":
		name = swagger.numberBase(prop).String()
	case "string":
		switch prop.GetString("format") {
		case "date-time":
			name = "base#Timestamp"
		default:
			name = "base#String"
		}
	case "boolean":
		name = "base#Bool"
	case "array":
		name = "base#Array"
	default:
		sch := prop.GetObject("schema")
		if sch != nil {
			return swagger.toCanonicalTypeName(sch)
		}
	}
	return model.AbsoluteIdentifier(name)
}

func (swagger *Swagger) ImportOperationOutput(sStatus string, def *data.Object) (*model.OperationOutput, error) {
	status, err := strconv.Atoi(sStatus)
	if err != nil {
		return nil, err
	}
	out := &model.OperationOutput{
		HttpStatus: int32(status),
		Comment: def.GetString("description"),
	}
	sch := def.GetObject("schema")
	if sch != nil {
		payloadType := swagger.toCanonicalTypeName(sch)
		fd := &model.OperationOutputField{
			Name: "body",
			Type: payloadType,
			HttpPayload: true,
		}
		out.Fields = append(out.Fields, fd)
	}
	if false {
		//need to verify how to name these, since Swagger doesn't.
		headers := def.GetObject("headers")
		for i, hname := range headers.Keys() {
			hdef := headers.GetObject(hname)
			htype := swagger.toCanonicalTypeName(hdef)
			fd := &model.OperationOutputField{
				Name: model.Identifier(fmt.Sprintf("header_%d", i)),
				Comment: hdef.GetString("description"),
				Type: htype,
				HttpHeader: hname,
			}
			out.Fields = append(out.Fields, fd)
		}
	}
	return out, nil
}

func (swagger *Swagger) ImportOperationInputField(def *data.Object) (*model.OperationInputField, error) {
	f := &model.OperationInputField{
		Name: model.Identifier(def.GetString("name")),
		Comment: def.GetString("description"),
	}
	switch def.GetString("in") {
	case "path":
		f.HttpPath = true
	case "query":
		f.HttpQuery = f.Name
	case "body":
		f.HttpPayload = true
	default:
		fmt.Println("FIX:", model.Pretty(def))
		panic("FIX ME")
	}
	if def.GetBool("required") {
		f.Required = true
	}
	f.Type = swagger.toCanonicalTypeName(def)
	if f.Type == "base#String" {
		minsize := def.GetInt64("minLength")
		if minsize != 0 {
			f.MinSize = minsize
		}
		maxsize := def.GetInt64("maxLength")
		if maxsize != 0 {
			f.MaxSize = maxsize
		}
	}
	return f, nil
}

func (swagger *Swagger) ImportOperationInput(def *data.Object) (*model.OperationInput, error) {
	input := &model.OperationInput{
	}
	for _, param := range def.GetArray("parameters") {
		p := data.AsObject(param)
		f, err := swagger.ImportOperationInputField(p)
		if err != nil {
			return nil, err
		}
		input.Fields = append(input.Fields, f)
	}
	return input, nil
}

func (swagger *Swagger) ImportOperation(method string, path string, def *data.Object) error {
	//current assumption: The first 2xx response encountered becomes the "expected" output, others are "exceptions"
	name := def.GetString("operationId")

	input, err := swagger.ImportOperationInput(def)
	if err != nil {
		return err
	}

	responses := def.GetObject("responses")
	var output *model.OperationOutput
	var exceptions []*model.OperationOutput

	for _, sStatus := range responses.Keys() {
		outdef, err := swagger.ImportOperationOutput(sStatus, responses.GetObject(sStatus))
		if err != nil {
			return err
		}
		if output == nil && strings.HasPrefix(sStatus, "2") {
			output = outdef
		} else {
			exceptions = append(exceptions, outdef)
		}
	}
	if name == "" {
		return fmt.Errorf("Cannot determine operation id: %s", model.Pretty(def))
	}
	op := &model.OperationDef{
		Id:   swagger.toCanonicalAbsoluteId(name),
		Comment: def.GetString("description"),
		HttpMethod: strings.ToUpper(method),
		HttpUri: path,
		Input: input,
		Output: output,
		Exceptions: exceptions,
	}
	return swagger.schema.AddOperationDef(op)
}

func (swagger *Swagger) ImportString(name string, def *data.Object) error {
	td := &model.TypeDef{
		Id:   swagger.toCanonicalAbsoluteId(name),
		Base: model.String,
		Comment: def.GetString("description"),
	}
	return swagger.schema.AddTypeDef(td)
}

func (swagger *Swagger) numberBase(def *data.Object) model.BaseType {
	switch def.GetString("type") {
	case "integer":
		switch def.GetString("format") {
		case "int8":
			return model.Int8
		case "int16":
			return model.Int16
		case "int32":
			return model.Int32
		case "int64":
			return model.Int64
		default:
			return model.Integer
		}
	default:
		switch def.GetString("format") {
		case "float":
			return model.Float32
		case "double":
			return model.Float64
		default:
			return model.Decimal
		}
	}
		
	}
func (swagger *Swagger) ImportNumber(name string, def *data.Object) error {
	td := &model.TypeDef{
		Id:   swagger.toCanonicalAbsoluteId(name),
		Base: swagger.numberBase(def),
		Comment: def.GetString("description"),
	}
	return swagger.schema.AddTypeDef(td)
}

func (swagger *Swagger) ImportStruct(name string, def *data.Object) error {
	td := &model.TypeDef{
		Id:   swagger.toCanonicalAbsoluteId(name),
		Base: model.Struct,
		Comment: def.GetString("description"),
	}
	props := def.GetObject("properties")
	req := def.GetStringSlice("required")
	for _, name := range props.Keys() {
		v := props.GetObject(name)
		fd := &model.FieldDef{
			Name: model.Identifier(name),
			Comment: v.GetString("description"),
		}
		fd.Type = swagger.toCanonicalTypeName(v)
		for _, rname := range req {
			if rname == name {
				fd.Required = true
				break
			}
		}
		td.Fields = append(td.Fields, fd)
	}
	return swagger.schema.AddTypeDef(td)
}

func (swagger *Swagger) ImportList(name string, def *data.Object) error {
	td := &model.TypeDef{
		Id:   swagger.toCanonicalAbsoluteId(name),
		Base: model.List,
		Comment: def.GetString("description"),
	}
    items := def.GetObject("items")
	td.Items = swagger.toCanonicalTypeName(items)
	return swagger.schema.AddTypeDef(td)
}

func (swagger *Swagger) ImportEnum(name string, def *data.Object) error {
	td := &model.TypeDef{
		Id:   swagger.toCanonicalAbsoluteId(name),
		Base: model.Enum,
		Comment: def.GetString("description"),
	}
    syms := def.GetStringSlice("enum")
	for _, val := range syms {
		sym := model.Identifier(val)
		//validate it *is* a sym!
		el := &model.EnumElement{
			Symbol: sym,
		}
		td.Elements = append(td.Elements, el)
	}
	return swagger.schema.AddTypeDef(td)
}

func (swagger *Swagger) fullName(name string) string {
	return swagger.namespace + "#" + name
}

func Load(path string) (*Swagger, error) {
	swagger := &Swagger{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot read swagger file: %v\n", err)
	}
	err = json.Unmarshal(data, &swagger.raw)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse swagger file: %v\n", err)
	}
	return swagger, nil
}

func TrimSpace(s string) string {
	return TrimLeftSpace(TrimRightSpace(s))
}

func TrimRightSpace(s string) string {
	return strings.TrimRight(s, " \t\n\v\f\r")
}

func TrimLeftSpace(s string) string {
	return strings.TrimLeft(s, " \t\n\v\f\r")
}

func Capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}
