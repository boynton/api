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

	"github.com/boynton/api/model"
	"github.com/boynton/data"
)

const IndentAmount = "    "

type Generator struct {
	model.BaseGenerator
	ns                  model.Namespace
	pkg                 string
	inlineSlicesAndMaps bool   //more idiomatic, but prevents validating constraints (i.e. list.maxLength)
	inlinePrimitives    bool   //more idomatic, but prevents validating constraints (i.e. string.Pattern)
	decimalPackage      string //use this package for the Decimal implementation. If "", then generate one in this package
	decimalPrefix       string //derived from the decimalPackage
	timestampPackage    string //use this package for the Timestamp implementation. If "", then generate one in this package
	timestampPrefix     string
	anyPackage          string //use this package for the Any type, if "", then generate one in this package
	anyPrefix           string //derived from the anyPackage
}

func (gen *Generator) GenerateOperation(op *model.OperationDef) error {
	return nil
}

func (gen *Generator) GenerateType(td *model.TypeDef) error {
	return nil
}

func (gen *Generator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.inlineSlicesAndMaps = config.GetBool("golang.inlineSlicesAndMaps")
	gen.inlinePrimitives = config.GetBool("golang.inlinePrimitives")
	gen.anyPackage = config.GetString("golang.anyPackage")
	if gen.anyPackage == "" {
		gen.anyPackage = "github.com/boynton/data"
	}
	gen.anyPrefix = path.Base(gen.anyPackage) + "."
	gen.decimalPackage = config.GetString("golang.decimalPackage")
	if gen.decimalPackage == "" {
		gen.decimalPackage = "github.com/boynton/data"
	}
	gen.decimalPrefix = path.Base(gen.decimalPackage) + "."
	gen.timestampPackage = config.GetString("golang.timestampPackage")
	if gen.timestampPackage != "" {
		gen.timestampPrefix = path.Base(gen.timestampPackage) + "."
	}
	gen.ns = model.Namespace(config.GetString("namespace"))
	if gen.ns == "" {
		gen.ns = schema.ServiceNamespace()
		if gen.ns == "" {
			gen.ns = schema.Namespace
		}
	}
	//go doesn't like compound package names, take only the last component of namespace
	el := strings.Split(string(gen.ns), ".")
	gen.pkg = el[len(el)-1]

	fbase := string(gen.pkg)
	if fbase == "" {
		fbase = "model"
	}
	err = gen.Validate() //for golang generation
	if err != nil {
		return err
	}

	fname := gen.FileName(fbase+"_types", ".go")
	s := gen.GenerateTypes()
	err = gen.Write(s, fname, "\n\n------------------"+fname+"\n")
	if err != nil {
		return err
	}
	if len(gen.Schema.Operations) > 0 {
		s = gen.GenerateOperations()
		fname = gen.FileName(fbase+"_operations", ".go")
		err = gen.Write(s, fname, "\n\n------------------"+fname+"\n")
		if err != nil {
			return err
		}
	}
	if len(gen.Schema.Operations) > 0 {
		fname = gen.FileName(fbase+"_server", ".go")
		s = gen.GenerateServer()
		err = gen.Write(s, fname, "\n\n------------------"+fname+"\n")
		if err != nil {
			return err
		}
		/*
			fname = gen.FileName(fbase + "_client", ".go")
			s = gen.GenerateClient(ns, model)
			err = gen.Write(s, fname, "\n\n------------------" + fname + "\n")
			if err != nil {
				return err
			}
		*/
	}
	return nil
}

func (gen *Generator) Validate() error {
	//this is to validate the model for Go code gen. For example, only http APIs currently are implemented (not RPC)
	return nil
}

type GolangWriter struct {
	buf    bytes.Buffer
	writer *bufio.Writer
	gen    *Generator
}

