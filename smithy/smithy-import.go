package smithy

import (
	"fmt"
	"strings"

	"github.com/boynton/api/model"
)

func Import(path string) (*model.Schema, error) {
	var ast *AST
	var err error
	if strings.HasSuffix(path, ".smithy") {
		ast, err = Parse(path)
	} else {
		ast, err = LoadAST(path)
	}
	if err != nil {
		return nil, err
	}
	//fmt.Println("smithy:", data.Pretty(ast))
	schema := model.NewSchema()
	err = ast.ForAllShapes(func(shapeId string, shape *Shape) error {
		return importShape(schema, ast, shapeId, shape)
	})
	return schema, err
}

func toCanonicalAbsoluteId(id string) model.AbsoluteIdentifier {
	lst := strings.Split(id, "#")
	if len(lst) == 2 {
		return model.AbsoluteIdentifier(strings.Join(lst, "#"))
	}
	fmt.Println("WARNING: non-absolute id:", id)
	//FIX: apply default namespace
	return model.AbsoluteIdentifier("fixme#" + id)
}

func toCanonicalTypeName(name string) model.AbsoluteIdentifier {
	switch name {
	case "boolean", "smithy.api#Boolean":
		return "base#Bool"
	case "byte", "smithy.api#Byte":
		return "base#Int8"
	case "short", "smithy.api#Short":
		return "base#Int16"
	case "integer", "smithy.api#Integer":
		return "base#Int32"
	case "long", "smithy.api#Long":
		return "base#Int64"
	case "float", "smithy.api#Float":
		return "base#Float32"
	case "double", "smithy.api#Double":
		return "base#Float64"
	case "bigInteger", "smithy.api#BigInteger":
		return "base#Integer"
	case "bigDecimal", "smithy.api#BigDecimal":
		return "base#Decimal"
	case "blob", "smithy.api#Blob":
		return "base#Bytes"
	case "string", "smithy.api#String":
		return "base#String"
	case "timestamp", "smithy.api#Timestamp":
		return "base#Timestamp"
	case "list", "smithy.api#List":
		return "base#List"
	case "map", "smithy.api#Map":
		return "base#Map"
	case "structure", "smithy.api#Structure":
		return "base#Struct"
	case "enum", "smithy.api#Enum":
		return "base#Enum"
	case "union", "smithy.api#Union":
		return "base#Union"
	default:
		return toCanonicalAbsoluteId(name)
	}
}

func addService(schema *model.Schema, ast *AST, shapeId string, shape *Shape) error {
	if schema.Id != "" {
		return fmt.Errorf("Cannot represent more than one service in model!")
	}
	schema.Id = model.AbsoluteIdentifier(shapeId)
	schema.Version = shape.Version
	schema.Comment = shape.Traits.GetString("smithy.api#documentation")
	return nil
}

func toOpInput(schema *model.Schema, ast *AST, shapeId string) *model.OperationInput {
	shape := ast.GetShape(shapeId)
	//shape.Traits.GetBool("smithy.api#input") should be true
	ti := &model.OperationInput{
		Id: model.AbsoluteIdentifier(shapeId),
		Comment: shape.Traits.GetString("smithy.api#documentation"),
	}	
	for _, k := range shape.Members.Keys() {
		mem := shape.Members.Get(k)
		f := &model.OperationInputField{
			Name: model.Identifier(k),
			Type: toCanonicalTypeName(mem.Target),
		}
		f.Required = mem.Traits.GetBool("smithy.api#required")
		q := mem.Traits.GetString("smithy.api#httpQuery")
		if q != "" {
			f.HttpQuery = model.Identifier(q)
		}
		h := mem.Traits.GetString("smithy.api#httpHeader")
		if h != "" {
			f.HttpHeader = model.Identifier(h)
		}
		f.HttpPath = mem.Traits.GetBool("smithy.api#httpLabel")
		f.HttpPayload = mem.Traits.GetBool("smithy.api#httpPayload")
		if f.HttpPath || f.HttpPayload {
			f.Required = true
		}
		ti.Fields = append(ti.Fields, f)
	}
	return ti
}

func toOpOutput(schema *model.Schema, ast *AST, shapeId string) *model.OperationOutput {
	shape := ast.GetShape(shapeId)
	//shape.Traits.GetBool("smithy.api#output") should be true
	to := &model.OperationOutput{
		Id: model.AbsoluteIdentifier(shapeId),
		Comment: shape.Traits.GetString("smithy.api#documentation"),
	}
	for _, k := range shape.Members.Keys() {
		mem := shape.Members.Get(k)
		f := &model.OperationOutputField{
			Name: model.Identifier(k),
			Type: toCanonicalTypeName(mem.Target),
		}
		h := mem.Traits.GetString("smithy.api#httpHeader")
		if h != "" {
			f.HttpHeader = model.Identifier(h)
		}
		f.HttpPayload = mem.Traits.GetBool("smithy.api#httpPayload")
		to.Fields = append(to.Fields, f)
	}
	return to
}

