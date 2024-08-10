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

func (gen *IdlGenerator) GenerateResource(rez *model.ResourceDef) error {
	if gen.ast == nil {
		ast, err := SmithyAST(gen.Schema, gen.Sort, "")
		if err != nil {
			return err
		}
		gen.ast = ast
	}
	gen.Emit(gen.ast.IDLForResourceShape(string(rez.Id), gen.Decorator))
	return nil
}

func (gen *IdlGenerator) GenerateOperation(op *model.OperationDef) error {
	if gen.ast == nil {
		ast, err := SmithyAST(gen.Schema, gen.Sort, "")
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
		ast, err := SmithyAST(gen.Schema, gen.Sort, "")
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
		ast, err := SmithyAST(gen.Schema, gen.Sort, "")
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

	ast, err := SmithyAST(schema, gen.Sort, config.GetString("namespace"))
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

/*
   func (ast *AST) flatten(id string, ns string) (string, *Shape) {
	shape := ast.GetShape(id)
	sid := ast.forceNamespace(id, ns)
	fmt.Print("flatten this:", pretty(shape))
	panic("here")
	switch shape.Type {
	case "service":
		for _, op := range shape.Operations {
			op.Target = ast.forceNamespace(op.Target, ns)
		}
		for _, rez := range shape.Resources {
			rez.Target = ast.forceNamespace(rez.Target, ns)
		}
	case "structure":
		for _, k := range shape.Members.Keys() {
			m := shape.Members.Get(k)
			fmt.Println("fix this:", m)
		}
	}
	return sid, shape
}
*/

func (ast *AST) forceNamespace(id model.AbsoluteIdentifier, ns string) string {
	sid := string(id)
	if ns == "" {
		return sid
	}
	name := stripNamespace(sid)
	return ns + "#" + name
}

/*
   func (ast *AST) FlattenNamespacesTo(ns string) error {
	newShapes := NewMap[*Shape]()
	for _, shapeId := range ast.Shapes.Keys() {
		sid, shape := ast.flatten(shapeId, ns)
		newShapes.Put(sid, shape)
	}
	ast.Shapes = newShapes
	return nil

}
*/
