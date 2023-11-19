package smithy

import(
	"github.com/boynton/data"
	"github.com/boynton/api/model"
	"github.com/boynton/api/common"
)

type AstGenerator struct {
	common.BaseGenerator
	ast *AST
}

func (gen *AstGenerator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.ast, err = gen.ToAST()
	if err != nil {
		return err
	}
	return gen.Write(model.Pretty(gen.ast), "model.json", "")
}

func SmithyAST(schema *model.Schema) (*AST, error) {
	gen := &AstGenerator{}
	gen.Configure(schema, data.NewObject())
	return gen.ToAST()
}

func (gen *AstGenerator) ToAST() (*AST, error) {
	ast := &AST{
		Smithy:   "2",
		//		Metadata: NewNodeValue(),
	}
	if gen.Schema.Id != "" {
		shape := &Shape{
			Type: "service",
			Version: gen.Schema.Version,
		}
		if gen.Schema.Comment != "" {
			ensureShapeTraits(shape).Put("smithy.api#documentation", gen.Schema.Comment)
		}
		for _, od := range gen.Schema.Operations {
			ref := &ShapeRef{
				Target: string(od.Id),
			}
			shape.Operations = append(shape.Operations, ref)
		}
		ast.PutShape(string(gen.Schema.Id), shape)
	}
	for _, op := range gen.Schema.Operations {
		err := gen.AddShapesFromOperation(ast, op)
		//shapeId, shape, err := gen.ShapeFromOperation(ast, od)
		if err != nil {
			return nil, err
		}
	}
	for _, td := range gen.Schema.Types {
		shapeId, shape, err := gen.ShapeFromType(td)
		if err != nil {
			return nil, err
		}
		ast.PutShape(shapeId, shape)
	}
	gen.ast = ast
	return ast, nil
}

func (gen *AstGenerator) AddShapesFromOperation(ast *AST, op *model.OperationDef) error {
	var err error
	var inputShapeId string
	var inputShape *Shape
	var outputShapeId string
	var outputShape *Shape
	var errShapeIds []string
	errShapes := make(map[string]*Shape, 0)
	shape := &Shape{
		Type: "operation",
	}
	status := 204 //no content
	if op.Output != nil {
		status = int(op.Output.HttpStatus)
	}
	ensureShapeTraits(shape).Put("smithy.api#http", httpTrait(op.HttpMethod, op.HttpUri, status))

	if op.Input != nil {
		inputShapeId = string(op.Id) + "Input"
		shape.Input = &ShapeRef{
			Target: inputShapeId,
		}
		inputShape, err = gen.shapeFromOpInput(op.Input)
		if err != nil {
			return err
		}
	} else {
		shape.Input = &ShapeRef{
			Target: "smithy.api#Unit",
		}
	}
	if op.Output != nil {
		if op.Output.Id != "" {
			outputShapeId = string(op.Id) + "Output"
			shape.Output = &ShapeRef{
				Target: outputShapeId,
			}
			outputShape, err = gen.shapeFromOpOutput(op.Output, false)
			if err != nil {
				return err
			}
		} else {
			shape.Output = &ShapeRef{
				Target: "smithy.api#Unit",
			}
		}
	}
	if op.Exceptions != nil {
		for _, ed := range op.Exceptions {
			errId := string(ed.Id)
			shape.Errors = append(shape.Errors, &ShapeRef{Target: errId})
			errShape, err := gen.shapeFromOpOutput(ed, true)
			if err != nil {
				return err
			}
			errShapeIds = append(errShapeIds, errId)
			errShapes[errId] = errShape
		}
	}
	ast.PutShape(string(op.Id), shape)
	if inputShape != nil {
		ast.PutShape(inputShapeId, inputShape)
	}
	if outputShape != nil {
		ast.PutShape(outputShapeId, outputShape)
	}
	for _,  errId := range errShapeIds {
		errShape := errShapes[errId]
		prev := ast.GetShape(errId)
		if prev != nil {
			if !model.Equivalent(prev, errShape) {
				panic("reused operation error shape but different definition: " + errId)
			}
		} else {
			ast.PutShape(errId, errShape)
		}
	}
	return nil
}

