/*
Copyright 2023 Lee R. Boynton

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
package golang

import (
	"bufio"
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/boynton/data"
	"github.com/boynton/api/common"
	"github.com/boynton/api/model"
)

const IndentAmount = "    "

type Generator struct {
	common.BaseGenerator
	ns model.Namespace
	inlineSlicesAndMaps bool //List/Map TypeDefs are inlined to be slices. This prevents validating constraints (i.e. list.maxLength)
	inlinePrimitives bool //primitive TypeDefs are inlined as the native types. This prevents validating constraints (i.e. string.Pattern)
	decimalPackage string //use this package for the Decimal implementation. If "", then generate one in this package
	decimalPrefix string
	timestampPackage string //use this package for the Timestamp implementation. If "", then generate one in this package
	timestampPrefix string
}

func (gen *Generator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.inlineSlicesAndMaps = config.GetBool("golang.inlineSlicesAndMaps")
	gen.inlinePrimitives = config.GetBool("golang.inlinePrimitives")
	gen.decimalPackage = config.GetString("golang.decimalPackage")
	if gen.decimalPackage != "" {
		gen.decimalPrefix = path.Base(gen.decimalPackage) + "."
		gen.decimalPackage = gen.decimalPackage
	}
	gen.timestampPackage = config.GetString("golang.timestampPackage")
	if gen.timestampPackage != "" {
		gen.timestampPrefix = path.Base(gen.timestampPackage) + "."
		gen.timestampPackage = gen.timestampPackage
	}
	gen.ns = model.Namespace(config.GetString("namespace"))
	if gen.ns == "" {
		gen.ns = schema.ServiceNamespace()
		if gen.ns == "" {
			gen.ns = "main"
		}
	}
	fbase := string(gen.ns)
	if fbase == "" {
		fbase = "model"
	}
	err = gen.Validate() //for golang generation
	if err != nil {
		return err
	}

	s := gen.GenerateOperations()
	fname := gen.FileName(fbase + "_operations", ".go")	
	err = gen.Emit(s, fname, "\n\n------------------" + fname + "\n")
	if err != nil {
		return err
	}
	fname = gen.FileName(fbase + "_types", ".go")
	s = gen.GenerateTypes()
	err = gen.Emit(s, fname, "\n\n------------------" + fname + "\n")
	if err != nil {
		return err
	}
	fname = gen.FileName(fbase + "_server", ".go")
	s = gen.GenerateServer()
	err = gen.Emit(s, fname, "\n\n------------------" + fname + "\n")
	if err != nil {
		return err
	}
	/*
	fname = gen.FileName(fbase + "_client", ".go")
	s = gen.GenerateClient(ns, model)
	err = gen.Emit(s, fname, "\n\n------------------" + fname + "\n")
	if err != nil {
		return err
	}
	*/
	return nil
}

func (gen *Generator) Validate() error {
	//this is to validate the model for Go code gen. For example, only http APIs currently are implemented (not RPC)
	return nil
}

type GolangWriter struct {
	buf       bytes.Buffer
	writer    *bufio.Writer
	//	namespace model.Namespace
	//	name      model.Identifier
	//	config    *data.Struct
	//	model     *model.Model
	gen       *Generator
}

func (gen *Generator) golangTypeDef(td *model.TypeDef) string {
	gname := golangTypeName(td.Id)
	comment := ""
	if td.Comment != "" {
		comment = common.FormatComment("", "// ", td.Comment, 80, true)
	}
	switch td.Base {
	default:
		return comment + "type " + gname + " " + golangBaseTypeName(td.Base) + "\n"
    case model.Decimal:
		//if @integral {
		//    return comment + "type " + gname + " big.Int\n"
		//}
		return comment + "type " + gname + "*" + gen.decimalPrefix + "Decimal\n"
    case model.Blob:
		return comment + "type " + gname + " []byte\n"
	case model.String:
		if gen.inlinePrimitives {
			return comment + "type " + gname + " string\n"
		}
    case model.Timestamp:
		return comment + "type " + gname + "*" + gen.timestampPrefix + "Timestamp\n"
    case model.List:
		return comment + "type " + gname + " []" + golangTypeRef(td.Items, false) + "\n"
    case model.Map:
		//		return "map[" + golangTypeRef(td.Keys) + "]" + golangTypeRef(td.Items)
		return comment + "type " + gname + " map[" + golangTypeRef(td.Keys, false) + "]" + golangTypeRef(td.Items, false) + "\n"
    //case model.BaseTypeStruct:
    //case model.BaseTypeEnum:
	//case model.BaseTypeUnion:
		//case model.BaseTypeNull:
	}
	return "//? " + string(td.Id)
}

