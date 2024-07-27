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
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/boynton/data"
)

func Parse(path string) (*Schema, error) {
	p, err := parseNoValidate(path)
	if err != nil {
		return nil, err
	}
	return p.Validate()
}

func (p *Parser) Validate() (*Schema, error) {
	//FIX ME
	return p.schema, nil
}

type Parser struct {
	path           string
	source         string
	scanner        *Scanner
	schema         *Schema
	lastToken      *Token
	prevLastToken  *Token
	ungottenToken  *Token
	currentComment string
}

func parseNoValidate(path string) (*Parser, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	src := string(b)
	p := &Parser{
		scanner: NewScanner(strings.NewReader(src)),
		path:    path,
		source:  src,
	}
	err = p.Parse()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Parser) CurrentComment() string {
	return p.currentComment
}

func (p *Parser) UngetToken() {
	p.ungottenToken = p.lastToken
	p.lastToken = p.prevLastToken
}

func (p *Parser) GetToken() *Token {
	if p.ungottenToken != nil {
		p.lastToken = p.ungottenToken
		p.ungottenToken = nil
		return p.lastToken
	}
	p.prevLastToken = p.lastToken
	tok := p.scanner.Scan()
	for {
		if tok.Type == EOF {
			return nil //fixme
		} else if tok.Type != BLOCK_COMMENT {
			break
		}
		tok = p.scanner.Scan()
	}
	p.lastToken = &tok
	return p.lastToken
}

func (p *Parser) Source() string {
	source := p.source
	if p.path != "" && source == "" {
		data, err := ioutil.ReadFile(p.path)
		if err == nil {
			source = string(data)
		}
	}
	return source
}

