// Package hreflect
//
// ----------------develop info----------------
//
//	@Author xunmuhuang@rastar.com
//	@DateTime 2026-1-4 19:41
//
// --------------------------------------------
package hreflect

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// EmbedCopy
//
//	@Description:
//	@param dst interface{}
//	@param src interface{}
//
// ----------------develop info----------------
//
//	@Author:		Calmu
//	@DateTime:		2024-08-04 19:42:55
//
// --------------------------------------------
func EmbedCopy(dst, src interface{}) {
	dv := reflect.ValueOf(dst).Elem()
	sv := reflect.Indirect(reflect.ValueOf(src))

	for i := 0; i < sv.NumField(); i++ {
		sf := sv.Type().Field(i)
		// 找 dst 里同名字段
		if df := dv.FieldByName(sf.Name); df.IsValid() && df.CanSet() {
			if df.Type() == sf.Type {
				df.Set(sv.Field(i))
			}
		}
	}
}

// StructToMap 将结构体转换为map
func StructToMap(obj interface{}) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	objValue := reflect.ValueOf(obj)
	objType := reflect.TypeOf(obj)

	// 如果是指针，获取其指向的元素
	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
		objType = objType.Elem()
	}

	// 确保传入的是结构体
	if objValue.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a struct or pointer to struct")
	}

	for i := 0; i < objValue.NumField(); i++ {
		field := objValue.Field(i)
		fieldType := objType.Field(i)

		// 获取json标签作为键名，如果没有则使用字段名
		key := fieldType.Name
		if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" {
			// 解析json标签，处理如 "name,omitempty" 的情况
			if commaIdx := strings.Index(jsonTag, ","); commaIdx != -1 {
				key = jsonTag[:commaIdx]
			} else {
				key = jsonTag
			}
			// 如果json标签为"-"，则跳过该字段
			if key == "-" {
				continue
			}
		}

		// 如果字段是可导出的，添加到map中
		if field.CanInterface() {
			data[key] = field.Interface()
		}
	}

	return data, nil
}

// MapToStruct 将map转换为结构体
func MapToStruct(data map[string]interface{}, obj interface{}) error {
	objValue := reflect.ValueOf(obj)
	objType := reflect.TypeOf(obj)

	// 确保是指针类型
	if objValue.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer to struct")
	}

	objValue = objValue.Elem()
	objType = objType.Elem()

	// 确保指向的是结构体
	if objValue.Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a pointer to struct")
	}

	for i := 0; i < objValue.NumField(); i++ {
		field := objValue.Field(i)
		fieldType := objType.Field(i)

		// 获取json标签作为键名，如果没有则使用字段名
		key := fieldType.Name
		if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" {
			// 解析json标签，处理如 "name,omitempty" 的情况
			if commaIdx := strings.Index(jsonTag, ","); commaIdx != -1 {
				key = jsonTag[:commaIdx]
			} else {
				key = jsonTag
			}
			// 如果json标签为"-"，则跳过该字段
			if key == "-" {
				continue
			}
		}

		// 检查map中是否存在对应的键
		if value, exists := data[key]; exists {
			// 确保字段可设置
			if field.CanSet() {
				// 类型转换并设置值
				setValue(field, value)
			}
		}
	}

	return nil
}

// setValue 设置字段值，处理类型转换
func setValue(field reflect.Value, value interface{}) {
	// 如果值为nil，直接返回
	if value == nil {
		return
	}

	fieldType := field.Type()
	valueType := reflect.TypeOf(value)

	// 如果类型相同，直接设置
	if fieldType == valueType {
		field.Set(reflect.ValueOf(value))
		return
	}

	// 尝试类型转换
	switch field.Kind() {
	case reflect.String:
		if str, ok := value.(string); ok {
			field.SetString(str)
		} else if str, ok := value.(fmt.Stringer); ok {
			field.SetString(str.String())
		} else {
			field.SetString(fmt.Sprintf("%v", value))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case int:
			field.SetInt(int64(v))
		case int8:
			field.SetInt(int64(v))
		case int16:
			field.SetInt(int64(v))
		case int32:
			field.SetInt(int64(v))
		case int64:
			field.SetInt(v)
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				field.SetInt(i)
			}
		case float64:
			field.SetInt(int64(v))
		case float32:
			field.SetInt(int64(v))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := value.(type) {
		case uint:
			field.SetUint(uint64(v))
		case uint8:
			field.SetUint(uint64(v))
		case uint16:
			field.SetUint(uint64(v))
		case uint32:
			field.SetUint(uint64(v))
		case uint64:
			field.SetUint(v)
		case string:
			if i, err := strconv.ParseUint(v, 10, 64); err == nil {
				field.SetUint(i)
			}
		case float64:
			field.SetUint(uint64(v))
		case float32:
			field.SetUint(uint64(v))
		}
	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float32:
			field.SetFloat(float64(v))
		case float64:
			field.SetFloat(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				field.SetFloat(f)
			}
		case int:
			field.SetFloat(float64(v))
		case int64:
			field.SetFloat(float64(v))
		case uint:
			field.SetFloat(float64(v))
		case uint64:
			field.SetFloat(float64(v))
		}
	case reflect.Bool:
		switch v := value.(type) {
		case bool:
			field.SetBool(v)
		case string:
			if b, err := strconv.ParseBool(v); err == nil {
				field.SetBool(b)
			} else {
				field.SetBool(v != "" && v != "0" && v != "false")
			}
		case int:
			field.SetBool(v != 0)
		case int64:
			field.SetBool(v != 0)
		case float64:
			field.SetBool(v != 0)
		case float32:
			field.SetBool(v != 0)
		}
	case reflect.Struct:
		// 如果目标字段是结构体，且源值是map，尝试递归转换
		if srcMap, ok := value.(map[string]interface{}); ok {
			tempStruct := reflect.New(fieldType).Interface()
			MapToStruct(srcMap, tempStruct)
			field.Set(reflect.ValueOf(tempStruct).Elem())
		}
	case reflect.Ptr:
		// 如果目标字段是指针，创建一个新实例并设置值
		if field.IsNil() {
			field.Set(reflect.New(fieldType.Elem()))
		}
		setValue(field.Elem(), value)
	case reflect.Slice:
		// 如果目标字段是切片，且源值是切片
		if srcSlice, ok := value.([]interface{}); ok {
			sliceValue := reflect.MakeSlice(fieldType, len(srcSlice), len(srcSlice))
			for i, v := range srcSlice {
				setItem := sliceValue.Index(i)
				setValue(setItem, v)
			}
			field.Set(sliceValue)
		}
	default:
		// 其他情况尝试直接设置
		if reflect.ValueOf(value).Type().ConvertibleTo(fieldType) {
			field.Set(reflect.ValueOf(value).Convert(fieldType))
		}
	}
}
