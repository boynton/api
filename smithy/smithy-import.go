package smithy

import (
	"fmt"
	"strings"

	"github.com/boynton/api/model"
)

func Warning(format string, a ...any) {
	model.Warning(format, a...)
}

func Import(paths []string, tags []string, parseOnly bool) (*model.Schema, error) {
	ast, err := Assemble(paths)
	if err != nil {
		return nil, err
	}
	if parseOnly {
		if len(tags) > 0 {
			ast.Filter(tags)
		}
		fmt.Println(model.Pretty(ast))
		return nil, nil
	}
	return ImportAST(ast, tags)
}

func isTagged(shape *Shape, tags []string) bool {
	if len(tags) == 0 {
		return true
	}
	shapeTags := shape.Traits.GetSlice("smithy.api#tags")
	for _, stag := range shapeTags {
		for _, tag := range tags {
			if stag == tag {
				return true
			}
		}
	}
	return false
}

func ImportAST(ast *AST, tags []string) (*model.Schema, error) {
	var err error
	schema := model.NewSchema()
	if len(tags) > 0 {
		ast.Filter(tags)
	} else {
		ns, err := ast.ServiceDependencies()
		if err != nil {
			return nil, err
		}
		schema.Namespace = model.Namespace(ns)
	}
	if ast.Metadata != nil {
		base := ast.Metadata.GetString("basePath")
		if base != "" {
			schema.Base = base
		}
	}
	err = ast.ForAllShapes(func(shapeId string, shape *Shape) error {
		return importShape(schema, ast, shapeId, shape)
	})
	if err != nil {
		return nil, err
	}
	err = ast.ForAllShapes(func(shapeId string, shape *Shape) error {
		if shape != nil {
			if shape.Type == "resource" {
				return addResource(schema, ast, shapeId, shape)
			}
		}
		return nil
	})
	return schema, err
}

func toCanonicalAbsoluteId(id string) model.AbsoluteIdentifier {
	lst := strings.Split(id, "#")
	if len(lst) == 2 {
		return model.AbsoluteIdentifier(strings.Join(lst, "#"))
	}
	model.Warning("Non-absolute id: %q\n", id)
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
	case "document", "smithy.api#Document":
		return "base#Any"
	default:
		return toCanonicalAbsoluteId(name)
	}
}

func addOperationFromRef(schema *model.Schema, ast *AST, ref *ShapeRef, rez string, lifecycle string) error {
	if ref != nil {
		shapeId := ref.Target
		shape := ast.GetShape(shapeId)
		return addOperation(schema, ast, shapeId, shape, rez, lifecycle)
	}
	return nil
}

func addResourceOperationsFromRef(schema *model.Schema, ast *AST, resRef *ShapeRef, rez string) error {
	shape := ast.GetShape(resRef.Target)
	return addResourceOperations(schema, ast, resRef.Target, shape, rez)
}

func addResourceOperations(schema *model.Schema, ast *AST, shapeId string, shape *Shape, resource string) error {
	rez := StripNamespace(shapeId)
	if resource != "" {
		rez = resource + "." + rez
	}
	var err error
	if shape == nil {
		return fmt.Errorf("Shape not found: %s", shapeId)
	}
	for _, ref := range shape.Operations {
		err = addOperationFromRef(schema, ast, ref, rez, "")
		if err != nil {
			return err
		}
	}
	for _, ref := range shape.Resources {
		err = addResourceOperationsFromRef(schema, ast, ref, rez)
		if err != nil {
			return err
		}
	}
	err = addOperationFromRef(schema, ast, shape.Create, rez, "create")
	if err != nil {
		return err
	}
	err = addOperationFromRef(schema, ast, shape.Put, rez, "put")
	if err != nil {
		return err
	}
	err = addOperationFromRef(schema, ast, shape.Read, rez, "read")
	if err != nil {
		return err
	}
	err = addOperationFromRef(schema, ast, shape.Update, rez, "update")
	if err != nil {
		return err
	}
	err = addOperationFromRef(schema, ast, shape.Delete, rez, "delete")
	if err != nil {
		return err
	}
	err = addOperationFromRef(schema, ast, shape.List, rez, "list")
	if err != nil {
		return err
	}
	for _, ref := range shape.CollectionOperations {
		err = addOperationFromRef(schema, ast, ref, rez, "collection")
		if err != nil {
			return err
		}
	}
	return nil
}

