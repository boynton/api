package httptrace

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/boynton/data"
	"github.com/boynton/api/model"
)

type Generator struct {
	model.BaseGenerator
}

func (gen *Generator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	for _, op := range gen.Operations() {
		for _, example := range op.Examples {
			snippet, err := gen.EmitHttpTrace(op, example)
			if err != nil {
				fmt.Println("*** Error:", err)
				os.Exit(1)
			}
			fmt.Println(snippet)
		}
	}
	return nil
}

func (gen *Generator) GenerateType(td *model.TypeDef) error {
	return nil
}

func (gen *Generator) GenerateResource(rez *model.ResourceDef) error {
	return nil
}

func (gen *Generator) GenerateOperation(op *model.OperationDef) error {
	return nil
}

func (gen *Generator) GenerateException(op *model.OperationOutput) error {
	return nil
}

func stringValue(s interface{}) string {
	if s == nil {
		return ""
	}
	return fmt.Sprint(s)
}

func (gen *Generator) EmitHttpTrace(op *model.OperationDef, example *model.OperationExample) (string, error) {
	body := "#\n# " + example.Title + "\n#\n"
	method := op.HttpMethod
	path := op.HttpUri
	bodyExample := ""
	headers := ""
	query := ""
	reqExample := data.AsObject(example.Input)
	for _, in := range op.Input.Fields {
		inName := string(in.Name)
		ex := reqExample.Get(string(inName))
		if in.HttpQuery != "" {
			sex := stringValue(ex)
			if sex != "" {
				query = query + "&" + inName + "=" + sex
			}
		} else if in.HttpPath {
			sex := stringValue(ex)
			if in.HttpPath {
				// urlEncode?
			}
			path = strings.Replace(path, "{" + inName + "}", sex, -1)
		} else if in.HttpHeader != "" {
			sex := stringValue(ex)
			headers = headers + in.HttpHeader + ": " + sex + "\n"
		} else { //in.HttpPayload
			bodyExample = data.Pretty(ex)
		}
	}
	if query != "" {
		query = "?" + query[1:]
	}
	path = path + query
	headers = headers + "Accept: application/json\n"
    if op.HttpMethod == "POST" || op.HttpMethod == "PUT" {
        headers = headers + "Content-Type: application/json; charset=utf-8\n"
		headers = headers + fmt.Sprintf("Content-Length: %d\n", len(bodyExample))
	}
	s := method + " " + path + " HTTP/1.1\n" + headers + "\n" + bodyExample
	body = body + s + "\n"

	out := op.Output
	exout := example.Output
	if example.Error != nil {
		out = gen.Schema.GetExceptionDef(example.Error.ShapeId)
		exout = example.Error.Output
		if out == nil {
			panic("error in example is not defined: " + example.Error.ShapeId)
		}
	}
	status := out.HttpStatus
	bodyExample = ""
	headers = "Content-Type: application/json; charset=utf-8\n"
	headers = headers + "Date: " + dateHeader() + "\n"
	respMessage := fmt.Sprintf("HTTP/1.1 %d %s\n", status, http.StatusText(int(status)))
	respExample := data.AsObject(exout)
	for _, o := range out.Fields {
		oName := string(o.Name)
		ex := respExample.Get(string(oName))
		if o.HttpHeader != "" {
			sex := stringValue(ex)
			headers = headers + o.HttpHeader + ": " + sex + "\n"
		} else { //body
			bodyExample = data.Pretty(ex)
		}
	}
	headers = fmt.Sprintf("Content-Length: %d\n", len(bodyExample)) + headers
	s = respMessage + headers + "\n" + bodyExample
	body = body + s + "\n"
	return body, nil
}

func dateHeader() string {
	t := time.Now()
	return t.Format("Mon, 2 Jan 2006 15:04:05 GMT")
}
