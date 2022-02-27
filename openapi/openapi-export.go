package openapi

import (
	"fmt"

	"github.com/boynton/data"
	"github.com/boynton/smithy"
)

type Generator struct {
	smithy.BaseGenerator
}

func (gen *Generator) Generate(ast *smithy.AST, config *data.Object) error {
	return fmt.Errorf("openapi.Generator NYI")
}
