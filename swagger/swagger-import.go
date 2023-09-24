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

	"github.com/boynton/api/smithy"
	"github.com/boynton/data"
)

func Import(path string, ns string) (*smithy.AST, error) {
	model, err := Load(path)
	if err != nil {
		return nil, err
	}
	file := filepath.Base(path)
	ext := filepath.Ext(path)
	name := file[:len(file)-len(ext)]
	fmt.Println("model:", model)
	return ToSmithy(model, ns, name)
}

type Model struct {
	name string
	namespace string
	raw data.Document
}

func (model *Model) String() string {
	return data.Pretty(model.raw)
}

func (model *Model) ImportInfo(ast *smithy.AST) error {
	if info := model.raw.GetDocument("info"); info != nil {
		ast.Metadata = data.NewDocument()
		license := info.GetDocument("license")
		if license != nil {
			ast.Metadata.Put("x_license_name", license.GetString("name"))
			ast.Metadata.Put("x_license_url", license.GetString("url"))
		}
		v := info.GetString("version")
		if v != "" {
			ast.Metadata.Put("version", info.GetString("version"))
		}
		return nil
	}
	return nil
}

func withTrait(traits *data.Document, key string, val interface{}) *data.Document {
	if val != nil {
		if traits == nil {
			traits = data.NewDocument()
		}
		traits.Put(key, val)
	}
	return traits
}

func withCommentTrait(traits *data.Document, val string) (*data.Document, string) {
	if val != "" {
		val = TrimSpace(val)
		traits = withTrait(traits, "smithy.api#documentation", val)
	}
	return traits, ""
}

func (model *Model) ImportService(ast *smithy.AST) error {
	doc := "service imported from swagger"
	if info := model.raw.GetDocument("info"); info != nil {
		s := info.GetString("title")
		if s != "" {
			doc = s
		}
	}
	//grab other metadata

	//enumerate the path/operation. Try to get an enumeration of the operationNames
	//output a service shape
	var traits *data.Document
	traits, _ = withCommentTrait(traits, doc)
	shape := &smithy.Shape{
		Type:   "service",
		Traits: traits,
	}
	ast.PutShape(model.fullName(Capitalize(model.name)), shape)
	return nil
}

func (model *Model) fullName(name string) string {
	return model.namespace + "#" + name
}

func ToSmithy(model *Model, ns string, name string) (*smithy.AST, error) {
	ast := &smithy.AST{
		Smithy: "2",
	}
	model.namespace = ns
	model.name = name
	err := model.ImportInfo(ast)
	if err != nil {
		return nil, err
	}

	model.ImportService(ast)

	//	ast.PutShape(model.fullName("foo"), &smithy.Shape{Type: "string"})

	return ast, nil
}

func Load(path string) (*Model, error) {
	model := &Model{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot read swagger file: %v\n", err)
	}
	err = json.Unmarshal(data, &model.raw)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse swagger file: %v\n", err)
	}
	return model, nil
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
