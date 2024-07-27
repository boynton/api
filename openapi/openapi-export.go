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
package openapi

import (
	"fmt"
	"strings"

	"github.com/boynton/api/model"
	"github.com/boynton/data"
)

type Generator struct {
	model.BaseGenerator
	name    string
	openapi *OpenAPI
}

func (gen *Generator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.openapi = &OpenAPI{
		OpenAPI: "3.0.0",
	}
	gen.GenerateService()
	gen.GenerateOperations()
	gen.GenerateTypes()
	fname := gen.FileName(gen.name, ".json")
	s := model.Pretty(gen.openapi)
	err = gen.Write(s, fname, "")
	return err
}

func (gen *Generator) GenerateService() error {
	version := gen.Schema.Version
	if version == "" {
		version = "1"
	}
	gen.openapi.Info = &Info{
		Title:       model.StripNamespace(gen.Schema.Id),
		Version:     version,
		Description: gen.Schema.Comment,
	}
	return nil
}

func (gen *Generator) GenerateOperations() error {
	for _, op := range gen.Schema.Operations {
		gen.GenerateOperation(op)
	}
	return nil
}

func (gen *Generator) GenerateTypes() error {
	for _, td := range gen.Schema.Types {
		gen.GenerateType(td)
	}
	return nil
}

func (gen *Generator) SchemaFromTypeRef(tref model.AbsoluteIdentifier) *Schema {
	ref := string(tref)
	if strings.HasPrefix(ref, "base#") {
		var otype, oformat string
		switch ref {
		case "base#Int8", "base#Int16", "base#Int32":
			otype = "integer"
			oformat = "int32"
		case "base#Int64":
			otype = "integer"
			oformat = "int64"
		case "base#Float64":
			otype = "number"
			oformat = "double"
		case "base#Float32":
			otype = "number"
			oformat = "float"
		case "base#Decimal", "base#Integer":
			otype = "number"
		case "base#Timestamp":
			otype = "string"
			oformat = "date-time"
		case "base#String":
			otype = "string"
		default:
			fmt.Println("ref to", tref)
			panic("here")
		}
		return &Schema{
			Type:   otype,
			Format: oformat,
		}
	} else {
		schema := &Schema{
			Ref: "#/components/schemas/" + model.StripNamespace(tref),
		}
		return schema
	}
}

func (gen *Generator) GenerateOperation(op *model.OperationDef) error {
	path := op.HttpUri
	if gen.openapi.Paths == nil {
		gen.openapi.Paths = make(map[string]*PathItem, 0)
	}
	var pi *PathItem
	if p, ok := gen.openapi.Paths[path]; ok {
		pi = p
	} else {
		pi = &PathItem{}
		gen.openapi.Paths[path] = pi
	}
	operation := &Operation{
		OperationId: model.StripNamespace(op.Id),
		Description: op.Comment,
	}
	var inPayload *model.OperationInputField
	if op.Input != nil {
		var params []*Parameter
		for _, in := range op.Input.Fields {
			param := &Parameter{
				Name:     string(in.Name),
				Required: in.Required,
				Schema:   gen.SchemaFromTypeRef(in.Type),
			}
			if in.HttpPath {
				param.In = "path"
			} else if in.HttpQuery != "" {
				param.In = "query"
			} else if in.HttpHeader != "" {
				param.In = "header"
			} else if in.HttpPayload {
				inPayload = in
				continue
			}
			params = append(params, param)
		}
		operation.Parameters = params
	}
	if inPayload != nil {
		content := make(map[string]*MediaType, 0)
		sch := gen.SchemaFromTypeRef(inPayload.Type)
		content["application/json"] = &MediaType{
			Schema: sch,
		}
		operation.RequestBody = &RequestBody{
			Required: true,
			Content:  content,
		}
	}

	if op.Output != nil {
		gen.GenerateOperationOutput(operation, op, op.Output, false)
	}
	for _, eid := range op.Exceptions {
		exc := gen.Schema.GetExceptionDef(eid)
		gen.GenerateOperationOutput(operation, op, exc, true)
	}
	switch op.HttpMethod {
	case "POST":
		pi.Post = operation
	case "GET":
		pi.Get = operation
	case "PUT":
		pi.Put = operation
	case "DELETE":
		pi.Delete = operation
	default:
		panic("handle this method: " + op.HttpMethod)
	}
	return nil
}