func (gen *Generator) golangBaseTypeName(bt model.BaseType) string {
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
	case model.Blob:
		return "[]byte"
	case model.String:
		return "string"
	case model.Integer:
		return "*" + gen.decimalPrefix + "Integer"
	case model.Decimal:
		return "*" + gen.decimalPrefix + "Decimal"
	case model.Timestamp:
		return "*" + gen.timestampPrefix + "Timestamp"
	case model.Any:
		return gen.anyPrefix + "Any"
	default:
		fmt.Println("bt:", bt)
		panic("not concrete")
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
	case "base#Integer":
		return "*" + gen.decimalPrefix + "Integer"
	case "base#Timestamp":
		return "*" + gen.timestampPrefix + "Timestamp"
	case "base#Any":
		return gen.anyPrefix + "Any"
	default:
		//not a base type, but an operation input or output (looks like a Struct, but not declared as a type)
		return "*" + stripLocalNamespace(typeRef, gen.ns)
	}
}

func (gen *Generator) golangTypeRef(typeRef model.AbsoluteIdentifier) string {
	td := gen.Schema.GetTypeDef(typeRef)
	if td == nil {
		return gen.baseTypeRef(typeRef)
	}
	indirect := ""
	switch td.Base {
	case model.Bool:
		if gen.inlinePrimitives {
			return "bool"
		}
	case model.Int8:
		if gen.inlinePrimitives {
			return "int8"
		}
	case model.Int16:
		if gen.inlinePrimitives {
			return "int16"
		}
	case model.Int32:
		if gen.inlinePrimitives {
			return "int32"
		}
	case model.Int64:
		if gen.inlinePrimitives {
			return "int64"
		}
	case model.Float32:
		if gen.inlinePrimitives {
			return "float32"
		}
	case model.Float64:
		if gen.inlinePrimitives {
			return "float64"
		}
	case model.String:
		if gen.inlinePrimitives {
			return "string"
		}
	case model.Blob:
		if gen.inlinePrimitives {
			return "[]byte"
		}
	case model.Integer:
		if gen.inlinePrimitives {
			return "*data.Integer"
		}
		indirect = "*"
	case model.Decimal:
		if gen.inlinePrimitives {
			return "*data.Decimal"
		}
		indirect = "*"
	case model.Timestamp:
		if gen.inlinePrimitives {
			return "*data.Timestamp"
		}
		indirect = "*"
	case model.List:
		if gen.inlineSlicesAndMaps {
			return "[]" + gen.golangTypeRef(td.Items)
		}
	case model.Map:
		if gen.inlineSlicesAndMaps {
			return "map[" + gen.golangTypeRef(td.Keys) + "]" + gen.golangTypeRef(td.Items)
		}
		indirect = "*"
	case model.Enum:
		//no indirection
	default:
		indirect = "*"
	}
	return indirect + stripLocalNamespace(typeRef, gen.ns)
}

// for now, assume a single package, so we strip the namespace of non-language types
func (gen *Generator) golangTypeName(typeRef model.AbsoluteIdentifier) string {
	switch typeRef {
	case "base#Bytes":
		return "[]byte"
	case "base#Bool":
		return "bool"
	case "base@String":
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
		return "*data.Decimal"
	case "base#Timestamp":
		return "*data.Timestamp"
	default:
		return stripNamespace(typeRef)
	}
}

func (gen *Generator) goImports(forDef bool) map[string]bool {
	includes := make(map[string]bool, 0)
	deps := gen.AllTypeDependencies()
	for _, dep := range deps {
		bt := gen.Schema.BaseType(dep)
		switch bt {
		case model.Decimal: //if expanded, then ["fmt", "math/big"]
			if gen.decimalPackage != "" {
				includes[gen.decimalPackage] = true
			} else {
				includes["fmt"] = true
				includes["math/big"] = true
			}
		case model.Enum:
			if forDef {
				includes["encoding/json"] = true
				includes["fmt"] = true
			}
		case model.Timestamp:
			if gen.timestampPackage != "" {
				includes[gen.timestampPackage] = true //if expanded, then ["encoding/json","fmt","strings","time"]
			}
		}
	}
	return includes
}

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