func addResource(schema *model.Schema, ast *AST, shapeId string, shape *Shape) error {
	return addResourceOperations(schema, ast, shapeId, shape, "")
}

func addService(schema *model.Schema, ast *AST, shapeId string, shape *Shape) error {
	if schema.Id != "" {
		return fmt.Errorf("Cannot represent more than one service in model!")
	}
	schema.Id = model.AbsoluteIdentifier(shapeId)
	schema.Version = shape.Version
	schema.Comment = shape.Traits.GetString("smithy.api#documentation")
	//TBD: other metadata
	for _, ref := range shape.Operations {
		err := addOperationFromRef(schema, ast, ref, "", "")
		if err != nil {
			return err
		}
	}
	for _, ref := range shape.Resources {
		err := addResourceOperationsFromRef(schema, ast, ref, "")
		if err != nil {
			return err
		}
	}
	return nil
}

func toOpInput(schema *model.Schema, ast *AST, shapeId string) *model.OperationInput {
	shape := ast.GetShape(shapeId)
	if shape == nil {
		panic("OpInput refers to undefined shape: " + shapeId)
	}
	//shape.Traits.GetBool("smithy.api#input") should be true
	ti := &model.OperationInput{
		Id:      model.AbsoluteIdentifier(shapeId),
		Comment: shape.Traits.GetString("smithy.api#documentation"),
	}
	var payloadContentFields []*model.FieldDef
	hasPayload := false
	for _, k := range shape.Members.Keys() {
		mem := shape.Members.Get(k)
		if mem == nil || mem.Target == "" {
			return nil
		}
		query := mem.Traits.GetString("smithy.api#httpQuery")
		header := mem.Traits.GetString("smithy.api#httpHeader")
		path := mem.Traits.GetBool("smithy.api#httpLabel")
		payload := mem.Traits.GetBool("smithy.api#httpPayload")
		if payload {
			hasPayload = true
		}
		if query == "" && header == "" && !path && !payload {
			structField := &model.FieldDef{
				Comment:  "",
				Name:     model.Identifier(k),
				Type:     toCanonicalTypeName(mem.Target),
				Required: mem.Traits.GetBool("smithy.api#required"),
			}
			payloadContentFields = append(payloadContentFields, structField)
		} else {
			f := &model.OperationInputField{
				Name:     model.Identifier(k),
				Type:     toCanonicalTypeName(mem.Target),
				Required: mem.Traits.GetBool("smithy.api#required"),
			}
			if query != "" {
				f.HttpQuery = model.Identifier(query)
			}
			if header != "" {
				f.HttpHeader = header
			}
			f.HttpPath = path
			f.HttpPayload = payload
			if f.HttpPath || f.HttpPayload {
				f.Required = true
			}
			d := mem.Traits.Get("smithy.api#default") //for synthesized payload content?!
			if d != nil {
				f.Default = d.RawValue()
			}
			if mem.Traits.Has("smithy.api#length") {
				length := mem.Traits.Get("smithy.api#length")
				f.MinSize = length.GetInt64("min", 0)
				f.MaxSize = length.GetInt64("max", 0)
			}
			//other traits!!!
			ti.Fields = append(ti.Fields, f)
		}
	}
	if len(payloadContentFields) > 0 {
		if hasPayload {
			model.Warning("Smithy operation input should have header/query/path/payload specified: %s\n", ti.Id)
		} else {
			model.Warning("Smithy operation input should have a payload specified: %s\n", ti.Id)
			contentType := model.AbsoluteIdentifier(shapeId + "Content")
			payloadField := &model.OperationInputField{
				Name:        model.Identifier("payload"),
				Type:        contentType,
				HttpPayload: true,
				Required:    true,
				Comment:     "(generated by smithy-import: use @httpPayload trait to avoid this)",
			}
			ti.Fields = append(ti.Fields, payloadField)
			payloadTd := &model.TypeDef{
				Id:      contentType,
				Base:    model.Struct,
				Comment: "synthesized content for operation input payload",
				Fields:  payloadContentFields,
			}
			err := schema.AddTypeDef(payloadTd)
			if err != nil {
				panic("Whoops!")
			}
		}
	}
	return ti
}

