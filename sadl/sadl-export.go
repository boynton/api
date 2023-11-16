/*
Copyright 2021 Lee R. Boynton

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package sadl

import (
	//	"bufio"
	//	"bytes"
	"fmt"
	"strings"

	"github.com/boynton/api/common"
	"github.com/boynton/api/model"
	"github.com/boynton/data"
)

const IndentAmount = "    "

func Uncapitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[0:1]) + s[1:]
}

type Generator struct {
	common.BaseGenerator
	ns   string
	name string
}

func (gen *Generator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.ns = string(schema.ServiceNamespace())
	gen.name = string(schema.ServiceName())
	fname := gen.FileName(gen.name, ".sadl")
	//	err = gen.Validate()
	//	if err != nil {
	//		return err
	//	}
	s := gen.ToSadl()
	return gen.Write(s, fname, "")
}

/*
func (gen *Generator) Validate() error {
	schema := gen.Schema
	ns := config.GetString("namespace")
	for _, nsk := range schema.Shapes.Keys() {
		shape := schema.GetShape(nsk)
		if shape == nil {
			return fmt.Errorf("Undefined shape: %s\n", nsk)
		}
		lst := strings.Split(nsk, "#")
		k := lst[1]
		if shape.Type == "operation" {
			err := gen.validateOperation(lst[0], k, shape)
			if err != nil {
				return err
			}
		} else {
			err := gen.validateType(lst[0], k, shape
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (gen *Generator) validateType(ns, n string, shape *smithy.Shape) error {
	switch shape.Type {
	case "intEnum":
		return fmt.Errorf("intEnum not supported by SADL: %s#%s", ns, n)
	}
	return nil
}

func (gen *Generator) validateOperation(ns, n string, shape *smithy.Shape) error {
	fullName := ns + "#" + n
	httpTrait := shape.Traits.GetDocument("smithy.api#http")
	if httpTrait == nil {
		return fmt.Errorf("Operation without @http trait not valid for SADL: %s", fullName)
	}
	method := httpTrait.GetString("method")
	expectInputPayload := method == "PUT" || method == "POST" || method == "PATCH"
	inputPayload := false
	if shape.Input != nil {
		inShape := ast.GetShape(shape.Input.Target)
		if inShape == nil {
			return fmt.Errorf("Undefined shape: %s\n", shape.Input.Target)
		}
		for _, k := range inShape.Members.Keys() {
			var isPayload, isHeader, isQuery, isLabel bool
			v := inShape.Members.Get(k)
			if v.Traits != nil {
				if v.Traits.Has("smithy.api#httpPayload") {
					if inputPayload {
						return fmt.Errorf("More than one @httpPayload specified in the input for operation %s", fullName)
					}
					inputPayload = true
					isPayload = true
				} else if v.Traits.Has("smithy.api#httpHeader") {
					//check header value
					isHeader = true
				} else if v.Traits.Has("smithy.api#httpLabel") {
					//check that label is present in path template
					isLabel = true
				} else if v.Traits.Has("smithy.api#httpQuery") {
					isQuery = true
				}
				if !isPayload && !isHeader && !isQuery && !isLabel {
					return fmt.Errorf("An input with no HTTP binding is present in operation %s: %s", fullName, k)
				}
			} else {
				return fmt.Errorf("An input with no HTTP binding is present in operation %s: %s", fullName, k)
			}
		}
	}
	if inputPayload != expectInputPayload {
		if inputPayload {
			return fmt.Errorf("HTTP operation '%s' with method %s expects no input payload, but one was specified", fullName, method)
		} else {
			return fmt.Errorf("HTTP operation '%s' with method %s expects an input payload, but none is specified", fullName, method)
		}
	}
	status := httpTrait.GetInt("code")
	expectOutputPayload := status != 204 && status != 304
	outputPayload := false
	if shape.Output != nil {
		outShape := ast.GetShape(shape.Output.Target)
		if outShape == nil {
			return fmt.Errorf("Undefined shape: %s\n", shape.Output.Target)
		}
		for _, k := range outShape.Members.Keys() {
			v := outShape.Members.Get(k)
			if v.Traits != nil {
				if v.Traits.Has("smithy.api#httpPayload") {
					if outputPayload {
						return fmt.Errorf("More than one @httpPayload specified in output for operation %s", fullName)
					}
					outputPayload = true
				} else if v.Traits.Has("smithy.api#httpResponseCode") {
					//
				} else if !v.Traits.Has("smithy.api#httpHeader") {
					return fmt.Errorf("An output with no HTTP binding is present in operation %s: %s", fullName, k)
				}
			} else {
				return fmt.Errorf("An output with no HTTP binding is present in operation %s: %s", fullName, k)
			}
		}
	}
	if outputPayload != expectOutputPayload {
		if outputPayload {
			return fmt.Errorf("HTTP operation '%s' with code %d expects no output payload, but one was specified", fullName, status)
		} else {
			return fmt.Errorf("HTTP operation '%s' with code %d expects an output payload, but none is specified", fullName, status)
		}
	}
	return nil
}
*/