func golangBaseTypeName(bt model.BaseType) string {
	switch bt {
    case model.Bool:
		return "bool"
    case model.Int8:
		return "int8"
    case model.Int16:
		return "int16"
	case model.Int32:
		return "int" //!
    case model.Int64:
		return "int64"
    case model.Float32:
		return "float32"
    case model.Float64:
		return "float64"
    case model.Decimal:
		//if @integral {
		//    return "big.Int"
		//}
		return "data.Decimal"
    case model.Blob:
		return "[]byte"
	case model.String:
		return "string"
    case model.Timestamp:
		return "data.Timestamp"
    //case model.List:
		//		return "[]" + golangTypeRef(td.Items)
    //case model.Map:
		//		return "map[" + golangTypeRef(td.Keys) + "]" + golangTypeRef(td.Items)
    //case model.Struct:
    //case model.Enum:
	//case model.Union:
	//case model.Null:
	default:
		fmt.Println("bt:", bt)
		panic("not concrete")
	}
}

//for now, assume a single package, so we strip the namespace of non-language types
func golangTypeRef(typeRef model.AbsoluteIdentifier, required bool) string {
	optional := ""
	//	if !required {
	//		optional = "*"
	//	}
	switch typeRef {
	case "base#Bytes":
		return "[]byte"
	case "base#Bool":
		return optional + "bool"
	case "base#String":
		return optional + "string"
	case "base#Int8":
		return optional + "int8"
	case "base#Int16":
		return optional + "int16"
	case "base#Int32":
		return optional + "int32"
	case "base#Int64":
		return optional + "int64"
	case "base#Float32":
		return optional + "float32"
	case "base#Float64":
		return optional + "float64"
	case "base#Decimal":
		return "*data.Decimal"
	case "base#Timestamp":
		return "*data.Timestamp"
	default:
		//BUG: if the type is a primitive type like int or string, we do not correctly handle the "optional" thing here.
		//the only way would be to look up the type
		return "*" + stripNamespace(typeRef)
	}
}

func (gen *Generator) baseTypeRef(typeRef model.AbsoluteIdentifier) string {
    switch typeRef {
    case "base#Bytes":
        return "[]byte"
    case "base#Bool":
        return "bool"
    case "base#String":
        return "string"
    case "base#Int8":
        return "int8"
    case "base#Int16":
        return "int16"
    case "base#Int32":
        return "int32"
    case "base#Int64":
        return "int64"
    case "base#Float32":
        return "float32"
    case "base#Float64":
        return "float64"
    case "base#Decimal":
        return "*" + gen.decimalPrefix + "Decimal"
    case "base#Timestamp":
        return "*" + gen.timestampPrefix + "Timestamp"
	default:
		panic("not a base type: " + typeRef)
	}
}

func (gen *Generator) golangTypeRef(typeRef model.AbsoluteIdentifier) string {
	td := gen.Schema.GetTypeDef(typeRef)
	if td == nil || gen.inlinePrimitives {
		return gen.baseTypeRef(typeRef)
	}
	switch td.Base {
	case model.Bool:
		return "bool"
	case model.Int8:
		return "int8"
	case model.Int16:
		return "int16"
	case model.Int32:
		return "int32"
	case model.Int64:
		return "int64"
	case model.Float32:
		return "float32"
	case model.Float64:
		return "float64"
	case model.String:
		return "string"
	case model.Blob:
		return "[]byte"
	case model.Decimal:
		return "*data.Decimal"
	case model.Timestamp:
		return "*data.Timestamp"
	case model.List:
		if gen.inlineSlicesAndMaps {
			return "[]" + gen.golangTypeRef(td.Items)
		}
	case model.Map:
		if gen.inlineSlicesAndMaps {
			return "map[" + gen.golangTypeRef(td.Keys) + "]" + gen.golangTypeRef(td.Items)
		}
	}
	return "*" + stripLocalNamespace(typeRef, gen.Schema.ServiceNamespace())
}

