package openapi

import (
	"fmt"

	"github.com/boynton/smithy"
)

func Import(path string) (*smithy.AST, error) {
	model, err := Load(path)
	if err != nil {
		return nil, err
	}
	return ToSmithy(model)
}

func ToSmithy(model *Model) (*smithy.AST, error) {
	return nil, fmt.Errorf("openapi.ToSmithy() NYI")
}
