package data

import(
	"encoding/json"
	"fmt"
	"strconv"
)

type Type int

const (
	_ Type = iota
	Type_Null
	Type_Bool
	Type_Number
	Type_String
	Type_Array
	Type_Object
)

func (t Type) String() string {
	switch t {
	case Type_Null:
		return "Null"
	case Type_Bool:
		return "Bool"
	case Type_Number:
		return "Number"
	case Type_String:
		return "String"
	case Type_Array:
		return "Array"
	case Type_Object:
		return "Object"
	}
	return ""
}

type Value struct {
	dtype Type
	tval string
	aval []*Value
	mval *Map[*Value]
}

func (v *Value) DataType() Type {
	return v.dtype
}

func (v *Value) IsBool() bool {
	return v.dtype == Type_Bool
}

func (v *Value) AsBool() bool {
	if v.dtype == Type_Bool && v.tval == "true" {
		return true
	}
	return false
}

func (v *Value) IsNumber() bool {
	return v.dtype == Type_Number
}

func (v *Value) AsInt() int {
	if v.dtype == Type_Number {
		i, err := strconv.Atoi(v.tval)
		if err == nil {
			return i
		}
	}
	return 0
}

func (v *Value) AsInt64() int64 {
	if v.dtype == Type_Number {
		i, err := strconv.ParseInt(v.tval, 10, 64)
		if err == nil {
			return i
		}
	}
	return 0
}

func (v *Value) AsFloat64() float64 {
	if v.dtype == Type_Number {
		f, err := strconv.ParseFloat(v.tval, 64)
		if err == nil {
			return f
		}
	}
	return 0
}

func (v *Value) IsString() bool {
	return v.dtype == Type_String
}

func (v *Value) AsString() string {
	if v.dtype == Type_String {
		return v.tval
	}
	return ""
}

func (v *Value) IsArray() bool {
	return v.dtype == Type_Array
}

func (v *Value) AsSlice() []*Value {
	if v.dtype == Type_Array {
		return v.aval
	}
	return nil
}

func (v *Value) IsObject() bool {
	return v.dtype == Type_Object
}

func (v *Value) AsMap() *Map[*Value] {
	if v.dtype == Type_Object {
		return v.mval
	}
	return nil
}

func (v *Value) Keys() []string {
	if v.dtype != Type_Object {
		return nil
	}
	return v.mval.Keys()
}

func (v *Value) String() string {
	switch v.dtype {
	case Type_Null, Type_Bool, Type_Number:
		return v.tval
	case Type_String:
		return fmt.Sprintf("%q", v.tval)
	case Type_Array:
		return JsonEncode(v.aval)
	case Type_Object:
		return v.mval.String()
	default:
		return "?" //not possible
	}
}

func (v *Value) MarshalJSON() ([]byte, error) {
	switch v.dtype {
    case Type_Null, Type_Bool, Type_Number:
        return []byte(v.tval), nil
	case Type_String:
		return json.Marshal(v.tval)
	case Type_Array:
		return json.Marshal(v.aval)
	case Type_Object:
		return json.Marshal(v.mval)
	default:
		panic("can't happen")
	}
}

func (v *Value) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*v = *NewValue(raw)
	return nil
}

func (v *Value) Copy() *Value {
	vv, err := JsonDecode(JsonEncode(v))
	if err != nil {
		return nil
	}
	return vv
}

var Null = &Value{dtype: Type_Null, tval: "null"}

var False = &Value{dtype: Type_Bool, tval: "false"}
var True = &Value{dtype: Type_Bool, tval: "true"}

func NewNumber(repr string) *Value {
	return &Value{dtype: Type_Number, tval: repr}
}