func (w *GolangWriter) xgolangTypeRef(typeRef model.AbsoluteIdentifier, required bool) string {
	bt := w.gen.Schema.BaseType(typeRef)
	switch bt {
	case model.Int8, model.Int16, model.Int32, model.Int64, model.Float32, model.Float64, model.String:
		return stripLocalNamespace(typeRef, w.gen.ns)
	}
	return w.gen.golangTypeRef(typeRef)
}

func (gen *Generator) generateTypeComment(td *model.TypeDef, w *GolangWriter) {
	if td.Comment != "" {
		w.Emit("//\n")
		w.Emit(model.FormatComment("", "// ", td.Comment, 80, true))
		w.Emit("//\n")
	}
}

func (gen *Generator) generateType(td *model.TypeDef, w *GolangWriter) {
	w.Emit("\n")
	switch td.Base {
	case model.Struct:
		gen.generateTypeComment(td, w)
		w.Emitf("type %s struct {\n", gen.golangTypeName(td.Id))
		for _, f := range td.Fields {
			opt := ""
			if !f.Required {
				opt = ",omitempty"
			}
			name := string(f.Name)
			w.Emitf("    %s %s `json:\"%s%s\"`\n", model.Capitalize(name), w.gen.golangTypeRef(f.Type), model.Uncapitalize(name), opt)
		}
		w.Emitf("}\n")
	case model.Union:
		gen.generateTypeComment(td, w)
		tname := gen.golangTypeName(td.Id)
		w.Emitf("type %sVariantTag int\n", tname)
		w.Emitf("const (\n")
		w.Emitf("    _ %sVariantTag = iota\n", tname)
		for _, f := range td.Fields {
			w.Emitf("    %sVariantTag%s\n", tname, model.Capitalize(string(f.Name)))
		}
		w.Emitf(")\n")
		w.Emitf("type %s struct {\n", tname)
		w.Emitf("    Variant %sVariantTag `json:\"-\"`\n", tname) //seems convenient, but how to set on json.Unmarshal?
		for _, f := range td.Fields {
			opt := ",omitempty"
			name := string(f.Name)
			w.Emitf("    %s %s `json:\"%s%s\"`\n", model.Capitalize(name), w.gen.golangTypeRef(f.Type), model.Uncapitalize(name), opt)
		}
		w.Emitf("}\n")
		w.Emitf("type raw%s struct {\n", tname)
		for _, f := range td.Fields {
			name := string(f.Name)
			w.Emitf("    %s %s%s `json:\"%s,omitempty\"`\n", model.Capitalize(name), gen.indirectOp(f.Type), w.gen.golangTypeRef(f.Type), model.Uncapitalize(name))
		}
		w.Emitf("}\n")
		w.Emitf("func (u *%s) UnmarshalJSON(b []byte) error {\n", tname)
		w.Emitf("    var tmp raw%s\n", tname)
		w.Emitf("    if err := json.Unmarshal(b, &tmp); err != nil {\n")
		w.Emitf("        return err\n")
		w.Emitf("    }\n")
		p := "if "
		for _, f := range td.Fields {
			fname := model.Capitalize(string(f.Name))
			w.Emitf("    %s tmp.%s != nil {\n", p, fname)
			w.Emitf("        u.Variant = %s%s\n", tname, fname)
			w.Emitf("        u.%s = %stmp.%s\n", fname, gen.indirectOp(f.Type), fname)
			p = "} else if "
		}
		w.Emitf("    } else {\n")
		w.Emitf("        return fmt.Errorf(\"%s: Missing required variant\")\n", tname)
		w.Emitf("    }\n")
		w.Emitf("    return nil\n")
		w.Emitf("}\n")
	case model.String, model.Bool, model.Int8, model.Int16, model.Int32, model.Int64, model.Float32, model.Float64:
		if !gen.inlinePrimitives {
			gen.generateTypeComment(td, w)
			w.Emitf("type %s %s\n", gen.golangTypeName(td.Id), gen.golangBaseTypeName(td.Base))
		}
		//    case model.BaseTypeTimestamp:
		//		return comment + "type " + gname + " *data.Timestamp\n"
	case model.List:
		if !gen.inlineSlicesAndMaps {
			gen.generateTypeComment(td, w)
			w.Emitf("type %s []%s\n", gen.golangTypeName(td.Id), gen.golangTypeRef(td.Items))
		}
	case model.Map:
		if !gen.inlineSlicesAndMaps {
			gen.generateTypeComment(td, w)
			w.Emitf("type %s map[%s]%s\n", gen.golangTypeName(td.Id), gen.golangTypeRef(td.Keys), gen.golangTypeRef(td.Items))
		}
	case model.Enum:
		gen.generateTypeComment(td, w)
		tname := gen.golangTypeName(td.Id)
		w.Emitf("type %s int\n", tname)
		w.Emitf("const (\n")
		w.Emitf("    _ %s = iota\n", tname)
		for _, e := range td.Elements {
			w.Emitf("    %s\n", e.Symbol)
		}
		w.Emitf(")\n")
		w.Emitf("var names%s = []string{\n", tname)
		for _, e := range td.Elements {
			w.Emitf("    %s: %q,\n", e.Symbol, e.Value) //assumes string enum!
		}
		w.Emitf("}\n")
		w.Emitf("func (e %s) String() string {\n", tname)
		w.Emitf("    return names%s[e]\n", tname)
		w.Emitf("}\n")
		w.Emitf("func (e %s) MarshalJSON() ([]byte, error) {\n", tname)
		w.Emitf("    return json.Marshal(e.String())\n")
		w.Emitf("}\n")
		w.Emitf("func (e *%s) UnmarshalJSON(b []byte) error {\n", tname)
		w.Emitf("    var s string\n")
		w.Emitf("    err := json.Unmarshal(b, &s)\n")
		w.Emitf("    if err == nil {\n")
		w.Emitf("        for v, s2 := range names%s {\n", tname)
		w.Emitf("            if s == s2 {\n")
		w.Emitf("                *e = %s(v)\n", tname)
		w.Emitf("                return nil\n")
		w.Emitf("             }\n")
		w.Emitf("        }\n")
		w.Emitf("        err = fmt.Errorf(\"Bad enum symbol for type %s: %%s\", s)\n", tname)
		w.Emitf("    }\n")
		w.Emitf("    return err\n")
		w.Emitf("}\n\n")
	default:
		gen.generateTypeComment(td, w)
		w.Emitf("type %s %s\n", gen.golangTypeName(td.Id), gen.golangBaseTypeName(td.Base))
	}
}