func (gen *AstGenerator) shapeFromOpInput(input *model.OperationInput) (*Shape, error) {
	shape := &Shape{
		Type: "structure",
	}
	members := NewMap[*Member]()
	for _, fd := range input.Fields {
		ftype := typeReference(string(fd.Type))
		member := &Member{
			Target: ftype,
		}
		if fd.Required {
			//note: import form Smithy forces required on any httpPayload field
			ensureMemberTraits(member).Put("smithy.api#required", NewNodeValue())
		}
		if fd.HttpHeader != "" {
			ensureMemberTraits(member).Put("smithy.api#httpHeader", fd.HttpHeader)
		} else if fd.HttpQuery != "" {
			ensureMemberTraits(member).Put("smithy.api#httpQuery", fd.HttpQuery)
		} else if fd.HttpPath {
			ensureMemberTraits(member).Put("smithy.api#httpLabel", NewNodeValue())
		} else if fd.HttpPayload {
			ensureMemberTraits(member).Put("smithy.api#httpPayload", NewNodeValue())
		}
		members.Put(string(fd.Name), member)
	}
	shape.Members = members
	ensureShapeTraits(shape).Put("smithy.api#input", NewNodeValue())
	return shape, nil
}

func (gen *AstGenerator) shapeFromOpOutput(output *model.OperationOutput, isException bool) (*Shape, error) {
	shape := &Shape{
		Type: "structure",
	}
	members := NewMap[*Member]()
	for _, fd := range output.Fields {
		ftype := typeReference(string(fd.Type))
		member := &Member{
			Target: ftype,
		}
		if fd.HttpHeader != "" {
			ensureMemberTraits(member).Put("smithy.api#httpHeader", fd.HttpHeader)
		} else if fd.HttpPayload {
			ensureMemberTraits(member).Put("smithy.api#httpPayload", NewNodeValue())
		}
		members.Put(string(fd.Name), member)
	}
	shape.Members = members
	if isException {
		ensureShapeTraits(shape).Put("smithy.api#error", "client") //!
		ensureShapeTraits(shape).Put("smithy.api#httpError", output.HttpStatus)
	} else {
		ensureShapeTraits(shape).Put("smithy.api#output", NewNodeValue())
	}
	return shape, nil	
}

func (gen *AstGenerator) ShapeFromType(td *model.TypeDef) (string, *Shape, error) {
	var id string
	var shape *Shape
	var err error
	switch td.Base {
	case model.Struct:
		id, shape, err = gen.ShapeFromStruct(td)
	case model.List:
		id, shape, err = gen.ShapeFromList(td)
	case model.Map:
		id, shape, err = gen.ShapeFromMap(td)
	case model.String:
		id, shape, err = gen.ShapeFromString(td)
	case model.Int8, model.Int16, model.Int32, model.Int64, model.Float32, model.Float64, model.Decimal, model.Integer:
		id, shape, err = gen.ShapeFromNumber(td)
	case model.Enum:
		id, shape, err = gen.ShapeFromEnum(td)
	case model.Timestamp:
		id, shape, err = gen.ShapeFromTimestamp(td)
	case model.Union:
		id, shape, err = gen.ShapeFromUnion(td)
	default:
		panic("handle this type:" + model.Pretty(td))
	}
	if td.Comment != "" {
		ensureShapeTraits(shape).Put("smithy.api#documentation", td.Comment)
	}
	return id, shape, err
}

func (gen *AstGenerator) ShapeFromString(td *model.TypeDef) (string, *Shape, error) {
	shape := &Shape{
		Type: "string",
	}
	if td.Pattern != "" {
		ensureShapeTraits(shape).Put("smithy.api#pattern", td.Pattern)
	}
	return string(td.Id), shape, nil
}