func toOpOutput(schema *model.Schema, ast *AST, shapeId string) *model.OperationOutput {
	shape := ast.GetShape(shapeId)
	if shape == nil {
		panic("OpOutput refers to undefined shape: " + shapeId)
	}
	//shape.Traits.GetBool("smithy.api#output") should be true
	to := &model.OperationOutput{
		Id: model.AbsoluteIdentifier(shapeId),
	}
	if shape.Traits != nil {
		to.Comment = shape.Traits.GetString("smithy.api#documentation")
	}
	var payloadContentFields []*model.FieldDef
	hasPayload := false
	for _, k := range shape.Members.Keys() {
		mem := shape.Members.Get(k)
		header := mem.Traits.GetString("smithy.api#httpHeader")
		payload := mem.Traits.GetBool("smithy.api#httpPayload")
		if payload {
			hasPayload = true
		}
		if header == "" && !payload {
			structField := &model.FieldDef{
				Comment:  "",
				Name:     model.Identifier(k),
				Type:     toCanonicalTypeName(mem.Target),
				Required: mem.Traits.GetBool("smithy.api#required"),
			}
			payloadContentFields = append(payloadContentFields, structField)
		} else {
			f := &model.OperationOutputField{
				Name: model.Identifier(k),
				Type: toCanonicalTypeName(mem.Target),
			}
			f.HttpHeader = header
			f.HttpPayload = payload
			to.Fields = append(to.Fields, f)
		}
	}
	if len(payloadContentFields) > 0 {
		if hasPayload {
			model.Warning("Smithy operation output should have header/payload specified: %s\n", to.Id)
		} else {
			//A warning is emitted in the parser, this code is to work around the problem
			contentType := model.AbsoluteIdentifier(shapeId + "Content")
			payloadField := &model.OperationOutputField{
				Name:        model.Identifier("payload"),
				Type:        contentType,
				HttpPayload: true,
				Required:    true,
				Comment:     "(generated by smithy-import: use @httpPayload trait to avoid this)",
			}
			to.Fields = append(to.Fields, payloadField)
			if schema.GetTypeDef(contentType) == nil {
				payloadTd := &model.TypeDef{
					Id:      contentType,
					Base:    model.Struct,
					Comment: "synthesized content for operation input payload",
					Fields:  payloadContentFields,
				}
				err := schema.AddTypeDef(payloadTd)
				if err != nil {
					model.Error("%s\n", err)
				}
			} else {
				//to do: verify it is identical
			}
		}
	}
	to.HttpStatus = int32(shape.Traits.GetInt("smithy.api#httpError", 0))
	return to
}

func operationAlreadyAdded(schema *model.Schema, shapeId string) bool {
	for _, op := range schema.Operations {
		if string(op.Id) == shapeId {
			return true
		}
	}
	return false
}