func (gen *Generator) indirectOp(id model.AbsoluteIdentifier) string {
	indirect := ""
	switch gen.Schema.BaseType(id) {
	case model.Bool, model.Int8, model.Int16, model.Int32, model.Int64, model.Float32, model.Float64, model.String:
		indirect = "*"
	}
	return indirect
}

func (gen *Generator) GenerateTypes() string {
	w := &GolangWriter{
		gen: gen,
	}
	w.Begin()
	w.Emit("/* Generated */\n")
	w.Emitf("\npackage %s\n", gen.pkg)
	imports := gen.goImports(true)
	if len(imports) > 0 {
		w.Emit(declareImports(imports))
	}
	for _, td := range gen.Schema.Types {
		gen.generateType(td, w)
	}
	return w.End()
}

func (gen *Generator) GenerateOperations() string {
	w := &GolangWriter{
		gen: gen,
	}
	w.Begin()
	w.Emit("/* Generated */\n")
	w.Emitf("\npackage %s\n", gen.pkg)
	imports := gen.goImports(false)
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
		for i := range imports {
			s = s + fmt.Sprintf("    %q\n", i)
		}
		s = s + ")\n"
	}
	return s
}

func (gen *Generator) baseConverter(typeref model.AbsoluteIdentifier, def interface{}, body string) string {
	s := gen.golangTypeRef(typeref) //this expands to primitives. Still need a typecast if golang.inlinePrimitives=false
	switch s {
	case "int8", "int16", "int32", "int64":
		r := fmt.Sprintf("intParam(%s, %d)", body, data.AsInt64(def))
		if s != "int64" {
			r = s + "(" + r + ")"
		}
		return r
	case "*data.Timestamp":
		return fmt.Sprintf("timestampParam(%s, %q)", body, data.AsString(def))
	default:
		return fmt.Sprintf("stringParam(%s, %q)", body, data.AsString(def))
	}
}

