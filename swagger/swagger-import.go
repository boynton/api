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
		var b []rune
		for _, r := range s {
			if r == '-' {
				r = '_'
			}
			b = append(b, r)
		}
		s = string(b)
	}
	return s
}

func (swagger *Swagger) ImportInfo(ns, name string) error {
	//the name is fromthe filename. I find that most uses of swagger embed the version in this name
	swagger.schema.Namespace = model.Namespace(ns)
	if info := swagger.raw.GetObject("info"); info != nil {
		//name := info.GetString("title")
		name := swagger.ServiceName(name)
		schema := swagger.schema
		schema.Id = model.AbsoluteIdentifier(string(schema.Namespace) + "#" + name)
		schema.Version = info.GetString("version")
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

func (swagger *Swagger) resolveRef(ref string) *data.Object {
	d := swagger.raw.Get("definitions")
	switch m := d.(type) {
	case map[string]interface{}:
		return data.ObjectFromMap(m)
	}
	return nil
}

func (swagger *Swagger) ImportService() error {
	defs := swagger.raw.GetObject("definitions")
	for _, b := range defs.Bindings() {
		k := b.Key
		def := data.AsObject(b.Value)
		otype := def.GetString("type")
		switch strings.ToLower(otype) {
		case "integer", "number":
			err := swagger.ImportNumber(k, def)
			if err != nil {
				return err
			}
		case "boolean":
			err := swagger.ImportBoolean(k, def)
			if err != nil {
				return err
			}
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
			//no type specified
			allof := def.Get("allOf")
			if allof != nil {
				err := swagger.ImportAllOf(k, def)
				if err != nil {
					return err
				}
			} else {
				fmt.Println("whoops:", k, "->", model.Pretty(def))
				return fmt.Errorf("Cannot import this type: %s", otype)
			}
		}
	}
	paths := swagger.raw.GetObject("paths")
	for _, b := range paths.Bindings() {
		path := b.Key
		def := data.AsObject(b.Value)
		for _, bb := range def.Bindings() {
			method := bb.Key
			switch method {
			case "post", "put", "get", "delete":
				swagger.ImportOperation(method, path, def.GetObject(method))
			}
		}
	}
	return nil
}

func (swagger *Swagger) toCanonicalAbsoluteId(name string) model.AbsoluteIdentifier {
	return model.AbsoluteIdentifier(string(swagger.schema.Namespace) + "#" + name)
}

func (swagger *Swagger) toCanonicalTypeName(prop *data.Object) model.AbsoluteIdentifier {
	return swagger.toCanonicalTypeNameWithContext(prop, "")
}

func (swagger *Swagger) toCanonicalTypeNameWithContext(prop *data.Object, context string) model.AbsoluteIdentifier {
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
		itemsType := swagger.toCanonicalTypeName(prop.GetObject("items"))
		genTypeId := itemsType + "Array"
		if context != "" {
			genTypeId = model.AbsoluteIdentifier(string(genTypeId) + "_" + context)
		}
		//bug: may not be unique, depending on context. If a field in a structure, also prefix that structure's name
		genTypeName := model.StripNamespace(genTypeId)
		if swagger.schema.GetTypeDef(genTypeId) == nil {
			swagger.ImportList(genTypeName, prop)
		}
		name = genTypeName
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
		for i, b := range headers.Bindings() {
			hname := b.Key
			hdef := data.AsObject(b.Value)
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

func identifierFromHeader(s string) string {
	parts := strings.Split(s, "-")
	r := parts[0]
	for i := 1; i<len(parts); i++ {
		r = r + Capitalize(parts[i])
	}
	return r
}

func (swagger *Swagger) ImportOperationInputField(def *data.Object) (*model.OperationInputField, error) {
	rawName := def.GetString("name")
	name := identifierFromHeader(rawName)
	f := &model.OperationInputField{
		Name: model.Identifier(name),
		Comment: def.GetString("description"),
	}
	switch def.GetString("in") {
	case "path":
		f.HttpPath = true
	case "query":
		f.HttpQuery = f.Name
	case "body":
		f.HttpPayload = true
	case "header":
		f.HttpHeader = rawName
	default:
		fmt.Println("FIX:", model.Pretty(def))
		panic("FIX ME")
	}
	if def.GetBool("required") {
		f.Required = true
	}
	f.Type = swagger.toCanonicalTypeNameWithContext(def, name)
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
	for _, param := range def.GetSlice("parameters") {
		p := data.AsObject(param)
		f, err := swagger.ImportOperationInputField(p)
		if err != nil {
			return nil, err
		}
		input.Fields = append(input.Fields, f)
	}
	return input, nil
}

func HttpStatusName(sStatus string) string {
	status, err := strconv.Atoi(sStatus)
	if err == nil {
		switch status {
		case 200:
			return "OK"
		case 201:
			return "Created"
		case 202:
			return "Accepted"
		case 203:
			return "NonAuthoritativeInformation"
		case 204:
			return "NoContent"
		case 205:
			return "ResetContent"
		case 206:
			return "PartialContent"
		case 300:
			return "MultipleChoices"
		case 301:
			return "MovedPermanently"
		case 302:
			return "Found"
		case 303:
			return "SeeOther"
		case 304:
			return "NotModified"
		case 307:
			return "TemporaryRedirect"
		case 308:
			return "PermanentRedirect"
		case 400:
			return "BadRequest"
		case 401:
			return "Unauthorized"
		case 403:
			return "Forbidden"
		case 404:
			return "NotFound"
		case 405:
			return "MethodNotAllowed"
		case 406:
			return "NotAcceptable"
		case 407:
			return "ProxyAuthenticationRequired"
		case 408:
			return "RequestTimeout"
		case 409:
			return "Conflict"
		case 410:
			return "Gone"
		case 411:
			return "LengthRequired"
		case 412:
			return "PreconditionFailed"
		case 413:
			return "PayloadTooLarge"
		case 414:
			return "URITooLong"
		case 415:
			return "UnsupportedMediaType"
		case 416:
			return "RangeNotSatisfiable"
		case 417:
			return "ExpectationFailed"
		case 418:
			return "Teapot"
		case 421:
			return "MisdirectedRequest"
		case 428:
			return "PreconditionRequired"
		case 429:
			return "TooManyRequests"
		case 431:
			return "RequestHeaderFieldsTooLarge"
		case 500:
			return "InternalServerError"
		case 501:
			return "NotImplemented"
		case 502:
			return "BadGateway"
		case 503:
			return "ServiceUnavailable"
		case 504:
			return "GatewayTimeout"
		case 505:
			return "HTTPVersionNotSupported"
		}
	}
	return sStatus + "Status"
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
	var exceptionRefs []model.AbsoluteIdentifier

	for _, b := range responses.Bindings() {
		sStatus := b.Key
		resp := data.AsObject(b.Value)
		outdef, err := swagger.ImportOperationOutput(sStatus, resp)
		if err != nil {
			return err
		}
		if output == nil && strings.HasPrefix(sStatus, "2") {
			output = outdef
		} else {
			if outdef.Id == "" {
				ename := HttpStatusName(sStatus) + "Exception"
				outdef.Id = swagger.toCanonicalAbsoluteId(ename)
			}
			exceptions = append(exceptions, outdef)
			exceptionRefs = append(exceptionRefs, outdef.Id)
		}
	}
	if name == "" {
		return fmt.Errorf("Cannot determine operation id: %s", model.Pretty(def))
	}
	for _, e := range exceptions {
		err := swagger.schema.EnsureExceptionDef(e)
		if err != nil {
			return err
		}
	}
	op := &model.OperationDef{
		Id:   swagger.toCanonicalAbsoluteId(name),
		Comment: def.GetString("description"),
		HttpMethod: strings.ToUpper(method),
		HttpUri: path,
		Input: input,
		Output: output,
		Exceptions: exceptionRefs,
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

func (swagger *Swagger) ImportBoolean(name string, def *data.Object) error {
	td := &model.TypeDef{
		Id:   swagger.toCanonicalAbsoluteId(name),
		Base: model.Bool,
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
	for _, b := range props.Bindings() {
		fname := b.Key
		v := data.AsObject(b.Value)
		fd := &model.FieldDef{
			Name: model.Identifier(fname),
			Comment: v.GetString("description"),
		}
		//fd.Type = swagger.toCanonicalTypeNameWithContext(v, name)
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
	}
	td.Comment = def.GetString("description")
    items := def.GetObject("items")
	if def.Has("minItems") {
		td.MinSize = def.GetInt64("minItems")
	}
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

func (swagger *Swagger) ImportAllOf(name string, adef *data.Object) error {
	td := &model.TypeDef{
		Id:   swagger.toCanonicalAbsoluteId(name),
		Base: model.Struct,
		Comment: adef.GetString("description"),
	}
	defs := adef.Get("allOf")
	for _, d := range data.AsSlice(defs) {
		def := data.AsObject(d)
		if def.Has("$ref") {
			s := def.GetString("$ref")
			def = swagger.resolveRef(s)
		}
		props := def.GetObject("properties")
		req := def.GetStringSlice("required")
		for _, b := range props.Bindings() {
			name := b.Key
			v := data.AsObject(b.Value)
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
