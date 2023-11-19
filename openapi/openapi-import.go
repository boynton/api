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
	"net/url"
	"strconv"
	"strings"

	"github.com/boynton/api/common"
	"github.com/boynton/api/model"
)

func Import(paths []string, tags []string, ns string) (*model.Schema, error) {
	if len(paths) != 1 {
		return nil, fmt.Errorf("Openapi import can aonly accept a single file")
	}
	path := paths[0]
	openapi, err := Load(path)
	if err != nil {
		return nil, err
	}
	mb := &ModelBuilder{
		openapi: openapi,
		schema:  model.NewSchema(),
		ns:      ns,
	}
	return mb.Build()
}

type ModelBuilder struct {
	openapi *OpenAPI
	schema  *model.Schema
	ns      string
}

func (mb *ModelBuilder) Build() (*model.Schema, error) {
	//fmt.Println("openapi:", model.Pretty(mb.openapi))
	if mb.openapi.OpenAPI != "3.0.0" {
		return nil, fmt.Errorf("Not a supported openapi document. Only version 3.0.0 is supported")
	}
	err := mb.ImportInfo()
	if err != nil {
		return nil, err
	}
	err = mb.ImportService()
	if err != nil {
		return nil, err
	}
	err = mb.schema.Validate()
	if err != nil {
		return nil, err
	}
	return mb.schema, nil
}

func (mb *ModelBuilder) basePath() (string, error) {
	if len(mb.openapi.Servers) > 0 {
		if len(mb.openapi.Servers) != 1 {
			return "", fmt.Errorf("Only a single server supported for openapi")
		}
		server := mb.openapi.Servers[0]
		if strings.Index(server.URL, "{") >= 0 {
			return "", fmt.Errorf("Templated server URI not yet supported")
		}
		u, err := url.Parse(server.URL)
		if err != nil {
			return "", fmt.Errorf("Malformed server URI: %q", server.URL)
		}
		return u.Path, nil
	}
	return "", nil
}

func (mb *ModelBuilder) ImportInfo() error {
	name := mb.openapi.Info.Title
	if name == "" {
		name = "Untitled"
	}
	mb.schema.Id = mb.toCanonicalAbsoluteId(name)
	mb.schema.Version = mb.openapi.Info.Version
	mb.schema.Comment = mb.openapi.Info.Description
	b, err := mb.basePath()
	if err != nil {
		return err
	}
	mb.schema.Base = b
	//metadata?
	return nil
}