// To Do: fix JSON marshaling of Extension so I can provide a name per input/output/exception. Currently, name is lost.
func outputName(op *model.OperationDef, output *model.OperationOutput, isException bool) string {
	if isException {
		defaultId := model.AbsoluteIdentifier(fmt.Sprintf("%sException%d", op.Id, output.HttpStatus))
		if output.Id != "" && output.Id != defaultId {
			return model.StripNamespace(op.Output.Id)
		}
	} else {
		if output.Id != "" && output.Id != (op.Id+"Output") {
			return model.StripNamespace(op.Output.Id)
		}
	}
	return ""
}

func (gen *Generator) GenerateOperationOutput(operation *Operation, op *model.OperationDef, output *model.OperationOutput, isException bool) error {
	var outPayload *model.OperationOutputField
	responses := operation.Responses
	if responses == nil {
		responses = make(map[string]*Response, 0)
		operation.Responses = responses
	}
	r := &Response{
		Description: output.Comment,
	}
	sStatus := fmt.Sprintf("%d", output.HttpStatus)
	responses[sStatus] = r
	for _, out := range output.Fields {
		if out.HttpHeader != "" {
			if r.Headers == nil {
				r.Headers = make(map[string]*Header, 0)
			}
			r.Headers[out.HttpHeader] = &Header{
				Schema:      gen.SchemaFromTypeRef(out.Type),
				Description: out.Comment,
			}
		} else if out.HttpPayload {
			outPayload = out
			continue
		}
	}
	if outPayload != nil {
		content := make(map[string]*MediaType, 0)
		sch := gen.SchemaFromTypeRef(outPayload.Type)
		content["application/json"] = &MediaType{
			Schema: sch,
		}
		r.Content = content
	}
	return nil
}

func (gen *Generator) GenerateResource(rez *model.ResourceDef) error {
	return nil
}

func (gen *Generator) GenerateException(op *model.OperationOutput) error {
	return nil
}

func (gen *Generator) GenerateType(td *model.TypeDef) error {
	sch := &Schema{
		Description: td.Comment,
	}
	switch td.Base {
	case model.BaseType_String:
		sch.Type = "string"
		if td.Pattern != "" {
			sch.Pattern = td.Pattern
		}
	case model.BaseType_Struct, model.BaseType_Union:
		sch.Type = "object"
		var required []string
		props := make(map[string]*Schema, 0)
		for _, fd := range td.Fields {
			props[string(fd.Name)] = gen.SchemaFromTypeRef(fd.Type)
			if fd.Required {
				required = append(required, string(fd.Name))
			}
		}
		sch.Properties = props
		sch.Required = required
	case model.BaseType_List:
		sch.Type = "array"
		sch.Items = gen.SchemaFromTypeRef(td.Items)
	case model.BaseType_Map:
		sch.Type = "array"
		sch.Items = gen.SchemaFromTypeRef(td.Items)
	case model.BaseType_Int8:
		sch.Type = "integer"
		sch.Format = "int8"
	case model.BaseType_Int16:
		sch.Type = "integer"
		sch.Format = "int16"
	case model.BaseType_Int32:
		sch.Type = "integer"
		sch.Format = "int32"
	case model.BaseType_Int64:
		sch.Type = "integer"
		sch.Format = "int64"
	case model.BaseType_Integer:
		sch.Type = "integer"
	case model.BaseType_Float32:
		sch.Type = "number"
		sch.Format = "float"
	case model.BaseType_Float64:
		sch.Type = "number"
		sch.Format = "double"
	case model.BaseType_Decimal:
		sch.Type = "number"
	case model.BaseType_Enum:
		sch.Type = "string"
		for _, el := range td.Elements {
			sch.Enum = append(sch.Enum, el.Value)
		}
		/*
			case model.Union:
					for _, fd := range td.Fields {
						sch.OneOf = append(sch.OneOf, gen.SchemaFromTypeRef(fd.Type))
					}
		*/
	default:
		fmt.Println("implement me: openapi.GenerateType: ", model.Pretty(td))
		panic("here")
	}
	if gen.openapi.Components == nil {
		gen.openapi.Components = &Components{
			Schemas: make(map[string]*Schema, 0),
		}
	}
	name := model.StripNamespace(td.Id)
	gen.openapi.Components.Schemas[name] = sch
	return nil
}
