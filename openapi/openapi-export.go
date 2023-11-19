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

	"github.com/boynton/api/common"
	"github.com/boynton/api/model"
	"github.com/boynton/data"
)

type Generator struct {
	common.BaseGenerator
}

func (gen *Generator) Generate(schema *model.Schema, config *data.Object) error {
	return fmt.Errorf("openapi.Generator NYI")
}

func (gen *Generator) GenerateOperation(op *model.OperationDef) error {
	return nil
}

func (gen *Generator) GenerateType(td *model.TypeDef) error {
	return nil
}