func (gen *Generator) ToSadl() string {
	gen.Begin()
	emitted := make(map[string]bool, 0)

	//gen.Emit("/* Generated by `api` tool (https://github.com/boynton/api) */\n\n")
	if gen.ns != "" {
		gen.Emitf("namespace %s\n", gen.ns)
	}
	//	if ast.RequiresDocumentType() {
	//		gen.Emit("\ntype Document Struct //SADL has no built-in Document type\n")
	//	}

	if len(gen.Schema.Operations) > 0 {
		for _, op := range gen.Schema.Operations {
			opts := gen.opAnnotations(op)
			gen.EmitOperation(op, opts)
			emitted[op.Name()] = true
			if op.Input != nil {
				if td := gen.Schema.GetTypeDef(op.Input.Id); td != nil {
					emitted[td.Name()] = true
				}
			}
			if op.Output != nil && op.Output.Id != "" {
				if td := gen.Schema.GetTypeDef(op.Output.Id); td != nil {
					emitted[td.Name()] = true
				}
			}
			if len(op.Exceptions) > 0 {
				for _, e := range op.Exceptions {
					if _, ok := emitted[e.Name()]; !ok {
						gen.EmitException(e)
						emitted[e.Name()] = true
					}
				}
			}
		}
	}
	for _, td := range gen.Schema.Types {
		gen.EmitType(td)
	}
	return gen.End()
}

func (gen *Generator) opAnnotations(op *model.OperationDef) []string {
	return nil
}

func (gen *Generator) EmitType(td *model.TypeDef) {
	switch td.Base {
	case model.Bool:
		gen.EmitBooleanType(td)
	case model.Int8, model.Int16, model.Int32, model.Int64, model.Float32, model.Float64, model.Integer, model.Decimal:
		gen.EmitNumericType(td)
	case model.Blob:
		gen.EmitBlobType(td)
	case model.String:
		gen.EmitStringType(td)
	case model.Timestamp:
		gen.EmitTimestampType(td)
	case model.List:
		gen.EmitListType(td)
	case model.Map:
		gen.EmitMapType(td)
	case model.Struct:
		gen.EmitStructType(td)
	case model.Union:
		gen.EmitUnionType(td)
	case model.Enum:
		gen.EmitEnumType(td)
		//	case "document":
		//		gen.EmitDocumentShape(name, shape, opts)
		//	case "resource":
		//no equivalent in SADL at the moment
	default:
		panic("fix: type " + td.Name() + " with base " + data.Pretty(td))
	}
}

func (gen *Generator) EmitComment(comment string) {
	gen.Emit("\n")
	if comment != "" {
		gen.Emit(common.FormatComment("", "// ", comment, 100, true))
	}
}

func (gen *Generator) EmitEnumType(td *model.TypeDef) {
	gen.EmitComment(td.Comment)
	gen.Emitf("type %s Enum {\n", td.Name())
	for _, el := range td.Elements {
		gen.Emitf("%s%s\n", IndentAmount, el.Symbol) //FIX: el.Value
	}
	gen.Emit("}\n")
}