//for now, assume a single package, so we strip the namespace of non-language types
func golangTypeName(typeRef model.AbsoluteIdentifier) string {
	switch typeRef {
	case "api.Bytes":
		return "[]byte"
	case "api.Bool":
		return "bool"
	case "api.String":
		return "string"
	case "api.Int8":
		return "int8"
	case "api.Int16":
		return "int16"
	case "api.Int32":
		return "int32"
	case "api.Int64":
		return "int64"
	case "api.Float32":
		return "float32"
	case "api.Float64":
		return "float64"
	case "api.Decimal":
		return "*data.Decimal"
	case "api.Timestamp":
		return "*data.Timestamp"
	default:
		return stripNamespace(typeRef)
	}
}

/*
func golangifyTypesAndNames(d *data.Struct) {
	d.PutStringIfNotEmpty("golang-type", golangTypeName(d.GetString("type")))
	d.PutStringIfNotEmpty("golang-items", golangTypeName(d.GetString("items")))
	d.PutStringIfNotEmpty("golang-keys", golangTypeName(d.GetString("keys")))
	for _, f := range d.GetStructSlice("fields") {
		// f.GetBool("Required")
		f.PutStringIfNotEmpty("golang-type", golangTypeName(f.GetString("type")))
	}
}
*/
func (gen *Generator)goImports() map[string]bool {
	includes := make(map[string]bool, 0)
	deps := gen.AllTypeDependencies()
	for _, dep := range deps {
		switch dep {
		case "base#Decimal": //if expanded, then ["fmt", "math/big"]
			if gen.decimalPackage != "" {
				includes[gen.decimalPackage] = true
			} else {
				includes["fmt"] = true
				includes["math.big"] = true
			}
		case "base#Enum":
			includes["encoding/json"] = true
			includes["fmt"] = true
		case "base#Timestamp":
			if gen.timestampPackage != "" {
				includes[gen.timestampPackage] = true //if expanded, then ["encoding/json","fmt","strings","time"]
			}
		}
	}
	return includes
}
/*
func (gen *Generator) createModelContext() *data.Struct {
	top := data.NewStruct()
	var types []data.Value
	for _, td := range gen.Model.Schema.Types {
		ty := createTemplateContext(td)
		golangifyTypesAndNames(ty)
		types = append(types, ty)
	}
	top.PutString("golang-package", "main")
	top.PutString("golang-imports", gen.goImports())
	top.Put(data.NewString("types"), data.NewVector(types...))
	return top
}
*/
/*
func createServiceContext(model *model.Model, emitted map[string]bool) *data.Struct {
	top := data.NewStruct()
	top.PutString("name", model.Name)
	if model.Comment != "" {
		top.PutString("golang-comment", common.FormatComment("", "// ", model.Comment, 100, true))
	}
	var operations []data.Value
	var optypes []data.Value
	errtypes := make(map[string]data.Value, 0)
	for _, op := range model.Operations {
		opd := data.StructValueOf(op) //converts the whole thing to generic data
		operations = append(operations, opd)
		if op.Input != nil {
			opdi := data.AsStruct(opd.Get(data.NewString("input")))
			intype := op.Input.Name
			if intype == "" {
				intype = op.Name + "Input"
			}
			intypename := golangTypeName(intype)
			optypes = append(optypes, data.NewString(intypename))
			opdi.PutString("golang-name", intypename)
			opdi.PutString("golang-ref", golangTypeRef(intype, false))
			for _, i := range opdi.GetSlice("fields") {
				s := data.AsStruct(i)
				optional := true
				if s.GetBool("required") {
					optional = false
				}
				ftype := s.GetString("type")
				s.PutString("golang-ref", golangTypeRef(ftype, optional))
			}
		}
		if op.Output != nil {
			opdo := data.AsStruct(opd.Get(data.NewString("output")))
			outtype := op.Output.Name
			if outtype == "" {
				outtype = op.Name + "Output"
			}
			outtypename := golangTypeName(outtype)
			optypes = append(optypes, data.NewString(outtypename))
			opdo.PutString("golang-name", outtypename)
			opdo.PutString("golang-ref", golangTypeRef(outtype, false))
			for _, i := range opdo.GetSlice("fields") {
				s := data.AsStruct(i)
				optional := true
				if s.GetBool("required") {
					optional = false
				}
				ftype := s.GetString("type")
				s.PutString("golang-ref", golangTypeRef(ftype, optional))
			}
		}
		if op.Exceptions != nil {
			//as above, but N times? It is *likely* that each error will get reused in other operations
			//so generate a list for the entire service
			//check for congruence for a given name (guaranteed if smithy-originated, but not otherwise)
			for _, ods := range opd.GetStructSlice("exceptions") {
				outtype := ods.GetString("name")
				//assert outtype != ""
				outtypename := golangTypeName(outtype)
				if _, ok := errtypes[outtypename]; ok {
					//fmt.Println("duplicate error:", outtypename)
					//TO DO: ensure congruent
				} else {
					for _, i := range ods.GetSlice("fields") {
						s := data.AsStruct(i)
						optional := true
						if s.GetBool("required") {
							optional = false
						}
						ftype := s.GetString("type")
						s.PutString("golang-ref", golangTypeRef(ftype, optional))
					}
					errtypes[outtypename] = ods
				}
			}
		}
	}
	var errs []data.Value
	for k, v := range errtypes {
		errs = append(errs, v)
		emitted[k] = true
	}
	for _, k := range optypes {
		emitted[data.AsString(k)] = true
	}
	top.Put(data.NewString("operations"), data.NewVector(operations...))
	top.Put(data.NewString("golang-optypes"), data.NewVector(optypes...))
	top.Put(data.NewString("golang-exceptions"), data.NewVector(errs...))
	fmt.Println("service context:", data.Pretty(top))
	return top
}

func createTemplateContext(td *model.TypeDef) *data.Struct {
	context := data.StructValueOf(td)
	golangifyTypesAndNames(context)
	return context
}

*/


