/* Generated */

package model

import (
	"encoding/json"
	"fmt"
	"github.com/boynton/data"
)

// BaseType - All other types are derived from these.
type BaseType int

const (
	_ BaseType = iota
	BaseType_Bool
	BaseType_Int8
	BaseType_Int16
	BaseType_Int32
	BaseType_Int64
	BaseType_Float32
	BaseType_Float64
	BaseType_Integer
	BaseType_Decimal
	BaseType_Blob
	BaseType_String
	BaseType_Timestamp
	BaseType_List
	BaseType_Map
	BaseType_Struct
	BaseType_Enum
	BaseType_Union
	BaseType_Any
)

var namesBaseType = []string{
	BaseType_Bool:      "Bool",
	BaseType_Int8:      "Int8",
	BaseType_Int16:     "Int16",
	BaseType_Int32:     "Int32",
	BaseType_Int64:     "Int64",
	BaseType_Float32:   "Float32",
	BaseType_Float64:   "Float64",
	BaseType_Integer:   "Integer",
	BaseType_Decimal:   "Decimal",
	BaseType_Blob:      "Blob",
	BaseType_String:    "String",
	BaseType_Timestamp: "Timestamp",
	BaseType_List:      "List",
	BaseType_Map:       "Map",
	BaseType_Struct:    "Struct",
	BaseType_Enum:      "Enum",
	BaseType_Union:     "Union",
	BaseType_Any:       "Any",
}

func (e BaseType) String() string {
	return namesBaseType[e]
}
func (e BaseType) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}
func (e *BaseType) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err == nil {
		for v, s2 := range namesBaseType {
			if s == s2 {
				*e = BaseType(v)
				return nil
			}
		}
		err = fmt.Errorf("Bad enum symbol for type BaseType: %s", s)
	}
	return err
}

// Identifier - a simple symbolic name that most programming languages can use,
// i.e. "Blah"
type Identifier string

// Namespace - A sequence of one or more names delimited by a '.', i.e.
// "foo.bar"
type Namespace string

// AbsoluteIdentifier - an Identifier in a Namespace, i.e. "foo.bar#Blah".
type AbsoluteIdentifier string

type StringList []string

type AbsoluteIdentifierList []AbsoluteIdentifier

type FieldDefList []*FieldDef

type EnumElementList []*EnumElement

// TypeDef - a structure defining a new type in this system. New types cannot be
// derived from these, but this new type can be used to specify the type of
// members in aggregate types. TypeDef could more properly be defined as a Union
// of various types, but this structure is more convenient.
type TypeDef struct {
	Comment  string             `json:"comment,omitempty"`
	Tags     StringList         `json:"tags,omitempty"`
	MinValue *data.Decimal      `json:"minValue,omitempty"`
	MaxValue *data.Decimal      `json:"maxValue,omitempty"`
	MinSize  int64              `json:"minSize,omitempty"`
	MaxSize  int64              `json:"maxSize,omitempty"`
	Required bool               `json:"required,omitempty"`
	Pattern  string             `json:"pattern,omitempty"`
	Items    AbsoluteIdentifier `json:"items,omitempty"`
	Keys     AbsoluteIdentifier `json:"keys,omitempty"`
	Fields   FieldDefList       `json:"fields,omitempty"`
	Elements EnumElementList    `json:"elements,omitempty"`
	Id       AbsoluteIdentifier `json:"id"`
	Base     BaseType           `json:"base"`
}

// Field - describes each field in a structure or union.
type FieldDef struct {
	Comment  string             `json:"comment,omitempty"`
	Tags     StringList         `json:"tags,omitempty"`
	MinValue *data.Decimal      `json:"minValue,omitempty"`
	MaxValue *data.Decimal      `json:"maxValue,omitempty"`
	MinSize  int64              `json:"minSize,omitempty"`
	MaxSize  int64              `json:"maxSize,omitempty"`
	Required bool               `json:"required,omitempty"`
	Pattern  string             `json:"pattern,omitempty"`
	Items    AbsoluteIdentifier `json:"items,omitempty"`
	Keys     AbsoluteIdentifier `json:"keys,omitempty"`
	Fields   FieldDefList       `json:"fields,omitempty"`
	Elements EnumElementList    `json:"elements,omitempty"`
	Name     Identifier         `json:"name"`
	Type     AbsoluteIdentifier `json:"type"`
}

// Element - describes each element of an Enum type
type EnumElement struct {
	Comment string     `json:"comment,omitempty"`
	Tags    StringList `json:"tags,omitempty"`
	Symbol  Identifier `json:"symbol"`
	Value   string     `json:"value,omitempty"`
}