func (gen *Generator) EmitBooleanType(td *model.TypeDef) {
	opt := ""
	gen.EmitComment(td.Comment)
	gen.Emit("type " + td.Name() + " Boolean" + opt + "\n")
}

func (gen *Generator) EmitNumericType(td *model.TypeDef) {
	var opts []string
	if td.MinValue != nil {
		opts = append(opts, fmt.Sprintf("min=%v", td.MinValue))
	}
	if td.MaxValue != nil {
		opts = append(opts, fmt.Sprintf("max=%v", td.MaxValue))
	}
	sopts := gen.annotationString(opts)
	gen.EmitComment(td.Comment)
	gen.Emitf("type %s %s%s\n", td.Name(), td.Base.String(), sopts)
}

func (gen *Generator) EmitStringType(td *model.TypeDef) {
	var opts []string
	if td.Pattern != "" {
		opts = append(opts, fmt.Sprintf("pattern=%q", td.Pattern))
	}
	sopts := gen.annotationString(opts)
	gen.EmitComment(td.Comment)
	gen.Emitf("type %s String%s\n", td.Name(), sopts)
}

func (gen *Generator) EmitTimestampType(td *model.TypeDef) {
	gen.EmitComment(td.Comment)
	gen.Emitf("type %s Timestamp\n", td.Name())
}

func (gen *Generator) EmitBlobType(td *model.TypeDef) {
	opts := "" //fixme
	gen.EmitComment(td.Comment)
	gen.Emitf("type %s Blob%s\n", td.Name(), opts)
}

func (gen *Generator) EmitListType(td *model.TypeDef) {
	var opts []string
	if td.MinSize != nil {
		opts = append(opts, fmt.Sprintf("minsize=%v", *td.MinSize))
	}
	if td.MaxSize != nil {
		opts = append(opts, fmt.Sprintf("maxsize=%v", *td.MaxSize))
	}
	sopts := gen.annotationString(opts)
	gen.EmitComment(td.Comment)
	gen.Emitf("type %s List<%s>%s\n", td.Name(), model.StripNamespace(td.Items), sopts)
}

func (gen *Generator) EmitMapType(td *model.TypeDef) {
	gen.EmitComment(td.Comment)
	gen.Emitf("type %s Map<%s,%s>\n", td.Name(), model.StripNamespace(td.Keys), model.StripNamespace(td.Items))
}

/*
   func (gen *Generator) EmitDocumentShape(name string, shape *smithy.Shape, opts []string) {
	sopts := gen.annotationString(opts)
	gen.EmitShapeComment(shape)
	gen.Emitf("type %s Struct%s\n", name, sopts)
}
*/

func (gen *Generator) EmitStructType(td *model.TypeDef) {
	//	sopts := gen.annotationString(opts)
	sopts := ""
	gen.EmitComment(td.Comment)
	gen.Emitf("type %s Struct%s {\n", td.Name(), sopts)
	for _, f := range td.Fields {
		tref := gen.stripNamespace(gen.sadlTypeRef(f.Type))
		sopts := "" //gen.traitsAsAnnotationString(v.Traits)
		gen.Emitf("%s%s %s%s\n", IndentAmount, f.Name, tref, sopts)
	}
	gen.Emit("}\n")
}

func (gen *Generator) EmitUnionType(td *model.TypeDef) {
	opt := ""
	gen.EmitComment(td.Comment)
	gen.Emit("type " + td.Name() + " Union" + opt + " {\n")
	for _, f := range td.Fields {
		tref := gen.stripNamespace(gen.sadlTypeRef(f.Type))
		sopts := "" //gen.traitsAsAnnotationString(v.Traits)
		gen.Emitf("%s%s %s%s\n", IndentAmount, f.Name, tref, sopts)
	}
	gen.Emit("}\n")
}