func stripNamespace(trait model.AbsoluteIdentifier) string {
	s := string(trait)
	n := strings.Index(s, "#")
	if n < 0 {
		return s
	}
	return s[n+1:]
}

func stripLocalNamespace(trait model.AbsoluteIdentifier, ns model.Namespace) string {
	t := string(trait)
	n := strings.Index(t, "#")
	if n < 0 {
		return t
	}
	if t[:n] == string(ns) {
		return t[n+1:]
	}
	return t
}

func (w *GolangWriter) golangTypeRef(typeRef model.AbsoluteIdentifier, required bool) string {
	bt := w.gen.Schema.BaseType(typeRef)
	switch bt {
	case model.Int8, model.Int16, model.Int32, model.Int64, model.Float32, model.Float64, model.String:
		return stripLocalNamespace(typeRef, w.gen.Schema.ServiceNamespace())
	}
	return golangTypeRef(typeRef, required)
}

func (gen *Generator) generateTypeComment(td *model.TypeDef, w *GolangWriter) {
	if td.Comment != "" {
		w.Emit("//\n")
		w.Emit(common.FormatComment("", "// ", td.Comment, 80, true))
		w.Emit("//\n")
	}
}

func (gen *Generator) GenerateType(td *model.TypeDef, w *GolangWriter) {
	w.Emit("\n")
	switch td.Base {
	case model.Struct:
		gen.generateTypeComment(td, w)
		w.Emitf("type %s struct {\n", golangTypeName(td.Id))
		for _, f := range td.Fields {
			opt := ""
			if !f.Required {
				opt = ",omitempty"
			}
			w.Emitf("    %s %s `json:\"%s%s\"`\n", common.Capitalize(f.Name), w.gen.golangTypeRef(f.Type), common.Uncapitalize(f.Name), opt)
		}
		w.Emitf("}\n")
	case model.String, model.Bool, model.Int8, model.Int16, model.Int32, model.Int64, model.Float32, model.Float64:
		if gen.inlinePrimitives {
			gen.generateTypeComment(td, w)
			w.Emitf("type %s %s\n", golangTypeName(td.Id), golangBaseTypeName(td.Base))
		}
		//    case model.BaseTypeTimestamp:
		//		return comment + "type " + gname + " *data.Timestamp\n"
	case model.List:
		if !gen.inlineSlicesAndMaps {
			gen.generateTypeComment(td, w)
			w.Emitf("type %s []%s\n", golangTypeName(td.Id), golangTypeRef(td.Items, false))
		}
    case model.Map:
		if !gen.inlineSlicesAndMaps {
			gen.generateTypeComment(td, w)
			w.Emitf("type %s map[%s]%s\n", golangTypeName(td.Id), golangTypeRef(td.Keys, false), golangTypeRef(td.Items, false))
		}
	default:
		gen.generateTypeComment(td, w)
		w.Emitf("type %s %s\n", golangTypeName(td.Id), golangBaseTypeName(td.Base))
		//	case model.BaseTypeInt:
		//		return comment + "type " + gname + " big.Int\n"
		//    case model.BaseTypeDecimal:
		//		return comment + "type " + gname + " big.Decimal\n"
		//    case model.BaseTypeBlob:
		//		return comment + "type " + gname + " []byte\n"
	}
}

