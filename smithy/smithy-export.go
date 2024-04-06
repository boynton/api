package smithy

import (
	"fmt"

	"github.com/boynton/api/model"
	"github.com/boynton/data"
)

type IdlGenerator struct {
	model.BaseGenerator
	Decorator *model.Decorator
	ast       *AST
}

func (gen *IdlGenerator) GenerateResource(id string) error {
	if gen.ast == nil {
		ast, err := SmithyAST(gen.Schema, gen.Sort)
		if err != nil {
			return err
		}
		gen.ast = ast
	}
	gen.Emit(gen.ast.IDLForResourceShape(id, gen.Decorator))
	return nil
}

func (gen *IdlGenerator) GenerateOperation(op *model.OperationDef) error {
	if gen.ast == nil {
		ast, err := SmithyAST(gen.Schema, gen.Sort)
		if err != nil {
			return err
		}
		gen.ast = ast
	}
	gen.Emit(gen.ast.IDLForOperationShape(string(op.Id), gen.Decorator))
	return nil
}

func (gen *IdlGenerator) GenerateType(op *model.TypeDef) error {
	if gen.ast == nil {
		ast, err := SmithyAST(gen.Schema, gen.Sort)
		if err != nil {
			return err
		}
		gen.ast = ast
	}
	gen.Emit(gen.ast.IDLForTypeShape(string(op.Id), gen.Decorator))
	return nil
}

func (gen *IdlGenerator) GenerateException(op *model.OperationOutput) error {
	if gen.ast == nil {
		ast, err := SmithyAST(gen.Schema, gen.Sort)
		if err != nil {
			return err
		}
		gen.ast = ast
	}
	gen.Emit(gen.ast.IDLForTypeShape(string(op.Id), gen.Decorator))
	return nil
}

func (gen *IdlGenerator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}

	ast, err := SmithyAST(schema, gen.Sort)
	if err != nil {
		return err
	}

	//fixme: preserve smithy metadata.
	needsSep := len(ast.Namespaces()) != 1
	for _, ns := range ast.Namespaces() {
		fname := gen.FileName(ns, ".smithy")
		sep := ""
		if needsSep {
			sep = fmt.Sprintf("\n// ===== File(%q)\n\n", fname)
		}
		s := ast.IDL(ns)
		err := gen.Write(s, fname, sep)
		if err != nil {
			return err
		}
	}
	return nil
}