func (p *Parser) Parse() error {
	p.schema = NewSchema()
	comment := ""
	for {
		var err error
		tok := p.GetToken()
		if tok == nil {
			break
		}
		switch tok.Type {
		case SYMBOL:
			switch tok.Text {
			case "service":
				err = p.parseNameDirective(comment)
			case "namespace":
				err = p.parseNamespaceDirective(comment)
			case "version":
				err = p.parseVersionDirective(comment)
			case "resource":
				err = p.parseResourceDirective(comment)
			case "type":
				err = p.parseTypeDirective(comment)
				//			case "example":
				//				err = p.parseExampleDirective(comment)
			case "base":
				err = p.parseBaseDirective(comment)
			case "operation":
				err = p.parseOperation(comment)
			case "exception":
				err = p.parseException(comment)
			default:
				if strings.HasPrefix(tok.Text, "x_") {
					p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
					//p.schema.Annotations, comment, err = p.parseExtendedOptionTopLevel(p.schema.Annotations, tok.Text)
					//p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
				}
			}
			comment = ""
		case LINE_COMMENT:
			comment = p.MergeComment(comment, tok.Text)
		case SEMICOLON:
			/* ignore */
		case NEWLINE:
			/* ignore */
		default:
			return p.expectedDirectiveError()
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) parseNamespaceDirective(comment string) error {
	p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
	ns := ""
	txt, err := p.expectText()
	if err != nil {
		return err
	}
	ns = txt
	for {
		tok := p.GetToken()
		if tok == nil {
			break
		}
		if tok.Type != DOT {
			p.UngetToken()
			break
		}
		ns = ns + "."
		txt, err = p.expectText()
		if err != nil {
			return err
		}
		ns = ns + txt
	}
	p.schema.Namespace = Namespace(ns)
	p.schema.Id = AbsoluteIdentifier(string(p.schema.Namespace) + "#" + txt)
	return err
}

func (p *Parser) parseNameDirective(comment string) error {
	p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
	txt, err := p.expectText()
	if err == nil {
		p.schema.Id = AbsoluteIdentifier(string(p.schema.Namespace) + "#" + txt)
	}
	return err
}

func (p *Parser) parseVersionDirective(comment string) error {
	p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	switch tok.Type {
	case NUMBER, SYMBOL, STRING:
		p.schema.Version = tok.Text
		return nil
	default:
		return p.Error("Bad version value: " + tok.Text)
	}
}

func (p *Parser) parseBaseDirective(comment string) error {
	p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
	base, err := p.ExpectString()
	if err == nil {
		p.schema.Base = base
		if base != "" && !strings.HasPrefix(base, "/") {
			err = p.Error("Bad base path value: " + base)
		}
	}
	return err
}

func (p *Parser) addAnnotation(annos map[string]string, name, val string) map[string]string {
	if annos == nil {
		annos = make(map[string]string, 0)
	}
	annos[name] = val
	return annos
}

func (p *Parser) parseOperationInput(op *OperationDef, comment string) (*OperationInput, error) {
	input := &OperationInput{
		Id:      op.Id + "Input",
		Comment: comment,
	}
	options, err := p.ParseOptions("operation.input", []string{"name"})
	if err != nil {
		return nil, err
	}
	if options.Name != "" {
		input.Id = p.schema.Namespaced(options.Name)
	}
	tok := p.GetToken()
	if tok.Type != OPEN_BRACE {
		return nil, p.SyntaxError()
	}
	tok = p.GetToken()
	var prevIn *OperationInputField
	for tok != nil {
		if tok.Type == CLOSE_BRACE {
			return input, nil
		} else if tok.Type == NEWLINE {
			if prevIn != nil {
				prevIn = nil
				comment = ""
			}
			tok = p.GetToken()
			if tok == nil {
				return nil, p.EndOfFileError()
			}
			continue
		} else if tok.Type == LINE_COMMENT {
			if prevIn != nil {
				prevIn.Comment = p.MergeComment(prevIn.Comment, tok.Text)
			} else {
				comment = p.MergeComment(comment, tok.Text)
			}
		} else {
			in := &OperationInputField{
				Comment: comment,
			}
			prevIn = in
			if tok.Type != SYMBOL {
				return nil, p.SyntaxError()
			}
			in.Name = Identifier(tok.Text)
			tok = p.GetToken()
			if tok.Type != SYMBOL {
				return nil, p.SyntaxError()
			}
			in.Type = p.schema.Namespaced(tok.Text)
			options, err := p.ParseOptions("operation.input."+string(in.Name), []string{"path", "query", "header", "payload", "required", "default"})
			if err != nil {
				return nil, err
			}
			in.Default = options.Default
			in.Required = options.Required
			if options.Path {
				in.HttpPath = true
			} else if options.Query != "" {
				in.HttpQuery = Identifier(options.Query)
			} else if options.Header != "" {
				in.HttpHeader = options.Header
			} else if options.Payload {
				in.HttpPayload = true
			} else {
				return nil, p.Error("Input field must specified as 'path', 'query', 'header', or 'payload': " + string(in.Name))
			}
			input.Fields = append(input.Fields, in)
		}
		tok = p.GetToken()
	}
	return nil, nil
}

func (p *Parser) parseOperationOutput(op *OperationDef, comment string, isException bool) (*OperationOutput, error) {
	output := &OperationOutput{
		Comment: comment,
	}
	comment = ""
	//this is now an option that defaults to 200: (status=404)
	/*
		estatus, err := p.expectInt32()
		if err != nil {
			return nil, err
		}
		output.HttpStatus = estatus
	*/
	if isException {
		eName, err := p.ExpectIdentifier()
		if err != nil {
			return nil, err
		}
		output.Id = p.schema.Namespaced(eName)
		//output.Id = AbsoluteIdentifier(fmt.Sprintf("%sException%d", op.Id, estatus))
	} else {
		output.Id = op.Id + "Output"
	}
	elName := StripNamespace(output.Id)
	outOpts := []string{"status"}
	if !isException {
		outOpts = append(outOpts, "name")
	}
	options, err := p.ParseOptions(elName, outOpts)
	if err != nil {
		return nil, err
	}
	if options.Name != "" {
		output.Id = p.schema.Namespaced(options.Name)
		elName = StripNamespace(output.Id)
	}
	if options.Status == 0 {
		if isException {
			return nil, p.SyntaxError()
		}
		output.HttpStatus = 200
	} else {
		output.HttpStatus = options.Status
	}
	tok := p.GetToken()
	if tok.Type != OPEN_BRACE {
		return nil, p.SyntaxError()
	}
	tok = p.GetToken()
	var prevOut *OperationOutputField
	for tok != nil {
		if tok.Type == CLOSE_BRACE {
			if len(output.Fields) == 0 {
				output.Id = ""
			}
			return output, nil
		} else if tok.Type == NEWLINE {
			if prevOut != nil {
				prevOut = nil
				comment = ""
			}
			tok = p.GetToken()
			if tok == nil {
				return nil, p.EndOfFileError()
			}
			continue
		} else if tok.Type == LINE_COMMENT {
			if prevOut != nil {
				prevOut.Comment = p.MergeComment(prevOut.Comment, tok.Text)
			} else {
				comment = p.MergeComment(comment, tok.Text)
			}
		} else {
			out := &OperationOutputField{
				Comment: comment,
			}
			prevOut = out
			comment = ""
			if tok.Type != SYMBOL {
				return nil, p.SyntaxError()
			}
			out.Name = Identifier(tok.Text)
			tok = p.GetToken()
			if tok.Type != SYMBOL {
				return nil, p.SyntaxError()
			}
			out.Type = p.schema.Namespaced(tok.Text)
			options, err := p.ParseOptions(elName, []string{"header", "payload"})
			if err != nil {
				return nil, err
			}
			if options.Header != "" {
				out.HttpHeader = options.Header
			} else if options.Payload {
				out.HttpPayload = true
			} else {
				return nil, p.Error("Output field must be specified as 'header' or 'payload': " + string(out.Name))
			}
			output.Fields = append(output.Fields, out)
		}
		tok = p.GetToken()
	}
	return nil, nil
}

func (p *Parser) parseException(comment string) error {
	e, err := p.parseOperationOutput(nil, comment, true)
	if err == nil {
		p.schema.AddExceptionDef(e)
	}
	return err
}

func (p *Parser) getIdentifier() string {
	tok := p.GetToken()
	if tok == nil {
		return ""
	}
	if tok.Type == COMMA {
		//ignore the comma, try again
		return p.getIdentifier()
	}
	if tok.Type == SYMBOL {
		return tok.Text
	}
	p.UngetToken()
	return ""
}

func (p *Parser) parseOperation(comment string) error {
	name, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	options, err := p.ParseOptions("operation", []string{"method", "url"})
	if err != nil {
		return err
	}
	return p.finishOperation(name, options.Method, options.Url, comment)
}

func (p *Parser) finishOperation(name, method, pathTemplate, comment string) error {
	op := &OperationDef{
		Id:         p.schema.Namespaced(name),
		HttpMethod: method,
		HttpUri:    pathTemplate,
		//Annotations: options.Annotations,
	}
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type == OPEN_BRACE {
		var err error
		var done bool
		op.Comment = p.ParseTrailingComment(comment)
		comment = ""
		for {
			done, comment, err = p.IsBlockDone(comment)
			if err != nil {
				return err
			}
			if done {
				break
			}
			sym, err := p.ExpectIdentifier()
			if err != nil {
				return err
			}
			switch sym {
			case "input":
				if op.Input != nil {
					return p.SyntaxError()
				}
				op.Input, err = p.parseOperationInput(op, comment)
				if err != nil {
					return err
				}
			case "output":
				if op.Output != nil {
					return p.SyntaxError()
				}
				op.Output, err = p.parseOperationOutput(op, comment, false)
				if err != nil {
					return err
				}
			case "exceptions":
				lst, err := p.ExpectIdentifierArray()
				if err != nil {
					return err
				}
				for _, ename := range lst {
					eid := p.schema.Namespaced(ename)
					for _, e := range op.Exceptions {
						if e == eid {
							return p.Error("Duplicate Exception name: " + ename)
						}
					}
					op.Exceptions = append(op.Exceptions, eid)
				}
			case "exception":
				//change this.
				//exception, err := p.parseOperationOutput(op, comment, true)
				ename, err := p.ExpectIdentifier()
				if err != nil {
					return err
				}
				eid := p.schema.Namespaced(ename)
				for _, e := range op.Exceptions {
					if e == eid {
						return p.Error("Duplicate Exception name: " + ename)
					}
				}
				op.Exceptions = append(op.Exceptions, eid)
			default:
				return p.SyntaxError()
			}
			comment = ""
			if err != nil {
				return err
			}
		}
		op.Comment, err = p.EndOfStatement(op.Comment)
		if err != nil {
			return err
		}
		p.schema.Operations = append(p.schema.Operations, op)
	} else {
		return p.SyntaxError()
	}
	return nil
}

func (p *Parser) IsBlockDone(comment string) (bool, string, error) {
	tok := p.GetToken()
	if tok == nil {
		return false, comment, p.EndOfFileError()
	}
	for {
		if tok.Type == CLOSE_BRACE {
			return true, comment, nil
		} else if tok.Type == LINE_COMMENT {
			comment = p.MergeComment(comment, tok.Text)
			tok = p.GetToken()
			if tok == nil {
				return false, comment, p.EndOfFileError()
			}
		} else if tok.Type == NEWLINE {
			tok = p.GetToken()
			if tok == nil {
				return false, comment, p.EndOfFileError()
			}
		} else {
			p.UngetToken()
			return false, comment, nil
		}
	}
}

func (p *Parser) parseResourceDirective(comment string) error {
	rezName, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	rd := &ResourceDef{
		Id:      p.schema.Namespaced(rezName),
		Comment: comment,
	}
	tok := p.GetToken()
	if tok.Type != OPEN_BRACE {
		return p.SyntaxError()
	}
	rd.Comment = p.ParseTrailingComment(rd.Comment)
	tok = p.GetToken()
	if tok == nil {
		return p.SyntaxError()
	}
	if tok.Type != NEWLINE {
		p.UngetToken()
	}
	tok = p.GetToken()
	for tok != nil {
		if tok.Type == CLOSE_BRACE {
			p.schema.Resources = append(p.schema.Resources, rd)
			return nil
		} else if tok.Type == NEWLINE {
			tok = p.GetToken()
			if tok == nil {
				return p.EndOfFileError()
			}
			continue
		} else if tok.Type == LINE_COMMENT {
			//no comments on resource "members", which in smithy are not really members
		} else {
			if tok.Type != SYMBOL {
				return p.SyntaxError()
			}
			switch tok.Text {
			case "create":
				rd.Create, err = p.expectIdentifierAndMakeAbsolute()
			case "read":
				rd.Read, err = p.expectIdentifierAndMakeAbsolute()
			case "update":
				rd.Update, err = p.expectIdentifierAndMakeAbsolute()
			case "delete":
				rd.Delete, err = p.expectIdentifierAndMakeAbsolute()
			case "list":
				rd.List, err = p.expectIdentifierAndMakeAbsolute()
				//case "put": //smithy
			case "operations":
				rd.Operations, err = p.expectIdentifierListAndMakeAbsolute()
			case "collectionOperations":
				rd.CollectionOperations, err = p.expectIdentifierListAndMakeAbsolute()
			case "resources":
				rd.Resources, err = p.expectIdentifierListAndMakeAbsolute()
			default:
				return p.SyntaxError()
			}
			if err != nil {
				return err
			}
		}
		tok = p.GetToken()
	}
	return p.EndOfFileError()
}

func (p *Parser) parseTypeDirective(comment string) error {
	typeName, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	base, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	td := &TypeDef{
		Id:      p.schema.Namespaced(typeName),
		Comment: comment,
	}
	switch base {
	case "Struct":
		err = p.parseStructDef(td)
	case "Union":
		err = p.parseUnionDef(td)
	case "Map":
		err = p.parseMapDef(td)
	case "List":
		err = p.parseListDef(td)
	case "String":
		err = p.parseStringDef(td)
	case "Blob":
		err = p.parseBlobDef(td)
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal", "Integer":
		err = p.parseNumberDef(td, base)
	case "Enum":
		err = p.parseEnumDef(td)
	case "Bool":
		err = p.parseSimpleDef(td, BaseType_Bool)
	case "Timestamp":
		err = p.parseSimpleDef(td, BaseType_Timestamp)
	//? case "Any":
	default:
		return p.Error("Base type NYI: " + base)
	}
	if err != nil {
		return err
	}
	p.schema.Types = append(p.schema.Types, td)
	return nil
}

func (p *Parser) parseSimpleDef(td *TypeDef, base BaseType) error {
	td.Base = base
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Comment, err = p.EndOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseBlobDef(td *TypeDef) error {
	td.Base = BaseType_Blob
	err := p.parseTypeOptions(td, "minsize", "maxsize")
	if err == nil {
		td.Comment, err = p.EndOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseNumberDef(td *TypeDef, ntype string) error {
	switch ntype {
	case "Int8":
		td.Base = BaseType_Int8
	case "Int16":
		td.Base = BaseType_Int16
	case "Int32":
		td.Base = BaseType_Int32
	case "Int64":
		td.Base = BaseType_Int64
	case "Float32":
		td.Base = BaseType_Float32
	case "Float64":
		td.Base = BaseType_Float64
	case "Decimal":
		td.Base = BaseType_Decimal
	case "Integer":
		td.Base = BaseType_Integer
	}
	err := p.parseTypeOptions(td, "minvalue", "maxvalue")
	if err == nil {
		td.Comment, err = p.EndOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseStructDef(td *TypeDef) error {
	td.Base = BaseType_Struct
	err := p.parseTypeOptions(td)
	if err != nil {
		return err
	}
	tok := p.GetToken()
	if tok.Type != OPEN_BRACE {
		return p.SyntaxError()
	}
	td.Comment = p.ParseTrailingComment(td.Comment)
	tok = p.GetToken()
	if tok == nil {
		return p.SyntaxError()
	}
	if tok.Type != NEWLINE {
		p.UngetToken()
	}
	err = p.parseFields(td, []string{"required"})
	return err
}

func (p *Parser) parseUnionDef(td *TypeDef) error {
	td.Base = BaseType_Union
	err := p.parseTypeOptions(td)
	if err != nil {
		return err
	}
	tok := p.GetToken()
	if tok.Type != OPEN_BRACE {
		return p.SyntaxError()
	}
	td.Comment = p.ParseTrailingComment(td.Comment)
	tok = p.GetToken()
	if tok == nil {
		return p.SyntaxError()
	}
	if tok.Type != NEWLINE {
		p.UngetToken()
	}
	err = p.parseFields(td, []string{})
	return err
}

func (p *Parser) parseListDef(td *TypeDef) error {
	td.Base = BaseType_List
	err := p.expect(OPEN_BRACKET)
	if err != nil {
		return err
	}
	id, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	td.Items = p.schema.Namespaced(id)
	err = p.expect(CLOSE_BRACKET)
	if err != nil {
		return err
	}
	err = p.parseTypeOptions(td, "minsize", "maxsize")
	if err != nil {
		return err
	}
	td.Comment = p.ParseTrailingComment(td.Comment)
	return nil
}

func (p *Parser) parseMapDef(td *TypeDef) error {
	td.Base = BaseType_Map
	err := p.expect(OPEN_BRACKET)
	if err != nil {
		return err
	}
	id, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	td.Keys = p.schema.Namespaced(id)
	err = p.expect(COMMA)
	if err != nil {
		return err
	}
	id, err = p.ExpectIdentifier()
	if err != nil {
		return err
	}
	td.Items = p.schema.Namespaced(id)
	err = p.expect(CLOSE_BRACKET)
	if err != nil {
		return err
	}
	err = p.parseTypeOptions(td, "minsize", "maxsize")
	if err != nil {
		return err
	}
	td.Comment = p.ParseTrailingComment(td.Comment)
	return nil
}

func (p *Parser) parseStringDef(td *TypeDef) error {
	td.Base = BaseType_String
	err := p.parseTypeOptions(td, "minsize", "maxsize", "pattern")
	if err != nil {
		return err
	}
	return err
}

func (p *Parser) Error(msg string) error {
	return fmt.Errorf("*** %s\n", FormattedAnnotation(p.path, p.Source(), "", msg, p.lastToken, RED, 5))
}

func (p *Parser) SyntaxError() error {
	return p.Error("Syntax error")
}

func (p *Parser) EndOfFileError() error {
	return p.Error("Unexpected end of file")
}

func (p *Parser) assertIdentifier(tok *Token) (string, error) {
	if tok == nil {
		return "", p.EndOfFileError()
	}
	if tok.Type == SYMBOL {
		return tok.Text, nil
	}
	return tok.Text, p.Error(fmt.Sprintf("Expected symbol, found %v", tok.Type))
}

func (p *Parser) ExpectIdentifier() (string, error) {
	tok := p.GetToken()
	return p.assertIdentifier(tok)
}

func (p *Parser) ExpectIdentifierArray() ([]string, error) {
	tok := p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	if tok.Type != OPEN_BRACKET {
		return nil, p.SyntaxError()
	}
	var items []string
	for {
		tok := p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == CLOSE_BRACKET {
			break
		}
		if tok.Type == SYMBOL {
			items = append(items, tok.Text)
		} else if tok.Type == COMMA || tok.Type == NEWLINE || tok.Type == LINE_COMMENT {
			//ignore
		} else {
			return nil, p.SyntaxError()
		}
	}
	return items, nil
}

func (p *Parser) ExpectCompoundIdentifier() (string, error) {
	tok := p.GetToken()
	s, err := p.assertIdentifier(tok)
	if err != nil {
		return s, err
	}
	tok = p.GetToken()
	if tok == nil {
		return s, nil
	}
	if tok.Type != DOT {
		p.UngetToken()
		return s, nil
	}
	ss, err := p.ExpectCompoundIdentifier()
	if err != nil {
		return "", err
	}
	return s + "." + ss, nil
}

func (p *Parser) expectEqualsIdentifier() (string, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return "", err
	}
	return p.ExpectIdentifier()
}

func (p *Parser) expectEqualsCompoundIdentifier() (string, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return "", err
	}
	return p.ExpectCompoundIdentifier()
}

func (p *Parser) assertString(tok *Token) (string, error) {
	if tok == nil {
		return "", p.EndOfFileError()
	}
	if tok.Type == STRING {
		return tok.Text, nil
	}
	return tok.Text, p.Error(fmt.Sprintf("Expected string, found %v", tok.Type))
}

func (p *Parser) ExpectString() (string, error) {
	tok := p.GetToken()
	return p.assertString(tok)
}

func (p *Parser) expectEqualsString() (string, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return "", err
	}
	return p.ExpectString()
}

func (p *Parser) expectIdentifierAndMakeAbsolute() (AbsoluteIdentifier, error) {
	s, err := p.ExpectIdentifier()
	if err != nil {
		return "", err
	}
	return p.schema.Namespaced(s), nil
}

func (p *Parser) expectIdentifierListAndMakeAbsolute() ([]AbsoluteIdentifier, error) {
	lst, err := p.ExpectIdentifierList()
	if err != nil {
		return nil, err
	}
	var result []AbsoluteIdentifier
	for _, s := range lst {
		result = append(result, p.schema.Namespaced(s))
	}
	return result, nil
}

func (p *Parser) ExpectIdentifierList() ([]string, error) {
	var lst []string
	err := p.expect(OPEN_BRACKET)
	if err != nil {
		return nil, err
	}
	for {
		tok := p.GetToken()
		if tok == nil {
			return nil, p.SyntaxError()
		}
		if tok.Type == CLOSE_BRACKET {
			return lst, nil
		}
		if tok.Type != SYMBOL {
			//		s, err := p.ExpectIdentifier()
			//		if err != nil {
			//			return nil, err
			return nil, p.SyntaxError()
		}
		lst = append(lst, tok.Text)
	}
}

func (p *Parser) expectText() (string, error) {
	tok := p.GetToken()
	if tok == nil {
		return "", fmt.Errorf("Unexpected end of file")
	}
	if tok.IsText() {
		return tok.Text, nil
	}
	return "", fmt.Errorf("Expected symbol or string, found %v", tok.Type)
}

func (p *Parser) expectInt32() (int32, error) {
	tok := p.GetToken()
	if tok == nil {
		return 0, p.EndOfFileError()
	}
	if tok.IsNumeric() {
		l, err := strconv.ParseInt(tok.Text, 10, 32)
		return int32(l), err
	}
	return 0, p.Error(fmt.Sprintf("Expected number, found %v", tok.Type))
}

func (p *Parser) expectEqualsInt32() (*int32, error) {
	var val int32
	err := p.expect(EQUALS)
	if err != nil {
		return nil, err
	}
	val, err = p.expectInt32()
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (p *Parser) expectInt64() (int64, error) {
	tok := p.GetToken()
	if tok == nil {
		return 0, p.EndOfFileError()
	}
	if tok.IsNumeric() {
		l, err := strconv.ParseInt(tok.Text, 10, 64)
		return int64(l), err
	}
	return 0, p.Error(fmt.Sprintf("Expected number, found %v", tok.Type))
}

func (p *Parser) expectEqualsInt64() (int64, error) {
	var val int64
	err := p.expect(EQUALS)
	if err != nil {
		return 0, err
	}
	val, err = p.expectInt64()
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (p *Parser) expectNumber() (*data.Decimal, error) {
	tok := p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	if tok.IsNumeric() {
		return data.DecimalFromString(tok.Text)
	}
	return nil, p.Error(fmt.Sprintf("Expected number, found %v", tok.Type))
}

func (p *Parser) expectEqualsNumber() (*data.Decimal, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return nil, err
	}
	return p.expectNumber()
}

func (p *Parser) expect(toktype TokenType) error {
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type == toktype {
		return nil
	}
	return p.Error(fmt.Sprintf("Expected %v, found %v", toktype, tok.Type))
}

func containsOption(options []string, option string) bool {
	if options != nil {
		for _, opt := range options {
			if opt == option {
				return true
			}
		}
	}
	return false
}

func (p *Parser) parseTypeOptions(td *TypeDef, acceptable ...string) error {
	options, err := p.ParseOptions(td.Base.String(), acceptable)
	if err == nil {
		td.Pattern = options.Pattern
		td.MinSize = options.MinSize
		td.MaxSize = options.MaxSize
		td.MinValue = options.MinValue
		td.MaxValue = options.MaxValue
		//td.Annotations = options.Annotations
	}
	return err
}

type Options struct {
	Required  bool
	Path      bool
	Query     string
	Payload   bool
	Default   interface{}
	Pattern   string
	Value     string
	Url       string
	MinSize   int64
	MaxSize   int64
	MinValue  *data.Decimal
	MaxValue  *data.Decimal
	Action    string
	Header    string
	Name      string
	Method    string
	Resource  string
	Lifecycle string
	Status    int32
	//Annotations map[string]string
}

func (p *Parser) ParseOptions(typeName string, acceptable []string) (*Options, error) {
	options := &Options{}
	var err error
	tok := p.GetToken()
	if tok == nil {
		return options, nil
	}
	if tok.Type == OPEN_PAREN {
		for {
			tok := p.GetToken()
			if tok == nil {
				return nil, p.SyntaxError()
			}
			if tok.Type == CLOSE_PAREN {
				return options, nil
			}
			if tok.Type == SYMBOL {
				match := strings.ToLower(tok.Text)
				if strings.HasPrefix(match, "x_") {
					//options.Annotations, err = p.parseExtendedOption(options.Annotations, tok.Text)
				} else if containsOption(acceptable, match) {
					switch match {
					case "min":
						options.MinValue, err = p.expectEqualsNumber()
					case "max":
						options.MaxValue, err = p.expectEqualsNumber()
					case "minsize":
						options.MinSize, err = p.expectEqualsInt64()
					case "maxsize":
						options.MaxSize, err = p.expectEqualsInt64()
					case "pattern":
						options.Pattern, err = p.expectEqualsString()
					case "value":
						options.Value, err = p.expectEqualsString()
					case "url":
						options.Url, err = p.expectEqualsString()
					case "required":
						options.Required = true
					case "payload":
						options.Payload = true
					case "path":
						options.Path = true
					case "default":
						options.Default, err = p.parseEqualsLiteral()
					case "method":
						options.Method, err = p.expectEqualsIdentifier()
					case "action", "operation":
						options.Action, err = p.expectEqualsIdentifier()
					case "header":
						options.Header, err = p.expectEqualsString()
					case "query":
						options.Query, err = p.expectEqualsString()
					case "name":
						options.Name, err = p.expectEqualsIdentifier()
					case "resource":
						options.Resource, err = p.expectEqualsCompoundIdentifier()
					case "lifecycle":
						options.Lifecycle, err = p.expectEqualsCompoundIdentifier()
					case "status":
						pStatus, e := p.expectEqualsInt32()
						if e == nil {
							options.Status = *pStatus
						}
						err = e
					default:
						err = p.Error("Unrecognized option: " + tok.Text)
					}
				} else {
					err = p.Error(fmt.Sprintf("Unrecognized option for %s: %s", typeName, tok.Text))
				}
				if err != nil {
					return nil, err
				}
			} else if tok.Type == COMMA {
				//ignore
			} else {
				return nil, p.SyntaxError()
			}
		}
	} else {
		p.UngetToken()
		return options, nil
	}
}

// parse the next string. And also a line comment, and the end of line, if present. Anything else is an error
func (p *Parser) parseStringToEndOfLine() (string, string, error) {
	val := ""
	comment := ""
	tok := p.GetToken()
	if tok == nil {
		return val, comment, nil
	}
	if tok.Type == EQUALS {
		//ignore it except error if at end of file
		tok = p.GetToken()
		if tok == nil {
			return "", "", p.EndOfFileError()
		}
	}
	if tok.Type == STRING {
		val = tok.Text
		tok = p.GetToken()
	}
	if tok == nil {
		return val, comment, nil
	}
	if tok.Type == LINE_COMMENT {
		comment = tok.Text
		tok = p.GetToken()
	}
	if tok == nil {
		return val, comment, nil
	}
	if tok.Type != NEWLINE {
		return "", "", p.SyntaxError()
	}
	return val, comment, nil
}

func (p *Parser) parseExtendedOptionTopLevel(annos map[string]string, anno string) (map[string]string, string, error) {
	val, comment, err := p.parseStringToEndOfLine()
	if annos == nil {
		annos = make(map[string]string, 0)
	}
	annos[anno] = val
	return annos, comment, err
}

func (p *Parser) parseExtendedOption(annos map[string]string, anno string) (map[string]string, error) {
	var err error
	var val string
	tok := p.GetToken()
	if tok != nil {
		if tok.Type == EQUALS {
			val, err = p.ExpectString()
		} else {
			p.UngetToken()
		}
	} else {
		err = p.EndOfFileError()
	}
	if err != nil {
		return annos, err
	}
	if annos == nil {
		annos = make(map[string]string, 0)
	}
	annos[anno] = val
	return annos, err
}

func (p *Parser) parseBytesOptions(typedef *TypeDef) error {
	tok := p.GetToken()
	if tok == nil {
		return p.SyntaxError()
	}
	expected := ""
	if tok.Type == OPEN_PAREN {
		for {
			tok := p.GetToken()
			if tok == nil {
				return p.SyntaxError()
			}
			if tok.Type == CLOSE_PAREN {
				return nil
			}
			if tok.Type == SYMBOL {
				switch tok.Text {
				case "minsize", "maxsize":
					expected = tok.Text
				default:
					return p.Error("invalid bytes option: " + tok.Text)
				}
			} else if tok.Type == EQUALS {
				if expected == "" {
					return p.SyntaxError()
				}
			} else if tok.Type == NUMBER {
				if expected == "" {
					return p.SyntaxError()
				}
				val, err := data.DecimalFromString(tok.Text)
				if err != nil {
					return err
				}
				if expected == "minsize" {
					i := val.AsInt64()
					typedef.MinSize = i
				} else if expected == "maxsize" {
					i := val.AsInt64()
					typedef.MinSize = i
				} else {
					return p.Error("bytes option must have numeric value")
				}
				expected = ""
			}
		}
	} else {
		p.UngetToken()
		return nil
	}
}

func (p *Parser) expectEqualsStringArray() ([]string, error) {
	var values []string
	tok := p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	if tok.Type != EQUALS {
		return nil, p.SyntaxError()
	}

	tok = p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	if tok.Type != OPEN_BRACKET {
		return nil, p.SyntaxError()
	}
	for {
		tok = p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == CLOSE_BRACKET {
			break
		}
		if tok.Type == STRING {
			values = append(values, tok.Text)
		} else if tok.Type == COMMA || tok.Type == NEWLINE {
			//ignore
		} else {
			return nil, p.SyntaxError()
		}
	}
	return values, nil
}

func (p *Parser) parseEnumDef(td *TypeDef) error {
	td.Base = BaseType_Enum
	tok := p.GetToken()
	if tok.Type != OPEN_BRACE {
		return p.SyntaxError()
	}
	td.Comment = p.ParseTrailingComment(td.Comment)
	tok = p.GetToken()
	if tok == nil {
		return p.SyntaxError()
	}
	if tok.Type != NEWLINE {
		p.UngetToken()
	}
	el, err := p.parseEnumElement()
	for el != nil {
		td.Elements = append(td.Elements, el)
		el, err = p.parseEnumElement()
	}
	return err
}

func (p *Parser) parseEnumElement() (*EnumElement, error) {
	comment := ""
	sym := ""
	var err error
	for {
		tok := p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == CLOSE_BRACE {
			return nil, nil
		} else if tok.Type == LINE_COMMENT {
			comment = p.MergeComment(comment, tok.Text)
		} else if tok.Type == SEMICOLON || tok.Type == NEWLINE || tok.Type == COMMA {
			//ignore
		} else {
			sym, err = p.assertIdentifier(tok)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	options, err := p.ParseOptions("Enum", []string{"value"})
	if err != nil {
		return nil, err
	}
	comment = p.ParseTrailingComment(comment)

	return &EnumElement{
		Symbol: Identifier(sym),
		Value:  options.Value,
		//Type: etype,
		Comment: comment,
		//Annotations: options.Annotations,
	}, nil
}

func (p *Parser) expectNewline() error {
	tok := p.GetToken()
	if tok == nil {
		return p.SyntaxError()
	}
	if tok.Type != NEWLINE {
		p.UngetToken()
		return p.SyntaxError()
	}
	return nil
}

func (p *Parser) parseFields(td *TypeDef, fieldOptions []string) error {
	//already parsed the open brace
	comment := ""
	tok := p.GetToken()
	var prev *FieldDef
	for tok != nil {
		if tok.Type == CLOSE_BRACE {
			return nil
		} else if tok.Type == NEWLINE {
			if prev != nil {
				prev = nil
				comment = ""
			}
			tok = p.GetToken()
			if tok == nil {
				return p.EndOfFileError()
			}
			continue
		} else if tok.Type == LINE_COMMENT {
			if prev != nil {
				prev.Comment = p.MergeComment(prev.Comment, tok.Text)
			} else {
				comment = p.MergeComment(comment, tok.Text)
			}
		} else {
			fd := &FieldDef{
				Comment: comment,
			}
			prev = fd
			if tok.Type != SYMBOL {
				return p.SyntaxError()
			}
			fd.Name = Identifier(tok.Text)
			tok = p.GetToken()
			if tok.Type != SYMBOL {
				return p.SyntaxError()
			}
			fd.Type = p.schema.Namespaced(tok.Text)
			options, err := p.ParseOptions(string(td.Id)+"."+string(fd.Name), fieldOptions)
			if err != nil {
				return err
			}
			fd.Required = options.Required
			fd.Comment, err = p.EndOfStatement(fd.Comment)
			if err != nil {
				return err
			}
			td.Fields = append(td.Fields, fd)
		}
		tok = p.GetToken()
	}
	return nil
}

/*

func (p *Parser) parseStructFieldOptions(field *StructFieldDef) error {
	var acceptable []string
	switch field.Type {
	case "String":
		acceptable = []string{"pattern", "values", "minsize", "maxsize", "reference"}
	case "UUID":
		acceptable = []string{"reference"}
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		acceptable = []string{"min", "max"}
	case "Bytes", "Array", "Map":
		acceptable = []string{"minsize", "maxsize"}
	}
	acceptable = append(acceptable, "required")
	acceptable = append(acceptable, "default")
	options, err := p.ParseOptions(field.Type, acceptable)
	if err == nil {
		field.Required = options.Required
		field.Default = options.Default
		field.Pattern = options.Pattern
		field.Values = options.Values
		field.MinSize = options.MinSize
		field.MaxSize = options.MaxSize
		field.Min = options.Min
		field.Max = options.Max
		field.Annotations = options.Annotations
		field.Reference = options.Reference
	}
	return err
}
*/

func (p *Parser) parseEqualsLiteral() (interface{}, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return 0, err
	}
	return p.parseLiteralValue()
}

func (p *Parser) parseLiteralValue() (interface{}, error) {
	tok := p.GetToken()
	if tok == nil {
		return nil, p.SyntaxError()
	}
	return p.parseLiteral(tok)
}

func (p *Parser) parseLiteral(tok *Token) (interface{}, error) {
	switch tok.Type {
	case SYMBOL:
		return p.parseLiteralSymbol(tok)
	case STRING:
		return p.parseLiteralString(tok)
	case NUMBER:
		return p.parseLiteralNumber(tok)
	case OPEN_BRACKET:
		return p.parseLiteralArray()
	case OPEN_BRACE:
		return p.parseLiteralObject()
	default:
		return nil, p.SyntaxError()
	}
}

func (p *Parser) parseLiteralSymbol(tok *Token) (interface{}, error) {
	switch tok.Text {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	default:
		return nil, fmt.Errorf("Not a valid symbol: %s", tok.Text)
	}
}
func (p *Parser) parseLiteralString(tok *Token) (*string, error) {
	s := "\"" + tok.Text + "\""
	q, err := strconv.Unquote(s)
	if err != nil {
		return nil, p.Error("Improperly escaped string: " + tok.Text)
	}
	return &q, nil
}

func (p *Parser) parseLiteralNumber(tok *Token) (interface{}, error) {
	num, err := data.DecimalFromString(tok.Text)
	if err != nil {
		return nil, p.Error(fmt.Sprintf("Not a valid number: %s", tok.Text))
	}
	return num, nil
}

func (p *Parser) parseLiteralArray() (interface{}, error) {
	var ary []interface{}
	for {
		tok := p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type != NEWLINE {
			if tok.Type == CLOSE_BRACKET {
				return ary, nil
			}
			if tok.Type != COMMA {
				obj, err := p.parseLiteral(tok)
				if err != nil {
					return nil, err
				}
				ary = append(ary, obj)
			}
		}
	}
}

func (p *Parser) parseLiteralObject() (interface{}, error) {
	// a JSON object
	obj := make(map[string]interface{}, 0)
	for {
		tok := p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == CLOSE_BRACE {
			return obj, nil
		}
		if tok.Type == STRING {
			pkey, err := p.parseLiteralString(tok)
			if err != nil {
				return nil, err
			}
			err = p.expect(COLON)
			if err != nil {
				return nil, err
			}
			val, err := p.parseLiteralValue()
			if err != nil {
				return nil, err
			}
			obj[*pkey] = val
		} else if tok.Type == SYMBOL {
			return nil, p.Error("Expected String key for JSON object, found symbol '" + tok.Text + "'")
		} else {
			//fmt.Println("ignoring this token:", tok)
		}
	}
}

func (p *Parser) arrayParams(params []string) (string, error) {
	var items string
	switch len(params) {
	case 0:
		items = "Any"
	case 1:
		items = params[0]
	default:
		return "", p.SyntaxError()
	}
	return items, nil
}

func (p *Parser) parseCollectionOptions(typedef *TypeDef) error {
	tok := p.GetToken()
	if tok == nil {
		return p.SyntaxError()
	}
	if tok.Type == OPEN_PAREN {
		for {
			tok := p.GetToken()
			if tok == nil {
				return p.SyntaxError()
			}
			if tok.Type == CLOSE_PAREN {
				return nil
			}
			if tok.Type == SYMBOL {
				switch tok.Text {
				case "minsize":
					num, err := p.expectEqualsInt64()
					if err != nil {
						return err
					}
					typedef.MinSize = num
				case "maxsize":
					num, err := p.expectEqualsInt64()
					if err != nil {
						return err
					}
					typedef.MaxSize = num
				}
			} else {
				return p.SyntaxError()
			}
		}
	} else {
		p.UngetToken()
		return nil
	}
}

func (p *Parser) mapParams(params []string) (string, string, error) {
	var keys string
	var items string
	switch len(params) {
	case 0:
		keys = "String"
		items = "Any"
	case 2:
		keys = params[0]
		items = params[1]
	default:
		return "", "", p.SyntaxError()
	}
	return keys, items, nil
}

func (p *Parser) unitValueParams(params []string) (string, string, error) {
	var value string
	var unit string
	var err error
	switch len(params) {
	case 0:
		value = "Decimal"
		unit = "String"
	case 2:
		value = params[0]
		unit = params[1]
	default:
		err = p.SyntaxError()
	}
	return value, unit, err
}

func (p *Parser) EndOfStatement(comment string) (string, error) {
	for {
		tok := p.GetToken()
		if tok == nil {
			return comment, nil
		}
		if tok.Type == SEMICOLON {
			//ignore it
		} else if tok.Type == LINE_COMMENT {
			comment = p.MergeComment(comment, tok.Text)
		} else if tok.Type == NEWLINE {
			return comment, nil
		} else {
			return comment, p.SyntaxError()
		}
	}
}

func (p *Parser) parseLeadingComment(comment string) string {
	for {
		tok := p.GetToken()
		if tok == nil {
			return comment
		}
		if tok.Type == LINE_COMMENT {
			comment = p.MergeComment(comment, tok.Text)
		} else {
			p.UngetToken()
			return comment
		}
	}
}

func (p *Parser) ParseTrailingComment(comment string) string {
	tok := p.GetToken()
	if tok != nil {
		if tok.Type == LINE_COMMENT {
			comment = p.MergeComment(comment, tok.Text)
		} else {
			p.UngetToken()
		}
	}
	return comment
}

func (p *Parser) MergeComment(comment1 string, comment2 string) string {
	comment1 = strings.TrimSpace(comment1)
	comment2 = strings.TrimSpace(comment2)
	if comment1 == "" {
		return comment2
	}
	if comment2 == "" {
		return comment1
	}
	return comment1 + " " + comment2
}

func (p *Parser) expectedDirectiveError() error {
	msg := "Expected one of 'type', 'operation', 'exception', 'namespace', 'name', 'version', 'base'"
	msg = msg + " or an 'x_*' style extended annotation"
	return p.Error(msg)
}
