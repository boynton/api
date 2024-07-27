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
package model

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/boynton/data"
)

type Generator interface {
	Configure(schema *Schema, conf *data.Object) error
	Generate(schema *Schema, config *data.Object) error
	GenerateResource(op *ResourceDef) error
	GenerateOperation(op *OperationDef) error
	GenerateException(exc *OperationOutput) error
	GenerateType(td *TypeDef) error
	Begin()
	End() string
	Sorted() bool
}

type BaseGenerator struct {
	Schema         *Schema
	Config         *data.Object
	OutDir         string
	ForceOverwrite bool
	buf            bytes.Buffer
	writer         *bufio.Writer
	Err            error
	Sort           bool
	typesEmitted   map[AbsoluteIdentifier]bool
}

func (gen *BaseGenerator) Sorted() bool {
	return gen.Sort
}

func (gen *BaseGenerator) Configure(schema *Schema, conf *data.Object) error {
	gen.Schema = schema
	gen.typesEmitted = make(map[AbsoluteIdentifier]bool, 0)
	//validate the config
	gen.Config = conf
	gen.OutDir = conf.GetString("outdir")
	gen.Sort = conf.GetBool("sort")
	gen.ForceOverwrite = conf.GetBool("force")
	return nil
}

func (gen *BaseGenerator) Operations() []*OperationDef {
	if gen.Sort {
		return gen.SortedOperations()
	}
	return gen.Schema.Operations
}

func (gen *BaseGenerator) SortedOperations() []*OperationDef {
	var r []*OperationDef
	if len(gen.Schema.Operations) > 0 {
		for _, i := range gen.Schema.Operations {
			r = append(r, i)
		}
		sort.Slice(r, func(i, j int) bool {
			return StripNamespace(r[i].Id) < StripNamespace(r[j].Id)
		})
	}
	return r
}

func (gen *BaseGenerator) Exceptions() []*OperationOutput {
	if gen.Sort {
		lst := gen.SortedExceptions()
		return lst
	}
	return gen.Schema.Exceptions
}

func (gen *BaseGenerator) SortedExceptions() []*OperationOutput {
	var r []*OperationOutput
	if len(gen.Schema.Exceptions) > 0 {
		for _, i := range gen.Schema.Exceptions {
			r = append(r, i)
		}
		sort.Slice(r, func(i, j int) bool {
			return StripNamespace(r[i].Id) < StripNamespace(r[j].Id)
		})
	}
	return r
}

func (gen *BaseGenerator) Types() []*TypeDef {
	if gen.Sort {
		return gen.SortedTypes()
	}
	return gen.Schema.Types
}

func (gen *BaseGenerator) SortedTypes() []*TypeDef {
	var r []*TypeDef
	if len(gen.Schema.Types) > 0 {
		for _, i := range gen.Schema.Types {
			r = append(r, i)
		}
		sort.Slice(r, func(i, j int) bool {
			return StripNamespace(r[i].Id) < StripNamespace(r[j].Id)
		})
	}
	return r
}

func (gen *BaseGenerator) HasEmitted(id AbsoluteIdentifier) bool {
	if gen.typesEmitted != nil {
		if b, ok := gen.typesEmitted[id]; ok && b {
			return true
		}
	}
	return false
}

func (gen *BaseGenerator) Emitted(id AbsoluteIdentifier) {
	gen.typesEmitted[id] = true
}

func (gen *BaseGenerator) Begin() {
	gen.buf.Reset()
	gen.writer = bufio.NewWriter(&gen.buf)
}

func (gen *BaseGenerator) End() string {
	gen.writer.Flush()
	return gen.buf.String()
}

func (gen *BaseGenerator) Emit(s string) {
	gen.writer.WriteString(s)
}

func (gen *BaseGenerator) Emitf(format string, args ...interface{}) {
	gen.writer.WriteString(fmt.Sprintf(format, args...))
}

func (gen *BaseGenerator) FileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func (gen *BaseGenerator) FileName(ns string, suffix string) string {
	return strings.ReplaceAll(ns, ".", "-") + suffix
}

func (gen *BaseGenerator) WriteFile(path string, content string) error {
	if gen.Err != nil {
		return gen.Err
	}
	if !gen.ForceOverwrite && gen.FileExists(path) {
		return fmt.Errorf("[%s already exists, not overwriting]", path)
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("cannot create %q\n", path)
		panic("ere")
		gen.Err = err
		return err
	}
	defer f.Close()
	writer := bufio.NewWriter(f)
	_, gen.Err = writer.WriteString(content)
	writer.Flush()
	return gen.Err
}

