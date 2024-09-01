package conf

import(
	"github.com/boynton/api/data"	
)

var current  *data.Value = data.NewObject()

func SetConf(newConf *data.Value) *data.Value {
	prev := current
	current = newConf
	return prev
}

func CopyConf() *data.Value {
	return SetConf(current.Copy())
}

func ClearConf() *data.Value {
	return SetConf(data.NewObject())
}

func Put(k string, rv any) {
	v := data.Null
	if rv != nil {
		switch vv := rv.(type) {
		case *data.Value:
			v = vv
		default:
			v = data.NewValue(rv)
		}
	}
	current.Put(k, v)
}

func Has(k string) bool {
	return current.Has(k)
}

func Get(k string) *data.Value {
	return current.Get(k)
}

func GetBool(k string) bool {
	return current.GetBool(k)
}

func GetInt(k string) int {
	return current.GetInt(k, 0)
}

func GetFloat64(k string) float64 {
	return current.GetFloat64(k, 0)
}

func GetString(k string) string {
	return current.GetString(k)
}

func GetSlice(k string) []*data.Value {
	return current.GetSlice(k)
}

func GetStringSlice(k string) []string {
	return current.GetStringSlice(k)
}

