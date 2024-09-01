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

	"github.com/boynton/api/model"
)

type Generator struct {
	model.BaseGenerator
}

func (gen *Generator) Generate(schema *model.Schema) error {
	return fmt.Errorf("swagger.Generate NYI")
}

func (gen *Generator) GenerateOperation(op *model.OperationDef) error {
	return nil
}

func (gen *Generator) GenerateResource(op *model.ResourceDef) error {
	return nil
}

func (gen *Generator) GenerateException(exc *model.OperationOutput) error {
	return nil
}

func (gen *Generator) GenerateType(td *model.TypeDef) error {
	return nil
}

