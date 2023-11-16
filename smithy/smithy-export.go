package smithy

import(
	"github.com/boynton/data"
	"github.com/boynton/api/model"
	//	"github.com/boynton/api/common"
)

type IdlGenerator struct {
	AstGenerator
}

func (gen *IdlGenerator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	//generate one file per namespace. For outdir == "", concatenate with separator indicating intended filename
	//fixme: preserve metadata. Smithy IDL is problematic for that, since metadata is not namespaced, and gets merged
	//on assembly. Should each namespaced IDL get all metadata? none?
	/*
	for _, ns := range ast.Namespaces() {
		fname := gen.FileName(ns, ".smithy")
		sep := fmt.Sprintf("\n// ===== File(%q)\n\n", fname)
		s := ast.IDL(ns)
		err := gen.Emit(s, fname, sep)
		if err != nil {
			return err
		}
	}*/
	panic("FIX ME: smithy.IdlGenerator.Generate")
	return nil
}

