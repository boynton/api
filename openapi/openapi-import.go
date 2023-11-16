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
	fmt.Println("openapi:", model.Pretty(mb.openapi))
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
	/*
		comp := mb.openapi.Components
		for k, s := range comp.Schemas {
			mb.ImportSchema(k, s)
		}
	*/
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
	op := &model.OperationDef{
		Id:         mb.toCanonicalAbsoluteId(common.Capitalize(pop.OperationId)),
		HttpMethod: method,
		HttpUri:    path,
		Comment:    mb.opComment(pop),
		Input:      &model.OperationInput{},
		Output:     &model.OperationOutput{},
	}
	//inputs
	for _, param := range pop.Parameters {
		ftype := mb.toCanonicalTypeName(param.Schema)
		fname := param.Name
		fd := &model.OperationInputField{
			Name:        model.Identifier(fname),
			Type:        ftype,
			Required:    true,
			HttpPayload: true,
		}
		op.Input.Fields = append(op.Input.Fields, fd)
		fmt.Println("param!:", param)
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

	//output
	//exceptions

	mb.schema.Operations = append(mb.schema.Operations, op)
	fmt.Printf("http %s %q : %s\n", method, path, model.Pretty(op))
	return nil
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
		return mb.toCanonicalAbsoluteId("base.String")
		//check formats
	case "number":
		return mb.toCanonicalAbsoluteId("base.Decimal")
		//check formats
	case "integer":
		return mb.toCanonicalAbsoluteId("base.Int32")
		//check formats
	case "boolean":
		return mb.toCanonicalAbsoluteId("base.Bool")
	case "array":
		return mb.toCanonicalAbsoluteId("base.List")
	case "object":
		return mb.toCanonicalAbsoluteId("base.Struct")
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
	case "array":
		td.Base = model.List
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