func (gen *Generator) GenerateServer() string {
	schema := gen.Schema
	w := &GolangWriter{
		gen: gen,
	}
	serviceName := gen.golangTypeName(schema.Id)
	w.Begin()
	w.Emit("\n/* Generated */\n")
	w.Emitf("\npackage %s\n", gen.pkg)
	imports := gen.goImports(false)
	imports["github.com/gorilla/mux"] = true
	imports["github.com/gorilla/handlers"] = true
	imports["net/http"] = true
	imports["net/url"] = true
	imports["encoding/json"] = true
	imports["strings"] = true
	imports["strconv"] = true
	imports["log"] = true
	imports["fmt"] = true
	imports["io"] = true
	imports["os"] = true
	if len(imports) > 0 {
		w.Emit(declareImports(imports))
	}

	w.Emit("var _ = data.ParseTimestamp\n\n")
	adaptorName := model.Uncapitalize(serviceName) + "Adaptor"
	w.Emitf("type %s struct {\n", adaptorName)
	w.Emitf("    impl %s\n", serviceName)
	w.Emit("}\n")
	w.Emit("\n")

	for _, op := range schema.Operations {
		opName := gen.golangTypeName(op.Id)
		opInput := ""
		opOutput := ""
		opStatus := int32(0)
		if op.Input != nil {
			opInput = gen.golangTypeName(op.Input.Id)
		}
		if op.Output != nil {
			opOutput = gen.golangTypeName(op.Output.Id)
			opStatus = op.Output.HttpStatus
		}
		w.Emitf("func (handler *%s) %sHandler(w http.ResponseWriter, r *http.Request) {\n", adaptorName, opName)
		arg := ""
		q := false
		if opInput != "" {
			w.Emitf("    req := new(%s)\n", opInput)
			for _, f := range op.Input.Fields {
				if f.HttpQuery != "" {
					q = true
				} else if f.HttpPath {
					body := fmt.Sprintf("mux.Vars(r)[%q]", f.Name)
					w.Emitf("    req.%s = %s\n", model.Capitalize(string(f.Name)), gen.baseConverter(f.Type, f.Default, body))
				} else if f.HttpPayload {
					w.Emitf("    err := json.NewDecoder(r.Body).Decode(&req.%s)\n", model.Capitalize(string(f.Name)))
					w.Emitf("    if err != nil {\n")
					w.Emitf("        errorResponse(w, 400, fmt.Sprint(err))\n")
					w.Emitf("        return\n")
					w.Emitf("    }\n")
				} else {
					//?headers
				}
			}
			if q {
				w.Emitf("    err := r.ParseForm()\n")
				w.Emitf("    if err != nil {\n")
				w.Emitf("        errorResponse(w, 400, fmt.Sprint(err))\n")
				w.Emitf("        return\n")
				w.Emitf("    }\n")
				for _, f := range op.Input.Fields {
					if f.HttpQuery != "" {
						body := fmt.Sprintf("r.Form.Get(%q)", f.Name)
						w.Emitf("    req.%s = %s\n", model.Capitalize(string(f.Name)), gen.baseConverter(f.Type, f.Default, body))
					}
				}
				w.Emitf("    // emit the query params here\n")
			}
			//headers!
			//entity!
			arg = "req"
		}
		result := ""
		resPayload := "nil"
		if opOutput != "" {
			resPayload = "res." + model.Capitalize(op.OutputHttpPayloadName())
			result = "res, "
		}
		w.Emitf("    %serr := handler.impl.%s(%s)\n", result, opName, arg)
		w.Emit("    if err != nil {\n")
		w.Emit("        switch e := err.(type) {\n")
		for _, e := range op.Exceptions {
			w.Emitf("       case %s:\n", gen.golangTypeRef(e.Id))
			p := gen.entityPayload(e)
			w.Emitf("           jsonResponse(w, %d, e%s)\n", e.HttpStatus, p)
		}
		w.Emit("        default:\n")
		w.Emit("            jsonResponse(w, 500, &serverError{Message: fmt.Sprint(err)})\n")
		w.Emit("        }\n")
		w.Emit("    } else {\n")
		w.Emitf("        jsonResponse(w, %d, %s)\n", opStatus, resPayload)
		w.Emit("    }\n")
		w.Emit("}\n\n")
	}
	w.Emitf("func InitServer(impl %s, baseURL string) http.Handler {\n", serviceName)
	w.Emitf("    adaptor := &%s{\n", adaptorName)
	w.Emitf("        impl: impl,\n")
	w.Emitf("    }\n")
	w.Emitf("    u, err := url.Parse(strings.TrimSuffix(baseURL, \"/\"))\n")
	w.Emitf("    if err != nil {\n")
	w.Emitf("        log.Fatal(err)\n")
	w.Emitf("    }\n")
	w.Emitf("    b := u.Path\n")
	w.Emitf("    r := mux.NewRouter()\n\n")
	for _, op := range schema.Operations {
		opName := gen.golangTypeName(op.Id)
		w.Emitf("    r.HandleFunc(b+%q, func(w http.ResponseWriter, r *http.Request) {\n", op.HttpUri)
		w.Emitf("        adaptor.%sHandler(w, r)\n", opName)
		w.Emitf("    }).Methods(%q)\n", op.HttpMethod)
	}
	w.Emitf("    r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n")
	w.Emit("        jsonResponse(w, 404, &serverError{Message: fmt.Sprintf(\"Not Found: %s\", r.URL.Path)})\n")
	w.Emitf("    })\n")
	w.Emitf("    return r\n")
	w.Emitf("}\n\n")
	w.Emit(serverUtilSource)
	return w.End()
}