func (gen *AstGenerator) ShapeFromNumber(td *model.TypeDef) (string, *Shape, error) {
	shape := Shape{}
	switch td.Base {
	case model.Int8:
		shape.Type = "byte"
	case model.Int16:
		shape.Type = "short"
	case model.Int32:
		shape.Type = "integer"
	case model.Int64:
		shape.Type = "long"
	case model.Float32:
		shape.Type = "float"
	case model.Float64:
		shape.Type = "double"
	case model.Decimal:
		shape.Type = "bigDecimal"
	case model.Integer:
		panic("whoa!")
		shape.Type = "bigInteger"
	}
	if td.MinValue != nil || td.MaxValue != nil {
		ensureShapeTraits(&shape).Put("smithy.api#range", rangeTrait(td.MinValue, td.MaxValue))
	}
	return string(td.Id), &shape, nil
}

func (gen *AstGenerator) ShapeFromTimestamp(td *model.TypeDef) (string, *Shape, error) {
	shape := &Shape{
		Type: "timestamp",
	}
	return string(td.Id), shape, nil
}

func ensureShapeTraits(shape *Shape) *NodeValue {
	if shape.Traits == nil {
		shape.Traits = NewNodeValue()
	}
	return shape.Traits
}

func ensureMemberTraits(member *Member) *NodeValue {
	if member.Traits == nil {
		member.Traits = NewNodeValue()
	}
	return member.Traits
}

func rangeTrait(min *data.Decimal, max *data.Decimal) *NodeValue {
	if min == nil && max == nil {
		return nil
	}
	l := NewNodeValue()
	if min != nil {
		l.Put("min", min.AsFloat64())
	}
	if max != nil {
		l.Put("max", max.AsFloat64())
	}
	return l
}

func httpTrait(method, path string, code int) *NodeValue {
	t := NewNodeValue()
	t.Put("method", method)
	t.Put("uri", path)
	if code != 0 {
		t.Put("code", code)
	}
	return t
}

//func typeReference(ns string, td *model.TypeDef) string {
//	return typeReferenceByName(ns, string(td.Id))
//}

func typeReference(name string) string {
	switch name {
	case "base#Bool":
		return "smithy.api#Boolean"
	case "base#Int8":
		return "smithy.api#Byte"
	case "base#Int16":
		return "smithy.api#Short"
	case "base#Int32":
		return "smithy.api#Integer"
	case "base#Int64":
		return "smithy.api#Long"
	case "base#Float32":
		return "smithy.api#Float"
	case "base#Float64":
		return "smithy.api#Double"
	case "base#Integer":
		return "smithy.api#BigInteger"
	case "base#Decimal":
		return "smithy.api#BigDecimal"
	case "base#Timestamp":
		return "smithy.api#Timestamp"
	case "base#Bytes":
		return "smithy.api#Blob"
	case "base#String":
		return "smithy.api#String"
	case "base#List":
		return "smithy.api#List"
	case "base#Map":
		return "smithy.api#Map"
	default:
		return name
	}
}

func (gen *AstGenerator) ShapeFromEnum(td *model.TypeDef) (string, *Shape, error) {
	shape := &Shape{
		Type: "enum",
		Members: NewMap[*Member](),
	}
	for _, el := range td.Elements {
		mem := &Member{
			Target: "smithy.api#Unit",
		}
		if el.Value != "" {
			ensureMemberTraits(mem).Put("smithy.api#enumValue", el.Value)
		}
		shape.Members.Put(string(el.Symbol), mem)
	}
	return string(td.Id), shape, nil
}

func (gen *AstGenerator) ShapeFromList(td *model.TypeDef) (string, *Shape, error) {
	shape := &Shape{
		Type: "list",
	}
	itype := typeReference(string(td.Items))
	shape.Member = &Member{
		Target: itype,
	}
	return string(td.Id), shape, nil
}

