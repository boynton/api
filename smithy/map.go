package smithy

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Map[V any] struct {
	keys     []string
	bindings map[string]V
}

func NewMap[V any]() *Map[V] {
	return &Map[V]{
		bindings: make(map[string]V, 0),
	}
}

func (s *Map[V]) UnmarshalJSON(data []byte) error {
	keys, err := JsonKeysInOrder(data)
	if err != nil {
		return err
	}
	str := NewMap[V]()
	str.keys = keys
	err = json.Unmarshal(data, &str.bindings)
	if err != nil {
		return err
	}
	*s = *str
	return nil
}

func (s Map[V]) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	for i, key := range s.keys {
		value := s.bindings[key]
		if i > 0 {
			buffer.WriteString(",")
		}
		jsonValue, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf("%q:%s", key, string(jsonValue)))
	}
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

func JsonKeysInOrder(data []byte) ([]string, error) {
	var end = fmt.Errorf("invalid end of array or object")

	var skipValue func(d *json.Decoder) error
	skipValue = func(d *json.Decoder) error {
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch t {
		case json.Delim('['), json.Delim('{'):
			for {
				if err := skipValue(d); err != nil {
					if err == end {
						break
					}
					return err
				}
			}
		case json.Delim(']'), json.Delim('}'):
			return end
		}
		return nil
	}
	d := json.NewDecoder(bytes.NewReader(data))
	t, err := d.Token()
	if err != nil {
		return nil, err
	}
	if t != json.Delim('{') {
		return nil, fmt.Errorf("expected start of object")
	}
	var keys []string
	for {
		t, err := d.Token()
		if err != nil {
			return nil, err
		}
		if t == json.Delim('}') {
			return keys, nil
		}
		keys = append(keys, t.(string))
		if err := skipValue(d); err != nil {
			return nil, err
		}
	}
}

func (s *Map[V]) String() string {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	//	enc.SetIndent("", indentSize)
	if err := enc.Encode(s); err != nil {
		return fmt.Sprint(*s)
	}
	return string(buf.String())
}

func (s *Map[V]) find(key string) int {
	for i, k := range s.keys {
		if k == key {
			return i
		}
	}
	return -1
}

func (s *Map[V]) Has(key string) bool {
	if s != nil {
		if _, ok := s.bindings[key]; ok {
			return true
		}
	}
	return false
}

func (s *Map[V]) Get(key string) V {
	return s.bindings[key]
}

func (s *Map[V]) Put(key string, val V) {
	if s == nil {
		*s = *NewMap[V]()
	}
	if _, ok := s.bindings[key]; !ok {
		s.keys = append(s.keys, key)
	}
	s.bindings[key] = val
}

func (s *Map[V]) Delete(key string) {
	if s != nil {
		if _, ok := s.bindings[key]; ok {
			var tmp []string
			for _, k := range s.keys {
				if k != key {
					tmp = append(tmp, k)
				}
			}
			s.keys = tmp
			delete(s.bindings, key)
		}
	}
}

func (s *Map[V]) Keys() []string {
	if s == nil {
		return nil
	}
	return s.keys
}

func (s *Map[V]) Length() int {
	if s == nil || s.keys == nil {
		return 0
	}
	return len(s.keys)
}