func (gen *Generator) EmitOperation(op *model.OperationDef, opts []string) {
	gen.EmitComment(op.Comment)
	method := op.HttpMethod
	path := op.HttpUri
	expected := op.Output.HttpStatus
	/*
		var inType string
		if op.Input != nil {
			inType = gen.sadlTypeRef(op.Input.shape.Input.Target)
		}
		var outType string
		if shape.Output != nil {
			outType = gen.shapeRefToTypeRef(shape.Output.Target)
		}
	*/
	opts = append(opts, fmt.Sprintf("operation=%s", Uncapitalize(op.Name())))
	sopts := "(" + strings.Join(opts, ", ") + ")"
	queryParams := gen.queryParams(op.Input)
	gen.Emitf("http %s %q %s {\n", method, path+queryParams, sopts)
	if op.Input != nil {
		//if inputIsPayload {
		//			k := "body"
		//			tref := w.stripNamespace(inType)
		//			gen.Emitf("\t%s %s (required)\n", k, tref)
		//} else {
		gen.EmitOperationInputFields(op.Input.Fields)
		//}
		gen.Emit("\n")
	}

	if len(op.Output.Fields) == 0 {
		gen.Emitf("    expect %d\n", expected) //no content
	} else {
		gen.Emitf("    expect %d {\n", expected)
		gen.EmitOperationOutputFields(op.Output.Fields, "    ")
		gen.Emit("    }\n")
	}
	//except: we have to iterate through the "errors" of the operation, and check each one for httpError
	//Note that there is in that case not much opportunity to do headers.
	if len(op.Exceptions) > 0 {
		for _, e := range op.Exceptions {
			errCode := e.HttpStatus
			if errCode != 0 {
				gen.Emitf("    except %d %s\n", errCode, e.Name())
			}
		}
	}
	gen.Emit("}\n")
}

func (gen *Generator) EmitException(e *model.OperationOutput) {
	gen.EmitComment(e.Comment)
	gen.Emitf("type %s Struct {\n", e.Name())
	gen.EmitOperationOutputFields(e.Fields, "")
	gen.Emit("}\n")
}

func (gen *Generator) EmitOperationInputFields(fields []*model.OperationInputField) {
	for _, f := range fields {
		var mopts []string
		if f.HttpPayload {
			mopts = append(mopts, "required")
		} else {
			s := f.HttpQuery
			if s != "" {
				//default?
			} else {
				if f.HttpPath {
					mopts = append(mopts, "required")
				} else {
					s := f.HttpHeader
					if s != "" {
						mopts = append(mopts, fmt.Sprintf("header=%q", s))
					}
				}
			}
		}
		sopts := ""
		if len(mopts) > 0 {
			sopts = " (" + strings.Join(mopts, ",") + ")"
		}
		tref := gen.stripNamespace(gen.sadlTypeRef(f.Type))
		gen.Emitf("    %s %s%s\n", f.Name, tref, sopts)
	}
}

func (gen *Generator) queryParams(in *model.OperationInput) string {
	queryParams := ""
	//inputIsPayload := false //method == "PUT" || method == "POST" || method == "PATCH"
	if in != nil {
		for _, f := range in.Fields {
			//if f.HttpPayload { //bogus, should *always* be true
			//	inputIsPayload = false
			//	break
			//}
			s := string(f.HttpQuery)
			if s != "" {
				p := s + "={" + string(f.Name) + "}"
				if queryParams == "" {
					queryParams = "?" + p
				} else {
					queryParams = queryParams + "&" + p
				}
			}
		}
	}
	return queryParams
}

func (gen *Generator) EmitOperationOutputFields(fields []*model.OperationOutputField, indent string) {
	for _, f := range fields {
		var mopts []string
		if f.HttpPayload {
		} else {
			s := f.HttpHeader
			if s != "" {
				mopts = append(mopts, fmt.Sprintf("header=%q", s))
			}
		}
		sopts := ""
		if len(mopts) > 0 {
			sopts = " (" + strings.Join(mopts, ", ") + ")"
		}
		tref := gen.stripNamespace(gen.sadlTypeRef(f.Type))
		gen.Emitf(indent+"    %s %s%s\n", f.Name, tref, sopts)
	}
}