func (gen *AstGenerator) ShapeFromMap(td *model.TypeDef) (string, *Shape, error) {
	shape := &Shape{
		Type: "map",
	}
	itype := typeReference(string(td.Items))
	shape.Member = &Member{
		Target: itype,
	}
	return string(td.Id), shape, nil
}

func (gen *AstGenerator) ShapeFromStruct(td *model.TypeDef) (string, *Shape, error) {
	shape := &Shape{
		Type: "structure",
	}
	members := NewMap[*Member]()
	for _, fd := range td.Fields {
		ftype := typeReference(string(fd.Type))
		/*
		switch ftype {
		case "List":
			ftype = listTypeReference(model, ns, shapes, tname, fd)
		case "Map":
			ftype = mapTypeReference(model, ns, shapes, tname, fd)
		case "Enum":
			ftype = enumTypeReference(model, ns, shapes, tname, fd)
		}
		*/
		member := &Member{
			Target: ftype,
		}
		if fd.Required {
			ensureMemberTraits(member).Put("smithy.api#required", NewNodeValue())
		}
		members.Put(string(fd.Name), member)
	}
	shape.Members = members
	return string(td.Id), shape, nil
}

func (gen *AstGenerator) ShapeFromUnion(td *model.TypeDef) (string, *Shape, error) {
	shape := &Shape{
		Type: "union",
	}
	members := NewMap[*Member]()
	for _, fd := range td.Fields {
		ftype := typeReference(string(fd.Type))
		/*
		switch ftype {
		case "List":
			ftype = listTypeReference(model, ns, shapes, tname, fd)
		case "Map":
			ftype = mapTypeReference(model, ns, shapes, tname, fd)
		case "Enum":
			ftype = enumTypeReference(model, ns, shapes, tname, fd)
		}
		*/
		member := &Member{
			Target: ftype,
		}
		members.Put(string(fd.Name), member)
	}
	shape.Members = members
	return string(td.Id), shape, nil
}

/*
func defineShapeFromTypeSpec(model *sadl.Model, ns string, shapes *smithylib.Shapes, ts *sadl.TypeSpec, name string, comment string, annos map[string]string) error {
	var shape smithylib.Shape
	switch ts.Type {
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		shape = shapeFromNumber(ts)
	case "String":
		shape = shapeFromString(ts)
	case "Enum":
		shape = shapeFromEnum(ts)
	case "Struct":
		shape = shapeFromStruct(model, ns, shapes, name, ts)
	case "Array":
		shape = shapeFromArray(model, ns, shapes, name, ts, annos)
	case "Union":
		shape = shapeFromUnion(model, ns, shapes, name, ts)
	case "Map":
		shape = shapeFromMap(model, ns, shapes, name, ts)
	case "UUID":
		shape = *uuidShape()
	default:
		fmt.Println("So far:", sadl.Pretty(model))
		panic("handle this type:" + sadl.Pretty(ts))
	}
	if comment != "" {
		ensureShapeTraits(&shape).Put("smithy.api#documentation", comment)
	}
	if annos != nil {
		for k, v := range annos {
			switch k {
			case "x_tags":
				ensureShapeTraits(&shape).Put("smithy.api#tags", strings.Split(v, ","))
			case "x_sensitive":
				ensureShapeTraits(&shape).Put("smithy.api#sensitive", true)
			case "x_deprecated":
				dep := make(map[string]interface{}, 0)
				if v != "" {
					n := strings.Index(v, "|")
					if n >= 0 {
						dep["since"] = v[:n]
						dep["message"] = v[n+1:]
					} else {
						dep["message"] = v
					}
					ensureShapeTraits(&shape).Put("smithy.api#deprecated", dep)
				}
			}
		}
	}
	shapes.Put(ns+"#"+name, &shape)
	return nil
}
*/
