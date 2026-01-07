// Package hreflect
//
// ----------------develop info----------------
//
//	@Author xunmuhuang@rastar.com
//	@DateTime 2026-1-7 10:18
//
// --------------------------------------------
package hreflect

import (
	"fmt"
	"testing"
)

func TestStructToStruct(t *testing.T) {
	type a struct {
		Name string
		Age  int
	}
	type b struct {
		Name string
		Age  int
	}

	tA := a{
		Name: "mm",
		Age:  18,
	}
	tB := b{}

	fmt.Println(tA, tB)

	EmbedCopy(&tB, &tA)

	fmt.Println(tA, tB)

	if tA.Name != tB.Name || tA.Age != tB.Age {
		t.Error("StructToStruct error")
	}

	tC := a{Name: "mm2", Age: 19}
	tD := b{}
	fmt.Println(tC, tD)
	EmbedCopy(&tD, tC)
	fmt.Println(tC, tD)

	t.Log("StructToStruct success")
}
