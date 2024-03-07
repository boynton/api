/*
Copyright 2022 Lee R. Boynton

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
)

func (schema *Schema) Validate() error {
	for _, op := range schema.Operations {
		err := schema.ValidateOperation(op)
		if err != nil {
			return err
		}
	}
	return nil
}

func (schema *Schema) ValidationError(context, msg string) error {
	return fmt.Errorf("*** Validation failure: " + context + ": " + msg)
}

func (schema *Schema) ValidationWarning(context, msg string) {
	Warning(context + ": " + msg)
}

func (schema *Schema) ValidateOperation(op *OperationDef) error {
	err := schema.ValidateOperationInput(op)
	if err != nil {
		return err
	}
	err = schema.ValidateOperationOutput(op, op.Output)
	if err != nil {
		return err
	}
	for _, eid := range op.Exceptions {
		e := schema.GetExceptionDef(eid)
		if e == nil {
			return fmt.Errorf("Exception not defined: %s", eid)
		}
		err = schema.ValidateOperationOutput(op, e)
		if err != nil {
			return err
		}
	}
	return nil
}

func (schema *Schema) ValidateOperationInput(op *OperationDef) error {
	if op.Input == nil {
		return nil
	}
	for _, in := range op.Input.Fields {
		if in.HttpPath {
			if !(in.HttpQuery != "" || in.HttpHeader != "" || in.HttpPayload) {
				continue
			}
		} else if in.HttpQuery != "" {
			if !(in.HttpPath || in.HttpHeader != "" || in.HttpPayload) {
				continue
			}
		} else if in.HttpHeader != "" {
			if !(in.HttpPath || in.HttpQuery != "" || in.HttpPayload) {
				continue
			}
		} else if in.HttpPayload {
			if !(in.HttpPath || in.HttpQuery != "" || in.HttpHeader != "") {
				continue
			}
		}
		context := StripNamespace(op.Id) + "$" + string(in.Name)
		return schema.ValidationError(context, "Input field should be specified as one of 'path', 'query', 'header', or 'payload'")
	}
	return nil
}

func (schema *Schema) ValidateOperationOutput(op *OperationDef, out *OperationOutput) error {
	if out == nil {
		panic("every operation must have an output")
	}
	for _, out := range out.Fields {
		if !schema.IsBaseType(out.Type) {
			td := schema.GetTypeDef(out.Type)
			if td == nil {
				return fmt.Errorf("Type not defined: %q in field %q of %q", out.Type, out.Name, op.Id)
			}
		}
		if out.HttpHeader != "" {
			if !out.HttpPayload {
				continue
			}
		} else if out.HttpPayload {
			if out.HttpHeader == "" {
				continue
			}
		}
		//errors with inlined fields as the payload are actually used in the wild.
		//it use to be: smithy openapi generation wopuld insert an XxxContent type to specify the
		//payload.
		context := StripNamespace(op.Id) + "$" + string(out.Name)
		schema.ValidationWarning(context, "Output field should be specified as one of 'header' or 'payload'\n")
	}
	return nil
}
