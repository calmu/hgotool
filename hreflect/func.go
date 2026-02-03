// Package hreflect
//
// ----------------develop info----------------
//
//	@Author Calmu
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
	// 获取源和目标的反射值
	srcValue := reflect.ValueOf(src)
	dstValue := reflect.ValueOf(dst)

	// 如果任一值不可寻址或者不是指针，返回
	if !dstValue.IsValid() || dstValue.Kind() != reflect.Ptr {
		return
	}

	// 解引用指针获取实际的值
	dstValue = dstValue.Elem()
	if !srcValue.IsValid() {
		return
	}

	// 如果源是指针，解引用它
	if srcValue.Kind() == reflect.Ptr {
		srcValue = srcValue.Elem()
	}

	// 确保目标是可设置的
	if !dstValue.CanSet() {
		return
	}

	// 逐个字段进行复制
	copyFieldsByName(dstValue, srcValue)
}

// copyFieldsByName 按字段名复制两个结构体之间的字段
func copyFieldsByName(dstValue, srcValue reflect.Value) {
	if dstValue.Kind() != reflect.Struct || srcValue.Kind() != reflect.Struct {
		return
	}

	dstType := dstValue.Type()

	// 遍历目标结构体的所有可导出字段
	for i := 0; i < dstValue.NumField(); i++ {
		dstField := dstValue.Field(i)
		dstFieldType := dstType.Field(i)

		// 跳过非导出字段
		if !dstField.CanSet() {
			continue
		}

		if dstFieldType.Anonymous {
			// 对于嵌入字段，递归处理其内部字段
			copyFieldsByName(dstField, srcValue)
		} else {
			// 尝试从源结构体中获取同名字段
			srcField := srcValue.FieldByName(dstFieldType.Name)
			if !srcField.IsValid() {
				// 如果直接没有找到同名字段，尝试在嵌入字段中查找
				srcField = findFieldRecursively(srcValue, dstFieldType.Name, dstField.Type())
			}

			if srcField.IsValid() && srcField.Type() == dstField.Type() && dstField.CanSet() {
				dstField.Set(srcField)
			}
		}
	}
}

// findFieldRecursively 递归查找字段
func findFieldRecursively(value reflect.Value, fieldName string, fieldType reflect.Type) reflect.Value {
	if value.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	// 直接查找
	field := value.FieldByName(fieldName)
	if field.IsValid() && field.Type() == fieldType {
		return field
	}

	// 在嵌入字段中查找
	for i := 0; i < value.NumField(); i++ {
		subField := value.Field(i)
		subFieldType := value.Type().Field(i)

		// 只在嵌入字段中递归查找
		if subFieldType.Anonymous && subField.Kind() == reflect.Struct {
			result := findFieldRecursively(subField, fieldName, fieldType)
			if result.IsValid() {
				return result
			}
		}
	}

	return reflect.Value{}
}

// copyStructFields 复制结构体字段
func copyStructFields(dstValue, srcValue reflect.Value) {
	if dstValue.Kind() != reflect.Struct || srcValue.Kind() != reflect.Struct {
		return
	}

	dstType := dstValue.Type()

	// 遍历目标结构体的所有字段
	for i := 0; i < dstValue.NumField(); i++ {
		dstField := dstValue.Field(i)
		dstFieldType := dstType.Field(i)

		// 跳过不可设置的字段
		if !dstField.CanSet() {
			continue
		}

		// 如果是嵌入字段，递归处理
		if dstFieldType.Anonymous {
			copyStructFields(dstField, findSrcEmbeddedField(srcValue, dstField.Type()))
		} else {
			// 查找源结构体中对应的字段
			srcFieldName := dstFieldType.Name
			srcField := srcValue.FieldByName(srcFieldName)

			if srcField.IsValid() && isAssignable(srcField.Type(), dstField.Type()) {
				if dstField.CanSet() {
					// 确保类型兼容后再进行赋值
					if srcField.Type() == dstField.Type() {
						dstField.Set(srcField)
					} else {
						// 如果类型不同但兼容，需要特殊处理
						setField(dstField, srcField)
					}
				}
			} else {
				// 如果直接字段不存在或类型不匹配，尝试更深入的查找
				// 搜索源结构体中是否有嵌入字段包含所需字段
				foundField := findFieldInEmbeddedStructs(srcValue, dstFieldType.Name, dstField.Type())
				if foundField.IsValid() && isAssignable(foundField.Type(), dstField.Type()) {
					if dstField.CanSet() {
						if foundField.Type() == dstField.Type() {
							dstField.Set(foundField)
						} else {
							setField(dstField, foundField)
						}
					}
				}
			}
		}
	}
}

