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
package sadl

import (
	"fmt"

	"github.com/boynton/smithy"
	"github.com/boynton/sadl"
	"github.com/boynton/data"
	sadlsmithy "github.com/boynton/sadl/smithy"
)

func Import(path string) (*smithy.AST, error) {
	model, err := sadl.ParseSadlFile(path, nil)
	if err != nil {
		return nil, err
	}
	model.Namespace = "example"
	fmt.Println("->", data.Pretty(model))
	ast, err := sadlsmithy.FromSADL(model, model.Namespace)
	fmt.Println("THIS:", data.Pretty(ast))
	return ast, err
	//	return sadlsmithy.FromSADL(model, model.Namespace)
}