func (gen *Generator) GenerateTypes() string {
	w := &GolangWriter{
		gen:       gen,
	}
	w.Begin()
	w.Emit("/* Generated */\n")
	w.Emitf("\npackage %s\n", gen.ns)
	imports := gen.goImports()
	if len(imports) > 0 {
		w.Emit(declareImports(imports))
	}
	
	for _, td := range gen.Schema.Types {
		gen.GenerateType(td, w)
	}
	return w.End()
}

func (gen *Generator) GenerateOperations() string {
	w := &GolangWriter{
		gen:       gen,
	}
	w.Begin()
	w.Emit("/* Generated */\n")
	w.Emitf("\npackage %s\n", gen.ns)
	imports := gen.goImports()
	if len(imports) > 0 {
		w.Emit(declareImports(imports))
	}

	if gen.Schema.Operations != nil && gen.Schema.ServiceName() != "" {
		w.EmitServiceInterface()
	}
	
	return w.End()
}

func declareImports(imports map[string]bool) string {
	s := ""
	if len(imports) > 0 {
		s = "\nimport(\n"
		for i, _ := range imports {
			s = s + fmt.Sprintf("    %q\n", i)
		}
		s = s + ")\n"
	}
	return s
}

func (gen *Generator) GenerateServer() string {
	schema := gen.Schema
	w := &GolangWriter{
		gen:       gen,
	}
	serviceName := golangTypeName(schema.Id)
	w.Begin()
	w.Emit("\n/* Generated */\n")
	w.Emitf("\npackage %s\n", gen.ns)
	imports := gen.goImports()
	imports["github.com/gorilla/mux"] = true
	imports["github.com/gorilla/handlers"] = true
	imports["net/http"] = true
	imports["fmt"] = true
	imports["io"] = true
	imports["os"] = true
	if len(imports) > 0 {
		w.Emit(declareImports(imports))
	}

	w.Emit("var _ = data.ParseTimestamp\n\n")
	adaptorName := common.Uncapitalize(serviceName) + "Adaptor"
	w.Emitf("type %s struct {\n", adaptorName)
	w.Emitf("    impl %s\n", serviceName)
	w.Emit("}\n")
	w.Emit("\n")
	
	for _, op := range schema.Operations {
		opName := golangTypeName(op.Id)
		opInput := ""
		opOutput := ""
		opStatus := int32(0)
		if op.Input != nil {
			opInput = golangTypeName(op.Input.Id)
		}
		if op.Output != nil {
			opOutput = golangTypeName(op.Output.Id)
			opStatus = op.Output.HttpStatus
		}
		w.Emitf("func (handler *%s) %sHandler(w http.ResponseWriter, r *http.Request) {\n", adaptorName, opName)
		arg := ""
		if opInput != "" {
			w.Emitf("    req := new(%s)\n", opInput)
			//iterate over the pathparams, queryparams, headerparams and bind them
			//bind entity if needed
			arg = "req"
		}
		result := ""
		resPayload := "nil"
		if opOutput != "" {
			resPayload = "res." + common.Capitalize(op.OutputHttpPayloadName())
			result = "res, "
		}
		w.Emitf("    %serr := handler.impl.%s(%s)\n", result, opName, arg)
		w.Emit("    if err != nil {\n")
		w.Emit("        switch err.(type) {\n")
		//iterate over declared exceptions
		w.Emit("        default:\n")
		w.Emit("            jsonResponse(w, 500, &serverError{Message: fmt.Sprint(err)})\n")
		w.Emit("        }\n")
		w.Emit("    } else {\n")
		w.Emitf("        jsonResponse(w, %d, %s)\n", opStatus, resPayload) //fixme: ensure payload declaration exists
		w.Emit("    }\n")
		w.Emit("}\n\n")
	}
	w.Emit(serverUtilSource)	
	return w.End()
}