/*
   func (w *SadlWriter) EmitExample(shape *smithy.Shape, obj *data.Document) {
	opName := obj.GetString("title")
	if obj.Has("input") {
		reqType := gen.stripNamespace(shape.Input.Target)
		gen.Emitf("\nexample %s (name=%s) ", reqType, opName)
		gen.Emit(data.Pretty(obj.GetDocument("input")))
	}
	if obj.Has("error") {
		er := obj.GetDocument("error")
		respType := gen.stripNamespace(er.GetString("shapeId"))
		gen.Emitf("\nexample %s (name=%s) ", respType, opName)
		gen.Emit(data.Pretty(er.GetDocument("error")))
	} else {
		respType := gen.stripNamespace(shape.Output.Target)
		gen.Emitf("\nexample %s (name=%s) ", respType, opName)
		gen.Emit(data.Pretty(obj.GetDocument("output")))
	}
}
*/

func (gen *Generator) stripNamespace(id string) string {
	//fixme: just totally ignore it for now
	n := strings.Index(id, "#")
	if n < 0 {
		return id
	} else {
		return id[n+1:]
	}
}

//func (w *SadlWriter) formatBlockComment(indent string, comment string) {
//}

func (gen *Generator) sadlTypeRef(typeRef model.AbsoluteIdentifier) string {
	switch typeRef {
	case "base#Blob", "Blob":
		return "Bytes"
	case "base#Bool", "Boolean":
		return "Bool"
	case "base#String", "String":
		return "String"
	case "base#Byte", "Byte":
		return "Int8"
	case "base#Short", "Short":
		return "Int16"
	case "base#Integer", "Integer":
		return "Int32"
	case "base#Long", "Long":
		return "Int64"
	case "base#Float", "Float":
		return "Float32"
	case "base#Double", "Double":
		return "Float64"
	case "base#BigInteger", "BigInteger":
		return "Decimal" //lossy!
	case "base#BigDecimal", "BigDecimal":
		return "Decimal"
	case "base#Timestamp", "Timestamp":
		return "Timestamp"
		//	case "base#Document", "Document":
		//		return "Document" //to do: a new primitive type for this. For now, a naked Struct works
	default:
		//		ltype := model.ensureLocalNamespace(typeRef)
		//		if ltype == "" {
		//			panic("external namespace type refr not supported: " + typeRef)
		//		}
		//implement "use" correctly to handle this.
		//typeRef = ltype
	}
	return string(typeRef)
}

func withAnnotation(annos map[string]string, key string, value string) map[string]string {
	if value != "" {
		if annos == nil {
			annos = make(map[string]string, 0)
		}
		annos[key] = value
	}
	return annos
}

func (gen *Generator) annotationString(opts []string) string {
	if len(opts) == 0 {
		return ""
	}
	return fmt.Sprintf(" (%s)", strings.Join(opts, ", "))
}

///func (gen *Generator) traitsAsAnnotationString(traits *data.Document) string {
//	return gen.annotationString(gen.traitsAsAnnotations(traits))
//}

/*
   func (gen *Generator) typeAnnotations(td *model.TypeDef) []string {
	var opts []string
	if traits != nil {
		for _, k := range traits.Keys() {
			v := traits.Get(k)
			switch k {
			case "smithy.api#required":
				opts = append(opts, "required")
			case "smithy.api#deprecated":
				if gen.Config.GetBool("annotate") {
					//				dv := data.AsMap(v)
					dv := data.AsDocument(v)
					msg := dv.GetString("message")
					opts = append(opts, fmt.Sprintf("x_deprecated=%q", msg))
				}
			case "smithy.api#timestampFormat":
				if gen.Config.GetBool("annotate") {
					opts = append(opts, fmt.Sprintf("x_timestampFormat=%q", v))
				}
			case "smithy.api#tags":
				if gen.Config.GetBool("annotate") {
					opts = append(opts, fmt.Sprintf("x_tags=%q", strings.Join(data.AsStringSlice(v), ",")))
				}
			case "smithy.api#error":
				if gen.Config.GetBool("annotate") {
					opts = append(opts, "x_error")
				}
			case "smithy.api#httpError":
				if gen.Config.GetBool("annotate") {
					opts = append(opts, fmt.Sprintf("x_httpError=\"%v\"", v))
				}
			}
		}
	}
	return opts
}
*/
