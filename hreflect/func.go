// Package hreflect
//
// ----------------develop info----------------
//
//	@Author xunmuhuang@rastar.com
//	@DateTime 2026-1-4 19:41
//
// --------------------------------------------
package hreflect

import "reflect"

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
	sv := reflect.ValueOf(src)

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