// findSrcEmbeddedField 在源结构体中查找指定类型的嵌入字段
func findSrcEmbeddedField(srcValue reflect.Value, targetType reflect.Type) reflect.Value {
	if srcValue.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	// 首先检查顶层字段是否匹配
	for i := 0; i < srcValue.NumField(); i++ {
		srcField := srcValue.Field(i)
		srcFieldType := srcValue.Type().Field(i)

		// 如果字段类型匹配，直接返回
		if srcField.Type() == targetType {
			return srcField
		}

		// 如果是嵌入字段，递归搜索
		if srcFieldType.Anonymous { // 是嵌入字段
			if srcField.Type() == targetType {
				return srcField
			}
			// 递归搜索嵌套的嵌入字段
			result := findSrcEmbeddedField(srcField, targetType)
			if result.IsValid() {
				return result
			}
		}
	}

	return reflect.Value{}
}

// findFieldInEmbeddedStructs 在嵌入结构体中查找指定名称和类型的字段
func findFieldInEmbeddedStructs(srcValue reflect.Value, fieldName string, fieldType reflect.Type) reflect.Value {
	if srcValue.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	// 检查当前层级是否包含该字段
	srcField := srcValue.FieldByName(fieldName)
	if srcField.IsValid() && srcField.Type() == fieldType {
		return srcField
	}

	// 递归搜索嵌入字段
	for i := 0; i < srcValue.NumField(); i++ {
		srcField := srcValue.Field(i)
		srcFieldType := srcValue.Type().Field(i)

		// 如果是嵌入字段，递归搜索
		if srcFieldType.Anonymous {
			result := findFieldInEmbeddedStructs(srcField, fieldName, fieldType)
			if result.IsValid() {
				return result
			}
		}
	}

	return reflect.Value{}
}

// isAssignable 检查源类型是否可分配给目标类型
func isAssignable(srcType, dstType reflect.Type) bool {
	// 基本类型相同则可分配
	if srcType == dstType {
		return true
	}
	// 检查是否为基本数据类型且可以转换
	switch srcType.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		return srcType.Kind() == dstType.Kind()
	}
	return false
}

// setField 设置字段值，处理类型转换
func setField(dstField, srcField reflect.Value) {
	if !srcField.CanInterface() || !dstField.CanSet() {
		return
	}

	// 如果类型完全一致，则直接设置
	if srcField.Type() == dstField.Type() {
		dstField.Set(srcField)
		return
	}

	// 按类型进行转换和设置
	switch srcField.Kind() {
	case reflect.String:
		if dstField.Kind() == reflect.String {
			dstField.SetString(srcField.String())
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if dstField.Kind() == reflect.Int || dstField.Kind() == reflect.Int64 {
			dstField.SetInt(srcField.Int())
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if dstField.Kind() == reflect.Uint || dstField.Kind() == reflect.Uint64 {
			dstField.SetUint(srcField.Uint())
		}
	case reflect.Float32, reflect.Float64:
		if dstField.Kind() == reflect.Float64 || dstField.Kind() == reflect.Float32 {
			dstField.SetFloat(srcField.Float())
		}
	case reflect.Bool:
		if dstField.Kind() == reflect.Bool {
			dstField.SetBool(srcField.Bool())
		}
	case reflect.Struct:
		// 对于结构体，递归复制其字段
		copyStructFields(dstField, srcField)
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