/*
   func (gen *BaseGenerator) RenderTemplate(tmplFs embed.FS, tmplName string, context *data.Object) (string, error) {
	ts := NewFSTemplateSet(tmplFs)
	tmpl, err := ts.GetTemplate(tmplName)
	if err != nil {
		panic("Whoops")
		return "", err
	}
	return tmpl.Execute(context)
}
*/

func (gen *BaseGenerator) EnsureDir(path string) error {
	if _, err := os.Stat(path); err != nil {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

func (gen *BaseGenerator) Write(text string, filename string, separator string) error {
	if gen.Err != nil {
		return gen.Err
	}
	if strings.HasPrefix(filename, "/") || strings.HasPrefix(filename, ".") {
		gen.Err = gen.WriteFile(filename, text)
	} else if gen.OutDir == "" {
		if separator != "" {
			fmt.Print(separator)
		}
		fmt.Print(text)
		gen.Err = nil
	} else {
		gen.Err = gen.EnsureDir(gen.OutDir)
		if gen.Err == nil {
			fpath := filepath.Join(gen.OutDir, filename)
			gen.Err = gen.WriteFile(fpath, text)
		}
	}
	return gen.Err
}

func (gen *BaseGenerator) accumulateDependenciesById(deps map[AbsoluteIdentifier]bool, id AbsoluteIdentifier) {
	switch id {
	case "base#Bool", "base#Int8", "base#Int16", "base#Int32", "base#Int64", "base#Float32", "base#Float64", "base#Bytes", "base#String", "base#Enum":
		return
	case "base#Timestamp", "base#Decimal":
		deps[id] = true
	}
	td := gen.Schema.GetTypeDef(id)
	if td == nil {
		return
	}
	gen.accumulateDependencies(deps, td)
}

func (gen *BaseGenerator) accumulateDependencies(deps map[AbsoluteIdentifier]bool, td *TypeDef) {
	if td == nil {
		return
	}
	if _, ok := deps[td.Id]; ok {
		return
	}
	deps[td.Id] = true
	switch td.Base {
	case BaseType_Bool, BaseType_Int8, BaseType_Int16, BaseType_Int32, BaseType_Int64, BaseType_Float32, BaseType_Float64:
		return
	case BaseType_Decimal, BaseType_Blob, BaseType_String, BaseType_Timestamp, BaseType_Enum:
		return
	case BaseType_List:
		gen.accumulateDependenciesById(deps, td.Items)
	case BaseType_Map:
		gen.accumulateDependenciesById(deps, td.Keys)
		gen.accumulateDependenciesById(deps, td.Items)
	case BaseType_Struct, BaseType_Union:
		for _, f := range td.Fields {
			gen.accumulateDependenciesById(deps, f.Type)
		}
	}
}

func (gen *BaseGenerator) accumulateOpDependencies(deps map[AbsoluteIdentifier]bool, op *OperationDef) {
	if op.Input != nil {
		for _, f := range op.Input.Fields {
			gen.accumulateDependenciesById(deps, f.Type)
		}
	}
	if op.Output != nil {
		for _, f := range op.Output.Fields {
			gen.accumulateDependenciesById(deps, f.Type)
		}
	}
	for _, eid := range op.Exceptions {
		edef := gen.Schema.GetExceptionDef(eid)
		for _, f := range edef.Fields {
			gen.accumulateDependenciesById(deps, f.Type)
		}
	}
}

func (gen *BaseGenerator) TypeDependencies(td *TypeDef) []AbsoluteIdentifier {
	deps := make(map[AbsoluteIdentifier]bool, 0)
	gen.accumulateDependencies(deps, td)
	var result []AbsoluteIdentifier
	for k := range deps {
		result = append(result, k)
	}
	return result
}

func (gen *BaseGenerator) AllTypeDependencies() []AbsoluteIdentifier {
	deps := make(map[AbsoluteIdentifier]bool, 0)
	for _, td := range gen.Schema.Types {
		gen.accumulateDependencies(deps, td)
	}
	if true {
		for _, op := range gen.Schema.Operations {
			gen.accumulateOpDependencies(deps, op)
		}
	}
	var result []AbsoluteIdentifier
	for k := range deps {
		result = append(result, k)
	}
	return result
}