func addOperation(schema *model.Schema, ast *AST, shapeId string, shape *Shape) error {
	//validate: that namespace is the same as the service we use (only one per model)
	op := model.OperationDef{
		Id: model.AbsoluteIdentifier(shapeId),
		Comment: shape.Traits.GetString("smithy.api#documentation"),
	}
	typesConsumed := make(map[model.AbsoluteIdentifier]bool, 0)
	if shape.Input != nil {
		op.Input = toOpInput(schema, ast, shape.Input.Target)
		typesConsumed[op.Input.Id] = true
	}
	if shape.Output != nil {
		op.Output = toOpOutput(schema, ast, shape.Output.Target)
		typesConsumed[op.Output.Id] = true
	}
	if shape.Errors != nil {
		var excs []*model.OperationOutput
		for _, e := range shape.Errors {
			out := toOpOutput(schema, ast, e.Target)
			typesConsumed[out.Id] = true
			excs = append(excs, out)
		}
		op.Exceptions = excs
	}
	httpTrait := shape.Traits.Get("smithy.api#http")
	if httpTrait == nil {
		op.Output.HttpStatus = int32(httpTrait.GetInt("code", 0))
		if op.Output.HttpStatus == 0 {
			op.Output.HttpStatus = 200
		}
		op.HttpMethod = httpTrait.GetString("method")
		op.HttpUri = httpTrait.GetString("uri")
	}
	schema.Operations = append(schema.Operations, &op)
	return nil
}

func addResource(schema *model.Schema, ast *AST, shapeId string, shape *Shape) error {
	panic("smithy resources NYI")
	return nil
}

func importShape(schema *model.Schema, ast *AST, shapeId string, shape *Shape) error {
	td := &model.TypeDef{
		Id: toCanonicalAbsoluteId(shapeId),
		Comment: shape.Traits.GetString("smithy.api#documentation"),
	}
	number := false
	switch shape.Type {
	case "byte":
		td.Base = model.Int8
		number = true
	case "short":
		td.Base = model.Int16
		number = true
	case "integer":
		td.Base = model.Int32
		number = true
	case "long":
		td.Base = model.Int64
		number = true
	case "float":
		td.Base = model.Float32
		number = true
	case "double":
		td.Base = model.Float64
		number = true
	case "bigInteger":
		td.Base = model.Decimal
		//td.@integral = true?
		number = true
	case "bigDecimal":
		td.Base = model.Decimal
		number = true
	case "string":
		td.Base = model.String
		td.Pattern = shape.Traits.GetString("smithy.api#pattern")
	case "list":
		td.Base = model.List
		td.Items = toCanonicalTypeName(shape.Member.Target)
	case "map":
		td.Base = model.Map
		td.Keys = toCanonicalTypeName(shape.Key.Target)
		td.Items = toCanonicalTypeName(shape.Value.Target)
	case "union":
		td.Base = model.Union
		for _, name := range shape.Members.Keys() {
			fd := &model.FieldDef{
				Name: name,
			}
			v := shape.Members.Get(name)
			fd.Type = toCanonicalTypeName(v.Target)
			if v.Traits != nil {
				comment := v.Traits.GetString("smithy.api#documentation")
				if comment != "" {
					fd.Comment = comment
				}
			}
			td.Fields = append(td.Fields, fd)
		}
	case "structure":
		if shape.Traits.Get("smithy.api#input") != nil {
			//the operation using it handles this
			return nil
		} else if shape.Traits.Get("smithy.api#output") != nil {
			//the operation using it handles this
			return nil
		} else if shape.Traits.Get("smithy.api#error") != nil {
			//the operation using it handles this
			return nil
		} else {
			td.Base = model.Struct
			for _, name := range shape.Members.Keys() {
				fd := &model.FieldDef{
					Name: name,
				}
				v := shape.Members.Get(name)
				fd.Type = toCanonicalTypeName(v.Target)
				if v.Traits != nil {
					if v.Traits.Get("smithy.api#required") != nil {
						fd.Required = true
					}
					comment := v.Traits.GetString("smithy.api#documentation")
					if comment != "" {
						fd.Comment = comment
					}
				}
				td.Fields = append(td.Fields, fd)
			}
		}
	case "enum":
		td.Base = model.Enum
		for _, sym := range shape.Members.Keys() {
			el := &model.EnumElement{
				Symbol: model.Identifier(sym),
			}
			v := shape.Members.Get(sym)
			if v.Traits != nil {
				val := v.Traits.GetString("smithy.api#enumValue")
				if val != "" {
					el.Value = val
				}
				comment := v.Traits.GetString("smithy.api#documentation")
				if comment != "" {
					el.Comment = comment
				}
			} else {
				el.Value = sym
			}
			td.Elements = append(td.Elements, el)
		}
	case "operation":
		return addOperation(schema, ast, shapeId, shape)
	case "service":
		return addService(schema, ast, shapeId, shape)
	case "resource":
		return addResource(schema, ast, shapeId, shape)
	default:
		panic("implement me:" + shape.Type)
	}
	if number {
		rng := shape.Traits.Get("smithy.api#range")
		if rng != nil {
			td.MinValue = rng.Get("min").AsDecimal()
			td.MaxValue = rng.Get("max").AsDecimal()
		}
	}
	return schema.AddTypeDef(td)
}

func nameFromId(id string) model.Identifier {
	l := strings.Split(id, "#")
	if len(l) == 2 {
		return model.Identifier(l[1])
	}
	fmt.Println("WARNING: id is no absolute:", id)
	return model.Identifier(id)
}