func (mb *ModelBuilder) ImportService() error {
	paths := mb.openapi.Paths
	for path, pi := range paths {
		if pi.Post != nil {
			err := mb.ImportOperation(path, "POST", pi.Post)
			if err != nil {
				return err
			}
		}
		if pi.Get != nil {
			err := mb.ImportOperation(path, "GET", pi.Get)
			if err != nil {
				return err
			}
		}
		if pi.Put != nil {
			err := mb.ImportOperation(path, "PUT", pi.Put)
			if err != nil {
				return err
			}
		}
		if pi.Patch != nil {
			err := mb.ImportOperation(path, "PATCH", pi.Patch)
			if err != nil {
				return err
			}
		}
		if pi.Delete != nil {
			err := mb.ImportOperation(path, "DELETE", pi.Delete)
			if err != nil {
				return err
			}
		}
	}
	comp := mb.openapi.Components
	for k, s := range comp.Schemas {
		err := mb.ImportSchema(k, s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (mb *ModelBuilder) opComment(pop *Operation) string {
	s := pop.Summary
	if s == "" {
		s = pop.Description
	}
	return strings.Trim(s, "\n")
}

func (mb *ModelBuilder) ImportOperation(path string, method string, pop *Operation) error {
	opId := mb.toCanonicalAbsoluteId(common.Capitalize(pop.OperationId))
	op := &model.OperationDef{
		Id:         opId,
		HttpMethod: method,
		HttpUri:    path,
		Comment:    mb.opComment(pop),
		Input: &model.OperationInput{
			Id: opId + "Input",
		},
	}
	//input
	for _, param := range pop.Parameters {
		ftype := mb.toCanonicalTypeName(param.Schema)
		fname := param.Name
		fd := &model.OperationInputField{
			Name:     model.Identifier(fname),
			Type:     ftype,
			Required: param.Required,
		}
		switch param.In {
		case "path":
			fd.HttpPath = true
		case "query":
			fd.HttpQuery = model.Identifier(param.Name)
		case "header":
			fd.HttpHeader = param.Name
		}
		op.Input.Fields = append(op.Input.Fields, fd)
	}
	if pop.RequestBody != nil {
		if content, ok := pop.RequestBody.Content["application/json"]; ok {
			ftype := mb.toCanonicalTypeName(content.Schema)
			fname := strings.ToLower(mb.toSimpleTypeName(ftype)) // lossy!
			fd := &model.OperationInputField{
				Name:        model.Identifier(fname),
				Type:        ftype,
				Required:    true,
				HttpPayload: true,
			}
			op.Input.Fields = append(op.Input.Fields, fd)
		}
	}

	//outputs
	expectedStatus := ""
	for status, _ := range pop.Responses {
		if strings.HasPrefix(status, "2") || strings.HasPrefix(status, "3") {
			expectedStatus = status
			break
		}
	}
	expected := 204
	if expectedStatus != "" {
		code, err := strconv.Atoi(expectedStatus)
		if err != nil {
			return err
		}
		expected = code
	}
	for status, eparam := range pop.Responses {
		//eparam := pop.Responses[status]
		if eparam == nil {
			return fmt.Errorf("no response entity type provided for operation %q", pop.OperationId)
		}
		var err error
		code := 200
		if status != "default" && strings.Index(status, "X") < 0 {
			code, err = strconv.Atoi(status)
			if err != nil {
				return err
			}
		}
		output := &model.OperationOutput{
			HttpStatus: int32(code),
			Comment:    eparam.Description,
		}
		for contentType, mediadef := range eparam.Content {
			if contentType == "application/json" { //for now
				if code == 204 || code == 304 {
					//not content in either of these
					continue
				}
				fd := &model.OperationOutputField{
					HttpPayload: true,
				}
				fd.Type = mb.toCanonicalTypeName(mediadef.Schema)
				fd.Name = mb.toIdentifier(fd.Type)
				output.Fields = append(output.Fields, fd)
			}
		}
		for header, def := range eparam.Headers {
			fd := &model.OperationOutputField{
				HttpHeader: header,
				Comment:    def.Description,
			}
			s := header
			//most app-defined headers start with "x-" or "X-". Strip that off for a more reasonable variable name.
			if strings.HasPrefix(s, "x-") || strings.HasPrefix(s, "X-") {
				s = s[2:]
			}
			fd.Name = model.Identifier(s)
			schref := def.Schema
			if schref != nil {
				fd.Type = mb.toCanonicalTypeName(schref)
				output.Fields = append(output.Fields, fd)
			}
		}
		if expected == code {
			output.Id = opId + "Output"
			op.Output = output
		} else {
			output.Id = model.AbsoluteIdentifier(fmt.Sprintf("%sException%d", opId, output.HttpStatus))
			op.Exceptions = append(op.Exceptions, output)
		}
	}
	mb.schema.Operations = append(mb.schema.Operations, op)
	return nil
}

func (mb *ModelBuilder) toIdentifier(n model.AbsoluteIdentifier) model.Identifier {
	return model.Identifier(common.Uncapitalize(mb.toSimpleTypeName(n)))
}

func (mb *ModelBuilder) toCanonicalAbsoluteId(name string) model.AbsoluteIdentifier {
	if strings.Index(name, "#") > 0 {
		return model.AbsoluteIdentifier(name)
	}
	return model.AbsoluteIdentifier(mb.ns + "#" + name)
}

func (mb *ModelBuilder) toSimpleTypeName(n model.AbsoluteIdentifier) string {
	lst := strings.Split(string(n), "#")
	return lst[len(lst)-1]
}

func (mb *ModelBuilder) toCanonicalTypeName(sch *Schema) model.AbsoluteIdentifier {
	if sch.Ref != "" {
		s := "#/components/schemas/"
		if strings.HasPrefix(sch.Ref, s) {
			s = sch.Ref[len(s):]
		}
		return mb.toCanonicalAbsoluteId(s)
	}
	switch sch.Type {
	case "string":
		switch sch.Format {
		case "date-time":
			return mb.toCanonicalAbsoluteId("base#Timestamp")
		case "binary":
			return mb.toCanonicalAbsoluteId("base#Blob")
		default:
			return mb.toCanonicalAbsoluteId("base#String")
		}
	case "number":
		switch sch.Format {
		case "float":
			return mb.toCanonicalAbsoluteId("base#Float32")
		case "double":
			return mb.toCanonicalAbsoluteId("base#Float64")
		default:
			return mb.toCanonicalAbsoluteId("base#Decimal")
		}
	case "integer":
		switch sch.Format {
		case "int32":
			return mb.toCanonicalAbsoluteId("base#Int32")
		case "int64":
			return mb.toCanonicalAbsoluteId("base#Int64")
		default:
			return mb.toCanonicalAbsoluteId("base#Integer")
		}
	case "boolean":
		return mb.toCanonicalAbsoluteId("base#Bool")
	case "array":
		return mb.toCanonicalAbsoluteId("base#List")
	case "object":
		return mb.toCanonicalAbsoluteId("base#Struct")
	}
	return model.AbsoluteIdentifier("???")
}

func (mb *ModelBuilder) ImportSchema(name string, s *Schema) error {
	td := &model.TypeDef{
		Id: mb.toCanonicalAbsoluteId(name),
	}
	switch s.Type {
	case "object":
		td.Base = model.Struct
		td.Fields = mb.ImportFields(s, s.Required)
	case "array":
		td.Base = model.List
		td.Items = mb.toCanonicalTypeName(s.Items)
	case "string":
		//check t.Format
		td.Base = model.String
	case "number":
		//check t.Format
		td.Base = model.Decimal
	}
	fmt.Println(name, "->", model.Pretty(s), "->", model.Pretty(td))
	return mb.schema.AddTypeDef(td)
}

func containsString(ary []string, val string) bool {
	for _, s := range ary {
		if s == val {
			return true
		}
	}
	return false
}

func (mb *ModelBuilder) ImportFields(s *Schema, required []string) []*model.FieldDef {
	var fields []*model.FieldDef
	for fname, sch := range s.Properties {
		fd := &model.FieldDef{
			Name: model.Identifier(fname),
			Type: mb.toCanonicalTypeName(sch),
		}

		if containsString(required, fname) {
			fd.Required = true
		}
		fields = append(fields, fd)
	}
	return fields
}