// OperationDef - describes an operation, including its HTTP bindings
type OperationDef struct {
	Comment    string                 `json:"comment,omitempty"`
	Tags       StringList             `json:"tags,omitempty"`
	Id         AbsoluteIdentifier     `json:"id"`
	HttpMethod string                 `json:"httpMethod,omitempty"`
	HttpUri    string                 `json:"httpUri,omitempty"`
	Input      *OperationInput        `json:"input,omitempty"`
	Output     *OperationOutput       `json:"output,omitempty"`
	Exceptions AbsoluteIdentifierList `json:"exceptions,omitempty"`
	Resource   string                 `json:"resource,omitempty"`
	Lifecycle  string                 `json:"lifecycle,omitempty"`
	Examples   OperationExampleList   `json:"examples,omitempty"`
}

type OperationOutputList []*OperationOutput

type OperationExampleList []*OperationExample

type OperationExample struct {
	Title  string                 `json:"title,omitempty"`
	Input  any                    `json:"input,omitempty"`
	Output any                    `json:"output,omitempty"`
	Error  *OperationErrorExample `json:"error,omitempty"`
}

type OperationErrorExample struct {
	ShapeId AbsoluteIdentifier `json:"shapeId,omitempty"`
	Output  any                `json:"output,omitempty"`
}

// OperationInput - the description of an operation input. It is similar to a
// Struct definition, but with HTTP bindings.
type OperationInput struct {
	Comment string                  `json:"comment,omitempty"`
	Tags    StringList              `json:"tags,omitempty"`
	Id      AbsoluteIdentifier      `json:"id,omitempty"`
	Fields  OperationInputFieldList `json:"fields,omitempty"`
}

type OperationInputFieldList []*OperationInputField

// OperationInputField - the description of an operation input field
type OperationInputField struct {
	Comment     string             `json:"comment,omitempty"`
	Tags        StringList         `json:"tags,omitempty"`
	MinValue    *data.Decimal      `json:"minValue,omitempty"`
	MaxValue    *data.Decimal      `json:"maxValue,omitempty"`
	MinSize     int64              `json:"minSize,omitempty"`
	MaxSize     int64              `json:"maxSize,omitempty"`
	Required    bool               `json:"required,omitempty"`
	Pattern     string             `json:"pattern,omitempty"`
	Items       AbsoluteIdentifier `json:"items,omitempty"`
	Keys        AbsoluteIdentifier `json:"keys,omitempty"`
	Fields      FieldDefList       `json:"fields,omitempty"`
	Elements    EnumElementList    `json:"elements,omitempty"`
	Name        Identifier         `json:"name"`
	Type        AbsoluteIdentifier `json:"type"`
	Default     any                `json:"default,omitempty"`
	HttpHeader  string             `json:"httpHeader,omitempty"`
	HttpQuery   Identifier         `json:"httpQuery,omitempty"`
	HttpPath    bool               `json:"httpPath,omitempty"`
	HttpPayload bool               `json:"httpPayload,omitempty"`
}

// OperationOutput - the description of an operation output. Similar to a Struct
// definition, but with HTTP bindings. Also used for OperationExceptions.
type OperationOutput struct {
	Comment    string                   `json:"comment,omitempty"`
	Tags       StringList               `json:"tags,omitempty"`
	Id         AbsoluteIdentifier       `json:"id,omitempty"`
	HttpStatus int32                    `json:"httpStatus,omitempty"`
	Fields     OperationOutputFieldList `json:"fields,omitempty"`
}

type OperationOutputFieldList []*OperationOutputField

// OperationOutputField
type OperationOutputField struct {
	Comment     string             `json:"comment,omitempty"`
	Tags        StringList         `json:"tags,omitempty"`
	MinValue    *data.Decimal      `json:"minValue,omitempty"`
	MaxValue    *data.Decimal      `json:"maxValue,omitempty"`
	MinSize     int64              `json:"minSize,omitempty"`
	MaxSize     int64              `json:"maxSize,omitempty"`
	Required    bool               `json:"required,omitempty"`
	Pattern     string             `json:"pattern,omitempty"`
	Items       AbsoluteIdentifier `json:"items,omitempty"`
	Keys        AbsoluteIdentifier `json:"keys,omitempty"`
	Fields      FieldDefList       `json:"fields,omitempty"`
	Elements    EnumElementList    `json:"elements,omitempty"`
	Name        Identifier         `json:"name"`
	Type        AbsoluteIdentifier `json:"type"`
	HttpHeader  string             `json:"httpHeader,omitempty"`
	HttpPayload bool               `json:"httpPayload,omitempty"`
}

type TypeDefList []*TypeDef

type OperationDefList []*OperationDef

// ServiceDef - the definition of a service, consisting of Types and Operations
type ServiceDef struct {
	Comment    string              `json:"comment,omitempty"`
	Tags       StringList          `json:"tags,omitempty"`
	Id         AbsoluteIdentifier  `json:"id"`
	Version    string              `json:"version,omitempty"`
	Base       string              `json:"base,omitempty"`
	Types      TypeDefList         `json:"types,omitempty"`
	Operations OperationDefList    `json:"operations,omitempty"`
	Exceptions OperationOutputList `json:"exceptions,omitempty"`
}