func (gen *Generator) GenerateClient() string {
	w := &GolangWriter{
		gen:       gen,
	}

	w.Begin()
	w.Emit("/* Generated */\n")
	w.Emitf("\npackage %s\n", gen.ns)
	w.Emitf("\n%s\n", gen.goImports())

	return w.End()
}

func (w *GolangWriter) Begin() {
	w.buf.Reset()
	w.writer = bufio.NewWriter(&w.buf)
}

func (w *GolangWriter) End() string {
	w.writer.Flush()
	return w.buf.String()
}

func (w *GolangWriter) Emit(s string) {
	w.writer.WriteString(s)
}

func (w *GolangWriter) Emitf(format string, args ...interface{}) {
	w.writer.WriteString(fmt.Sprintf(format, args...))
}

func (w *GolangWriter) EmitServiceInterface() error {
	schema := w.gen.Schema
	if schema.Id != "" && !w.gen.HasEmitted(schema.Id) {
		w.Emit("\n")
		if schema.Comment != "" {
			w.Emit("//\n")
			w.Emit(common.FormatComment("", "// ", schema.Comment, 80, true))
			w.Emit("//\n")
		}
		w.Emitf("type %s interface {\n", golangTypeName(schema.Id)) //!
		for _, op := range schema.Operations {
			in := ""
			if op.Input != nil {
				in = w.golangTypeRef(op.Input.Id, false)
			}
			out := "error"
			if op.Output != nil {
				out = "(" + w.golangTypeRef(op.Output.Id, false) + ", error)"
			}
			w.Emitf("    %s(%s) %s\n", golangTypeName(op.Id), in, out)
		}
		w.Emit("}\n\n")
		for _, op := range schema.Operations {
			if op.Input != nil {
				w.Emitf("type %s struct {\n", golangTypeName(op.Input.Id))
				for _, f := range op.Input.Fields {
					opt := ""
					if !f.Required {
						opt = ",omitempty"
					}
					w.Emitf("    %s %s `json:\"%s%s\"`\n", f.Name.Capitalized(), w.gen.golangTypeRef(f.Type), f.Name.Uncapitalized(), opt)
				}
				w.Emitf("}\n\n")
			}
			if op.Output != nil {
				w.Emitf("type %s struct {\n", golangTypeName(op.Output.Id))
				for _, f := range op.Output.Fields {
					isRequired := true //Should I support f.Required?
					opt := ""
					if !isRequired {
						opt = ",omitempty"
					}
					w.Emitf("    %s %s `json:\"%s%s\"`\n", f.Name.Capitalized(), golangTypeRef(f.Type, !isRequired), f.Name.Uncapitalized(), opt)
				}
				w.Emitf("}\n\n")
			}
		}
		w.gen.Emitted(schema.Id)
	}
	return nil
}

var serverUtilSource = `func jsonResponse(w http.ResponseWriter, status int, entity interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    io.WriteString(w, data.Pretty(entity))
}

func param(r *http.Request, name string) string {
    return mux.Vars(r)[name]
}

func errorResponse(w http.ResponseWriter, status int, message string) {
    jsonResponse(w, status, &serverError{Error: http.StatusText(status), Message: message})
}

func normalizeHeaderValue(key string, value interface{}) string {
    switch v := value.(type) {
    case *data.Timestamp:
        return v.ToRfc2616String()
    case string:
        return v
    }
    return fmt.Sprint(value)
}

func intFromString(s string) int64 {
	var n int64 = 0
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

func floatFromString(s string) float64 {
	var n float64 = 0
	_, _ = fmt.Sscanf(s, "%g", &n)
	return n
}

// FoldHttpHeaderName adapts to the Go misfeature: all headers are
// canonicalized as Capslike-This (for a header "CapsLike-this").
func FoldHttpHeaderName(name string) string {
	return http.CanonicalHeaderKey(name)
}

type serverError struct {
    Error string  ` + "`json:\"error\"`" + `
    Message string ` + "`json:\"message\"`" + `
}

func WebLog(h http.Handler) http.Handler {
	return handlers.CombinedLoggingHandler(os.Stdout, h)
}

func AllowCors(next http.Handler, host string) http.Handler {
   return handlers.CORS(handlers.AllowedOrigins([]string{"*"}), handlers.AllowedHeaders([]string{"Content-Type", "api_key", "Authorization"}), handlers.AllowedMethods([]string{"GET","PUT","DELETE","POST","OPTIONS"}))(next)
}
`
