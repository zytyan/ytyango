package groupstatv2

import (
	"reflect"
)

// valuer 若类型实现了这个接口，则使用 Value() 返回值代替字段本身
type valuer interface {
	Value() any
}

// StructToMap 把结构体递归转换为 map[string]any
func StructToMap(obj any) any {
	return processValue(reflect.ValueOf(obj))
}

func processValue(rv reflect.Value) any {
	switch rv.Kind() {
	case reflect.Struct:
		if rv.CanAddr() {
			vp := rv.Addr().Interface()
			if val, ok := vp.(valuer); ok {
				return val.Value()
			}
		}
		tv := rv.Type()
		n := rv.NumField()

		m := make(map[string]any, n)
		for i := 0; i < n; i++ {
			field := tv.Field(i)
			if !field.IsExported() {
				continue
			}
			name := field.Name
			value := rv.Field(i)
			m[name] = processValue(value)
		}
		return m
	case reflect.Ptr:
		for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
			if rv.IsNil() {
				return nil
			}
			rv = rv.Elem()
		}
		return processValue(rv)
	case reflect.Slice, reflect.Array:
		l := rv.Len()
		arr := make([]any, l)
		for i := 0; i < l; i++ {
			arr[i] = processValue(rv.Index(i))
		}
		return arr
	case reflect.Map:
		return rv.Interface()
	default:
		return nil
	}
}
