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

	"github.com/boynton/api/smithy"
)

func Import(path string, ns string) (*smithy.AST, error) {
	model, err := Load(path)
	if err != nil {
		return nil, err
	}
	return ToSmithy(model, ns)
}

func ToSmithy(model *Model, ns string) (*smithy.AST, error) {
	return nil, fmt.Errorf("openapi.ToSmithy() NYI")
}
