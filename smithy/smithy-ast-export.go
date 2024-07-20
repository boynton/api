package smithy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/boynton/api/model"
	"github.com/boynton/data"
)

type AstGenerator struct {
	model.BaseGenerator
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

func (gen *AstGenerator) GenerateOperation(op *model.OperationDef) error {
	return nil
}

func (gen *AstGenerator) GenerateException(op *model.OperationOutput) error {
	return nil
}

func (gen *AstGenerator) GenerateType(td *model.TypeDef) error {
	return nil
}

func SmithyAST(schema *model.Schema, sorted bool) (*AST, error) {
	gen := &AstGenerator{}
	gen.Configure(schema, data.NewObject())
	gen.Sort = sorted
	return gen.ToAST()
}

func (gen *AstGenerator) GenerateResources() (map[string]*Shape, map[model.AbsoluteIdentifier]bool, error) {
	resources := make(map[string]*Shape, 0)
	operations := make(map[model.AbsoluteIdentifier]bool, 0)
	for _, od := range gen.Schema.Operations {
		if od.Resource != "" {
			operations[od.Id] = true
			rezId := strings.Split(string(od.Id), "#")[0] + "#" + od.Resource
			var shape *Shape
			if rez, ok := resources[rezId]; ok {
				shape = rez
			} else {
				shape = &Shape{
					Type: "resource",
				}
				resources[rezId] = shape
			}
			if od.Input != nil {
				for _, fd := range od.Input.Fields {
					if fd.HttpPath {
						if shape.Identifiers == nil {
							shape.Identifiers = NewMap[*ShapeRef]()
						}
						fref := &ShapeRef{
							Target: typeReference(string(fd.Type)),
						}
						shape.Identifiers.Put(string(fd.Name), fref)
					}
				}
			}
			ref := &ShapeRef{
				Target: string(od.Id),
			}
			switch od.Lifecycle {
			case "create":
				shape.Create = ref
			case "read":
				shape.Read = ref
			case "update":
				shape.Update = ref
			case "delete":
				shape.Delete = ref
			case "list":
				shape.List = ref
			case "collection":
				shape.CollectionOperations = append(shape.CollectionOperations, ref)
			default:
				shape.Operations = append(shape.Operations, ref)
			}
		}
	}
	return resources, operations, nil
}

func (gen *AstGenerator) EnsureNamespaced(name string) string {
	if strings.Index(name, "#") < 0 {
		return string(gen.Schema.Namespace) + "#" + name
	}
	return name
}

func (gen *AstGenerator) ToAST() (*AST, error) {
	ast := &AST{
		Smithy: "2",
		//		Metadata: NewNodeValue(),
	}
	resources, resourceOps, err := gen.GenerateResources()
	if err != nil {
		return nil, err
	}
	var resourceKeys []string
	if len(resources) > 0 {
		for k := range resources {
			resourceKeys = append(resourceKeys, k)
		}
		if gen.Sort {
			sort.Slice(resourceKeys, func(i, j int) bool {
				return resourceKeys[i] < resourceKeys[j]
			})
		}
	}
	if gen.Schema.Id != "" {
		//the service we create needs to include resources
		shape := &Shape{
			Type:    "service",
			Version: gen.Schema.Version,
		}
		if gen.Schema.Comment != "" {
			ensureShapeTraits(shape).Put("smithy.api#documentation", gen.Schema.Comment)
		}
		for _, k := range resourceKeys {
			ref := &ShapeRef{
				Target: gen.EnsureNamespaced(k),
			}
			shape.Resources = append(shape.Resources, ref)
		}
		for _, od := range gen.Schema.Operations {
			if _, ok := resourceOps[od.Id]; !ok {
				ref := &ShapeRef{
					Target: string(od.Id),
				}
				shape.Operations = append(shape.Operations, ref)
			}
		}
		ast.PutShape(string(gen.Schema.Id), shape)
	}
	for _, k := range resourceKeys {
		v := resources[k]
		ast.PutShape(k, v)
	}
	for _, op := range gen.Schema.Operations {
		err := gen.AddShapesFromOperation(ast, op)
		if err != nil {
			return nil, err
		}
	}
	for _, edef := range gen.Schema.Exceptions {
		shapeId := string(edef.Id)
		shape, err := gen.shapeFromOpOutput(edef, true)
		if err != nil {
			return nil, err
		}
		if ast.GetShape(shapeId) == nil {
			ast.PutShape(shapeId, shape)
		}
	}
	for _, td := range gen.Schema.Types {
		shapeId, shape, err := gen.ShapeFromType(td)
		if err != nil {
			return nil, err
		}
		if ast.GetShape(shapeId) == nil {
			ast.PutShape(shapeId, shape)
		}
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
	if op.Comment != "" {
		ensureShapeTraits(shape).Put("smithy.api#documentation", op.Comment)
	}
	switch op.HttpMethod {
	case "GET":
		ensureShapeTraits(shape).Put("smithy.api#readonly", NewNodeValue())
	case "DELETE", "PUT":
		ensureShapeTraits(shape).Put("smithy.api#idempotent", NewNodeValue())
	}

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
	if op.Output.Fields != nil {
		outputShapeId = string(op.Output.Id)
		if outputShapeId == "" {
			outputShapeId = string(op.Id) + "Output"
		}
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
	if op.Exceptions != nil {
		for _, errId := range op.Exceptions {
			shape.Errors = append(shape.Errors, &ShapeRef{Target: string(errId)})
		}
	}
	if op.Examples != nil {
		var examples []any
		for _, ex := range op.Examples {
			smex := make(map[string]any, 0)
			smex["title"] = ex.Title
			if ex.Input != nil {
				smex["input"] = ex.Input
			}
			if ex.Output != nil {
				smex["output"] = ex.Output
			}
			if ex.Error != nil {
				smexerr := make(map[string]any, 0)
				smexerr["shapeId"] = string(ex.Error.ShapeId)
				smexerr["content"] = ex.Error.Entity
				smex["error"] = smexerr
				smex["allowConstraintErrors"] = true
			}
			examples = append(examples, smex)
		}
		if len(examples) > 0 {
			ensureShapeTraits(shape).Put("smithy.api#examples", examples)
		}
	}
	ast.PutShape(string(op.Id), shape)
	if inputShape != nil {
		ast.PutShape(inputShapeId, inputShape)
	}
	if outputShape != nil {
		ast.PutShape(outputShapeId, outputShape)
	}
	for _, errId := range errShapeIds {
		errShape := errShapes[errId]
		prev := ast.GetShape(errId)
		if prev != nil {
			if !model.Equivalent(prev, errShape) {
				fmt.Println("prev:", model.Pretty(prev))
				fmt.Println("errShape:", model.Pretty(errShape))
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
	var fields []*model.OperationInputField
	for _, fd := range input.Fields {
		fields = append(fields, fd)
	}
	if gen.Sort {
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Name < fields[j].Name
		})
	}

	for _, fd := range fields {
		ftype := typeReference(string(fd.Type))
		member := &Member{
			Target: ftype,
		}
		if fd.Required {
			//note: import form Smithy forces required on any httpPayload field
			ensureMemberTraits(member).Put("smithy.api#required", NewNodeValue())
		}
		if fd.HttpHeader != "" {
			ensureMemberTraits(member).Put("smithy.api#httpHeader", string(fd.HttpHeader))
		} else if fd.HttpQuery != "" {
			ensureMemberTraits(member).Put("smithy.api#httpQuery", string(fd.HttpQuery))
		} else if fd.HttpPath {
			ensureMemberTraits(member).Put("smithy.api#httpLabel", NewNodeValue())
		} else if fd.HttpPayload {
			ensureMemberTraits(member).Put("smithy.api#httpPayload", NewNodeValue())
		}
		if fd.Default != nil {
			ensureMemberTraits(member).Put("smithy.api#default", AsNodeValue(fd.Default))
		}
		if fd.MinValue != nil || fd.MaxValue != nil {
			n := NewNodeValue()
			if fd.MinValue != nil {
				n.Put("min", fd.MinValue.AsInt64())
			}
			if fd.MaxValue != nil {
				n.Put("max", fd.MaxValue.AsInt64())
			}
			ensureMemberTraits(member).Put("smithy.api#range", n)
		}
		if fd.MinSize != 0 || fd.MaxSize != 0 {
			n := NewNodeValue()
			if fd.MinSize != 0 {
				n.Put("min", fd.MinSize)
			}
			if fd.MaxSize != 0 {
				n.Put("max", fd.MaxSize)
			}
			ensureMemberTraits(member).Put("smithy.api#length", n)
		}
		if fd.Pattern != "" {
			ensureMemberTraits(member).Put("smithy.api#pattern", fd.Pattern)
		}
		members.Put(string(fd.Name), member)
	}
	shape.Members = members
	ensureShapeTraits(shape).Put("smithy.api#documentation", input.Comment)
	ensureShapeTraits(shape).Put("smithy.api#input", NewNodeValue())
	return shape, nil
}

func (gen *AstGenerator) shapeFromOpOutput(output *model.OperationOutput, isException bool) (*Shape, error) {
	shape := &Shape{
		Type: "structure",
	}
	var fields []*model.OperationOutputField
	shape.Members = NewMap[*Member]()
	for _, fd := range output.Fields {
		fields = append(fields, fd)
	}
	if gen.Sort {
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Name < fields[j].Name
		})
	}
	for _, fd := range fields {
		ftype := typeReference(string(fd.Type))
		member := &Member{
			Target: ftype,
		}
		if fd.HttpHeader != "" {
			ensureMemberTraits(member).Put("smithy.api#httpHeader", fd.HttpHeader)
		} else if fd.HttpPayload {
			ensureMemberTraits(member).Put("smithy.api#httpPayload", NewNodeValue())
		}
		shape.Members.Put(string(fd.Name), member)
	}
	if isException {
		fault := "server"
		if output.HttpStatus < 500 {
			fault = "client"
		}
		ensureShapeTraits(shape).Put("smithy.api#error", fault)
		ensureShapeTraits(shape).Put("smithy.api#httpError", output.HttpStatus)
	} else {
		ensureShapeTraits(shape).Put("smithy.api#output", NewNodeValue())
	}
	ensureShapeTraits(shape).Put("smithy.api#documentation", output.Comment)
	return shape, nil
}

func (gen *AstGenerator) ShapeFromType(td *model.TypeDef) (string, *Shape, error) {
	var id string
	var shape *Shape
	var err error
	switch td.Base {
	case model.BaseType_Struct:
		id, shape, err = gen.ShapeFromStruct(td)
	case model.BaseType_List:
		id, shape, err = gen.ShapeFromList(td)
	case model.BaseType_Map:
		id, shape, err = gen.ShapeFromMap(td)
	case model.BaseType_String:
		id, shape, err = gen.ShapeFromString(td)
	case model.BaseType_Int8, model.BaseType_Int16, model.BaseType_Int32, model.BaseType_Int64, model.BaseType_Float32, model.BaseType_Float64, model.BaseType_Decimal, model.BaseType_Integer:
		id, shape, err = gen.ShapeFromNumber(td)
	case model.BaseType_Enum:
		id, shape, err = gen.ShapeFromEnum(td)
	case model.BaseType_Timestamp:
		id, shape, err = gen.ShapeFromTimestamp(td)
	case model.BaseType_Union:
		id, shape, err = gen.ShapeFromUnion(td)
	case model.BaseType_Bool:
		id, shape, err = gen.ShapeFromBool(td)
	case model.BaseType_Any:
		id, shape, err = gen.ShapeFromAny(td)
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

func (gen *AstGenerator) ShapeFromAny(td *model.TypeDef) (string, *Shape, error) {
	shape := &Shape{
		Type: "document",
	}
	return string(td.Id), shape, nil
}

func (gen *AstGenerator) ShapeFromNumber(td *model.TypeDef) (string, *Shape, error) {
	shape := Shape{}
	switch td.Base {
	case model.BaseType_Int8:
		shape.Type = "byte"
	case model.BaseType_Int16:
		shape.Type = "short"
	case model.BaseType_Int32:
		shape.Type = "integer"
	case model.BaseType_Int64:
		shape.Type = "long"
	case model.BaseType_Float32:
		shape.Type = "float"
	case model.BaseType_Float64:
		shape.Type = "double"
	case model.BaseType_Decimal:
		shape.Type = "bigDecimal"
	case model.BaseType_Integer:
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

func (gen *AstGenerator) ShapeFromBool(td *model.TypeDef) (string, *Shape, error) {
	shape := &Shape{
		Type: "boolean",
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
		Type:    "enum",
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
	var fields []*model.FieldDef
	shape.Members = NewMap[*Member]()
	for _, fd := range td.Fields {
		fields = append(fields, fd)
	}
	if gen.Sort {
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Name < fields[j].Name
		})
	}
	for _, fd := range fields {
		ftype := typeReference(string(fd.Type))
		member := &Member{
			Target: ftype,
		}
		if fd.Comment != "" {
			ensureMemberTraits(member).Put("smithy.api#documentation", fd.Comment)
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
		member := &Member{
			Target: ftype,
		}
		members.Put(string(fd.Name), member)
	}
	shape.Members = members
	return string(td.Id), shape, nil
}