func addOperation(schema *model.Schema, ast *AST, shapeId string, shape *Shape, resource string, lifecycle string) error {
	//validate: that namespace is the same as the service we use (only one per model)
	if shape == nil {
		return fmt.Errorf("Operation shape not found: %s", shapeId)
	}
	id := model.AbsoluteIdentifier(shapeId)
	if operationAlreadyAdded(schema, shapeId) {
		prev := schema.GetOperationDef(id)
		if prev != nil {
			prev.Resource = resource
			prev.Lifecycle = lifecycle
		}
		return nil
	}
	//use tags for the resource and lifecycle, i.e. tags: [resource=ItemResource, lifecycle:Read]
	op := model.OperationDef{
		Id:        id,
		Comment:   shape.GetStringTrait("smithy.api#documentation"),
		Resource:  resource,
		Lifecycle: lifecycle,
	}
	typesConsumed := make(map[model.AbsoluteIdentifier]bool, 0)
	if shape.Input != nil && shape.Input.Target != "smithy.api#Unit" {
		op.Input = toOpInput(schema, ast, shape.Input.Target)
		if op.Input != nil {
			typesConsumed[op.Input.Id] = true
		}
	}
	if shape.Output != nil && shape.Output.Target != "smithy.api#Unit" {
		op.Output = toOpOutput(schema, ast, shape.Output.Target)
		typesConsumed[op.Output.Id] = true
	} else {
		op.Output = &model.OperationOutput{}
	}
	if shape.Errors != nil {
		var eids []model.AbsoluteIdentifier
		for _, e := range shape.Errors {
			eids = append(eids, model.AbsoluteIdentifier(e.Target))
		}
		op.Exceptions = eids
	}
	httpTrait := shape.Traits.Get("smithy.api#http")
	if httpTrait != nil {
		op.Output.HttpStatus = int32(httpTrait.GetInt("code", 0))
		if op.Output.HttpStatus == 0 {
			op.Output.HttpStatus = 200
		}
		op.HttpMethod = httpTrait.GetString("method")
		op.HttpUri = httpTrait.GetString("uri")
		if op.HttpMethod == "POST" || op.HttpMethod == "PUT" {
			if op.Input == nil {
				return fmt.Errorf("Smithy operation with Http Method %s requires a payload", op.HttpMethod)
			}
		}
		hasPayload := false
		for _, field := range op.Output.Fields {
			if field.HttpPayload {
				hasPayload = true
			}
		}
		if op.Output.HttpStatus == 204 { //note: Smithy cannot do a 304, but would have same constraint
			if hasPayload {
				return fmt.Errorf("Smithy operation output for a 204 response must have no payload: %s", op.Id)
			}
		} else if !hasPayload {
			return fmt.Errorf("Smithy operation output for a non-204 response must have a payload specified: %s", op.Id)
		}
	}
	schema.Operations = append(schema.Operations, &op)
	return nil
}

func importShape(schema *model.Schema, ast *AST, shapeId string, shape *Shape) error {
	if shape == nil {
		return nil
	}
	td := &model.TypeDef{
		Id:      toCanonicalAbsoluteId(shapeId),
		Comment: shape.GetStringTrait("smithy.api#documentation"),
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
		td.Base = model.Integer
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
				Name: model.Identifier(name),
			}
			v := shape.Members.Get(name)
			fd.Type = toCanonicalTypeName(v.Target)
			if v.Traits != nil {
				comment := v.GetStringTrait("smithy.api#documentation")
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
			out := toOpOutput(schema, ast, shapeId)
			out.Comment = shape.GetStringTrait("smithy.api#documentation")
			return schema.EnsureExceptionDef(out)
		} else {
			td.Base = model.Struct
			for _, name := range shape.Members.Keys() {
				fd := &model.FieldDef{
					Name: model.Identifier(name),
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
				//BUG: arbitrary traits on the field are not preserved. Notably: base#Int32 cannot have a smithy.api#range
				// trait, the MinValue/MaxValue properties require that a new type be defined: type Foo Int32 (MinValue...)
				// Yet, Smithy does not allow defining inline arrays or maps or other types. Just traits on the declared type
				// That is: traits can be on the field, in addition to on the type. The field traits override the type traits
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
	case "timestamp":
		td.Base = model.Timestamp
	case "service":
		return addService(schema, ast, shapeId, shape)
	case "operation":
		return addOperation(schema, ast, shapeId, shape, "", "")
	case "resource":
		return addResource(schema, ast, shapeId, shape)
	case "apply":
		/*
			//apply to another shape. Do I handle forward references? The model keeps separate. Hmm.
			shapeMember := strings.Split(shapeId, "$")
			if len(shapeMember) != 2 {
			panic("apply id has no member component")
			} else {
			targetId := model.AbsoluteIdentifier(shapeMember[0])
			targetTd := schema.GetTypeDef(targetId)
			fmt.Printf("targetId: %q, targetTd: %v\n", targetId, targetTd)
			if targetTd == nil {
			fmt.Print("Cannot find target shape for apply: " + shapeMember[0])
			panic("whoa")
			} else {
			fmt.Println("apply to", shapeMember, ", targetTd:", targetTd, ", these traits:", model.Pretty(shape))
			panic("here")
			}
			}
			//		panic("implement 'apply': '" + shapeId + "' -> " + model.Pretty(shape))
		*/
		return nil
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
	model.Warning("Id is not absolute:", id)
	return model.Identifier(id)
}
