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
	"fmt"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/boynton/data"
	"github.com/boynton/api/model"
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
	fmt.Println("swagger:", swagger)
	return ImportSwagger(swagger, ns, name)
}

type Swagger struct {
	name string
	namespace string
	raw data.Object
}

func (swagger *Swagger) String() string {
	return data.Pretty(swagger.raw)
}

func ImportSwagger(swagger *Swagger, ns string, name string) (*model.Schema, error) {
	if swagger.raw.GetString("swagger") != "2.0" {
		return nil, fmt.Errorf("Not a valid swagger document. Only version 2.0 is supported")
	}
	schema := model.NewSchema()
	err := swagger.ImportInfo(schema, ns)
	if err != nil {
		return nil, err
	}
	err = swagger.ImportService(schema)
	return schema, err
}


func (swagger *Swagger) ImportInfo(schema *model.Schema, ns string) error {
	if info := swagger.raw.GetObject("info"); info != nil {
		name := info.GetString("title")
		if name == "" {
			name = "Untitled"
		}
		schema.Id = model.AbsoluteIdentifier(ns + "#" + name)
		schema.Version = info.GetString("version")
		schema.Comment = info.GetString("description")
		schema.Base = swagger.raw.GetString("basePath")
		license := info.GetObject("license")
		if license != nil {
			schema.Metadata = data.NewObject()
			schema.Metadata.Put("x_license_name", license.GetString("name"))
			schema.Metadata.Put("x_license_url", license.GetString("url"))
		}
		return nil
	}
	return nil
}

func (swagger *Swagger) ImportService(schema *model.Schema) error {
	//paths := swagger.raw.GetObject("paths")
	defs := swagger.raw.GetObject("definitions")
	for _, k := range defs.Keys() {
		def := defs.GetObject(k)
		if def.Has("properties") {
			err := swagger.ImportStruct(k, schema, def)
			if err != nil {
				return err
			}
		}
	}
	//enumerate the path/operation. Try to get an enumeration of the operationNames
	return nil
}

func (swagger *Swagger) toCanonicalAbsoluteId(name string) model.AbsoluteIdentifier {
	return model.AbsoluteIdentifier(gen.ns + "#" + name)
}

func (swagger *Swagger) toCanonicalTypeName(sch *data.Object) model.AbsoluteIdentifier {
	ref := sch.GetString("$ref")
	if ref != "" {
		return
	}
	tname := sch.GetString("type")
	name := "?"
	switch name {
	case "number":
		name = "base.Decimal"
		//check formats
	case "string":
		name = "base.String"
		//check formats
	case "boolean":
		name = "base.Bool"
	case "object":
		//recurse, and build name
		name
	}
	return model.AbsoluteIdentifier(name)
}

func (swagger *Swagger) ImportStruct(name string, schema *model.Schema, def *data.Object) error {
	td := &model.TypeDef{
		Id: swagger.toCanonicalAbsoluteId(name),
		Base: model.Struct,
	}
	props := def.GetObject("properties")
	for _, name := range props.Keys() {
		fd := &model.FieldDef{
			Name: name,
		}
		v := props.GetObject(name)
		fd.Type = swagger.toCanonicalTypeName(v.GetString("type"))
		if v.Traits != nil {
					if v.Traits.Get("smithy.api#required") != nil {
						fd.Required = true
					}
					comment := v.Traits.GetString("smithy.api#documentation")
					if comment != "" {
						fd.Comment = comment
					}
				}
				td.Fields = append(td.Fields, fd)
			}
	
	fmt.Println("struct: ", name, def)
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