func NewValue(raw any) *Value {
	if raw == nil {
		return Null
	}
	switch v := raw.(type) {
	case bool:
		if v {
			return True
		}
		return False
	case int:
		return &Value{dtype: Type_Number, tval: fmt.Sprintf("%d", v)}
	case int8:
		return &Value{dtype: Type_Number, tval: fmt.Sprintf("%d", v)}
	case int16:
		return &Value{dtype: Type_Number, tval: fmt.Sprintf("%d", v)}
	case int32:
		return &Value{dtype: Type_Number, tval: fmt.Sprintf("%d", v)}
	case int64:
		return &Value{dtype: Type_Number, tval: fmt.Sprintf("%d", v)}
	case float32:
		return &Value{dtype: Type_Number, tval: fmt.Sprintf("%g", v)}
	case float64:
		return &Value{dtype: Type_Number, tval: fmt.Sprintf("%g", v)}
	case string:
		return &Value{dtype: Type_String, tval: v}
	case *Decimal:
		return &Value{dtype: Type_String, tval: fmt.Sprintf("%v", v)}
	case Decimal:
		return &Value{dtype: Type_String, tval: fmt.Sprintf("%v", v)}
	case []any:
		return NewArrayFromSlice(v)
	case map[string]any:
		return NewObjectFromMap(v)
	case *Value:
		return v
	default:
		fmt.Println("what is this:", raw, "typeOf() ->", TypeOf(raw))
		panic("whoops")
	}
}

func (o *Value) Put(k string, v *Value) *Value {
	if o.dtype != Type_Object {
		return nil
	}
	o.mval.Put(k, v)
	return o
}

func (o *Value) PutString(k string, s string) *Value {
	return o.Put(k, NewValue(s))
}

func (o *Value) PutInt(k string, i int) *Value {
	return o.Put(k, NewValue(i))
}


func (o *Value) Length() int {
	switch o.dtype {
	case Type_Object:
		return o.mval.Length()
	case Type_Array:
		return len(o.aval)
	default:
		return -1
	}
}

func (o *Value) Has(k string) bool {
	if o == nil || o.dtype != Type_Object {
		return false
	}
	return o.mval.Has(k)
}

func (o *Value) Get(k string) *Value {
	if o == nil || o.dtype != Type_Object {
		return nil
	}
	r := o.mval.Get(k)
	if r == nil {
		return nil
	}
	return r
}

func (v *Value) GetBool(k string) bool {
	a := v.Get(k)
	if a != nil {
		return a.AsBool()
	}
	return false
}

func (v *Value) GetSlice(k string) []*Value {
	a := v.Get(k)
	if a != nil {
		if a.dtype == Type_Array {
			return a.aval
		}
	}
	return nil
}

func (v *Value) GetString(k string) string {
	n := v.Get(k)
	if n != nil {
		return n.AsString()
	}
	return ""
}

func (v *Value) AsStringSlice() []string {
	if v.dtype != Type_Array {
		return nil
	}
	result := make([]string, 0)
	for _, o := range v.aval {
		result = append(result, o.AsString())
	}
	return result
}

func (v *Value) GetStringSlice(k string) []string {
	a := v.Get(k)
	if a == nil {
		return nil
	}
	return a.AsStringSlice()
}

func (v *Value) GetInt(k string, def int) int {
	n := v.Get(k)
	if n != nil {
		return n.AsInt()
	}
	return def
}

func (v *Value) GetInt64(k string, def int64) int64 {
	n := v.Get(k)
	if n != nil {
		return n.AsInt64()
	}
	return def
}

func (v *Value) GetFloat64(k string, def float64) float64 {
	n := v.Get(k)
	if n != nil {
		return n.AsFloat64()
	}
	return def
}

type Array []*Value
func (a Array) String() string {
	return JsonEncode(a)
}

func NewString(s string) *Value {
	return &Value{dtype: Type_String, tval: s}
}

func NewArray() *Value {
	return &Value{dtype: Type_Array}
}

func NewArrayFromSlice(raw []any) *Value {
	a := NewArray()
	for _, e := range raw {
		a.aval = append(a.aval, NewValue(e))
	}
	return a
}

func NewObject() *Value {
	return &Value{dtype: Type_Object, mval: NewMap[*Value]()}
}

func NewObjectFromMap(raw map[string]any) *Value {
	m := NewObject()
	if raw != nil {
		for k, v := range raw {
			m.Put(k, NewValue(v))
		}
	}
	return m
}