func (gen *Generator) entityPayload(out *model.OperationOutput) string {
	for _, field := range out.Fields {
		if field.HttpPayload {
			return "." + model.Capitalize(string(field.Name))
		}
	}
	return ""
}

func (gen *Generator) GenerateClient() string {
	w := &GolangWriter{
		gen: gen,
	}

	w.Begin()
	w.Emit("/* Generated */\n")
	w.Emitf("\npackage %s\n", gen.pkg)
	imports := gen.goImports(false)
	if len(imports) > 0 {
		w.Emit(declareImports(imports))
	}

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
	gen := w.gen
	schema := gen.Schema
	if schema.Id != "" && !w.gen.HasEmitted(schema.Id) {
		w.Emit("\n")
		if schema.Comment != "" {
			w.Emit("//\n")
			w.Emit(model.FormatComment("", "// ", schema.Comment, 80, true))
			w.Emit("//\n")
		}
		w.Emitf("type %s interface {\n", gen.golangTypeName(schema.Id)) //!
		for _, op := range schema.Operations {
			in := ""
			if op.Input != nil {
				//in = w.golangTypeRef(op.Input.Id, false)
				in = gen.golangTypeRef(op.Input.Id)
			}
			out := "error"
			if op.Output.Id != "" {
				//out = "(" + w.golangTypeRef(op.Output.Id, false) + ", error)"
				out = "(" + gen.golangTypeRef(op.Output.Id) + ", error)"
			}
			w.Emitf("    %s(%s) %s\n", gen.golangTypeName(op.Id), in, out)
		}
		w.Emit("}\n\n")
		for _, op := range schema.Operations {
			if op.Input != nil {
				if !w.gen.HasEmitted(op.Input.Id) {
					w.Emitf("type %s struct {\n", gen.golangTypeName(op.Input.Id))
					for _, f := range op.Input.Fields {
						opt := ""
						if !f.Required {
							opt = ",omitempty"
						}
						w.Emitf("    %s %s `json:\"%s%s\"`\n", f.Name.Capitalized(), w.gen.golangTypeRef(f.Type), f.Name.Uncapitalized(), opt)
					}
					w.Emitf("}\n\n")
					w.gen.Emitted(op.Input.Id)
				}
			}
			if op.Output.Id != "" {
				if !w.gen.HasEmitted(op.Output.Id) {
					w.Emitf("type %s struct {\n", gen.golangTypeName(op.Output.Id))
					for _, f := range op.Output.Fields {
						isRequired := true //Should I support f.Required?
						opt := ""
						if !isRequired {
							opt = ",omitempty"
						}
						//w.Emitf("    %s %s `json:\"%s%s\"`\n", f.Name.Capitalized(), golangTypeRef(f.Type, !isRequired), f.Name.Uncapitalized(), opt)
						w.Emitf("    %s %s `json:\"%s%s\"`\n", f.Name.Capitalized(), gen.golangTypeRef(f.Type), f.Name.Uncapitalized(), opt)
					}
					w.Emitf("}\n\n")
					w.gen.Emitted(op.Output.Id)
				}
			}
			if op.Exceptions != nil {
				for _, e := range op.Exceptions {
					if !w.gen.HasEmitted(e.Id) {
						eType := gen.golangTypeName(e.Id)
						w.Emitf("type %s struct {\n", eType)
						//hack: the first string field defined will be used to create the error message
						msg := ""
						for _, f := range e.Fields {
							isRequired := true                   //Should I support f.Required?
							if string(f.Type) == "base#String" { //fix: isStringBase(e.Type)
								msg = model.Capitalize(string(f.Name))
							}
							opt := ""
							if !isRequired {
								opt = ",omitempty"
							}
							//w.Emitf("    %s %s `json:\"%s%s\"`\n", f.Name.Capitalized(), golangTypeRef(f.Type, !isRequired), f.Name.Uncapitalized(), opt)
							w.Emitf("    %s %s `json:\"%s%s\"`\n", f.Name.Capitalized(), gen.golangTypeRef(f.Type), f.Name.Uncapitalized(), opt)
						}
						if msg == "" {
							msg = "String()"
						}
						w.Emitf("}\n\n")
						w.Emitf("func (e *%s) Error() string {\n", eType)
						w.Emitf("    return e.%s\n", msg)
						w.Emitf("}\n\n")
						w.gen.Emitted(e.Id)
					}
				}
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

func stringParam(val string, def string) string {
    if val == "" {
        return def
    }
    return val
}

func timestampParam(val string, def string) *data.Timestamp {
    if val != "" {
        ts, err := data.ParseTimestamp(val)
        if err != nil {
            return &ts
        }
    }
    ts, err := data.ParseTimestamp(def)
    if err != nil {
        return &ts
    }
    return nil
}

func intParam(val string, def int64) int64 {
    if val != "" {
        i, err := strconv.ParseInt(val, 10, 64)
        if err == nil {
            return i
        }
    }
    return def
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
