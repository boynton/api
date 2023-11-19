package smithy

import(
	"fmt"
	
	"github.com/boynton/data"
	"github.com/boynton/api/model"
)

type IdlGenerator struct {
	AstGenerator
}

func (gen *IdlGenerator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}

	ast, err := gen.ToAST()
	if err != nil {
		return err
	}

	//fixme: preserve smithy metadata.
	for _, ns := range ast.Namespaces() {
		fname := gen.FileName(ns, ".smithy")
		sep := fmt.Sprintf("\n// ===== File(%q)\n\n", fname)
		s := ast.IDL(ns)
		err := gen.Write(s, fname, sep)
		if err != nil {
			return err
		}
	}
	return nil
}
