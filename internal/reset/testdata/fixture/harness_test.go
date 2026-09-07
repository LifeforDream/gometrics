package fixture

import (
	"bytes"
	"testing"
	"time"
)

// Этот файл вместе с sample.go копируется интеграционным тестом cmd/reset
// во временный модуль. Он проверяет reset.gen.go, который должен
// сгенерироваться для ResetableStruct, поэтому находится в том же пакете —
// иначе не будет доступа к неэкспортированным полям.

func TestResetableStruct_Reset(t *testing.T) {
	strVal := "hello"
	rs := &ResetableStruct{
		i:    42,
		str:  "hello",
		strP: &strVal,
		s:    []int{1, 2, 3},
		s5:   [5]int{1, 2, 3, 4, 5},
		m:    map[string]string{"a": "b"},
		il:   namedList{1, 2, 3},
		im:   namedMap{"a": "b"},
		child: &ResetableStruct{
			i: 7,
		},
		t:      time.Now(),
		rchild: ResettableChild{s: []int{1, 2, 3, 4, 5}},
		pbuf:   &bytes.Buffer{},
	}
	rs.buf.WriteString("hello")
	rs.pbuf.WriteString("world")

	rs.Reset()

	if rs.i != 0 {
		t.Errorf("i = %d, want 0", rs.i)
	}
	if rs.str != "" {
		t.Errorf("str = %q, want empty", rs.str)
	}
	if rs.strP == nil {
		t.Fatal("strP should stay non-nil, only its pointee should reset")
	}
	if *rs.strP != "" {
		t.Errorf("*strP = %q, want empty", *rs.strP)
	}
	if len(rs.s) != 0 {
		t.Errorf("len(s) = %d, want 0", len(rs.s))
	}
	if cap(rs.s) < 3 {
		t.Errorf("cap(s) = %d, want capacity preserved (slice truncated, not reallocated)", cap(rs.s))
	}
	for _, elem := range rs.s5 {
		if elem != 0 {
			t.Errorf("elements of fixed array s5 should get zeroed")
		}
	}
	if len(rs.m) != 0 {
		t.Errorf("len(m) = %d, want 0 (cleared, not nilled)", len(rs.m))
	}
	if rs.m == nil {
		t.Error("m should stay non-nil, only cleared")
	}
	if len(rs.il) != 0 {
		t.Errorf("len(il) = %d, want 0", len(rs.il))
	}
	if cap(rs.il) < 3 {
		t.Errorf("cap(il) = %d, want capacity preserved (slice truncated, not reallocated)", cap(rs.il))
	}
	if len(rs.im) != 0 {
		t.Errorf("len(im) = %d, want 0 (cleared, not nilled)", len(rs.im))
	}
	if rs.im == nil {
		t.Error("im should stay non-nil, only cleared")
	}
	if rs.child == nil {
		t.Fatal("child should stay non-nil, only its own Reset should run")
	}
	if rs.child.i != 0 {
		t.Errorf("child.i = %d, want 0 (nested Reset() should have been called)", rs.child.i)
	}
	if rs.t.IsZero() != true {
		t.Errorf("t should become zero")
	}
	if rs.t2.IsZero() != true {
		t.Errorf("t2 should become zero")
	}
	if len(rs.rchild.s) != 0 {
		t.Errorf("len(rchild.s) = %d, want 0", len(rs.rchild.s))
	}
	if cap(rs.rchild.s) < 5 {
		t.Errorf("cap(rchild.s) = %d, want capacity preserved (slice truncated, not reallocated)", cap(rs.rchild.s))
	}
	if rs.buf.Len() != 0 {
		t.Errorf("buf.Len() = %d, want 0 (value-typed field from an imported package must delegate to its own Reset())", rs.buf.Len())
	}
	if rs.buf.Cap() == 0 {
		t.Errorf("buf.Cap() = 0, want capacity preserved (Reset() truncates, *new(bytes.Buffer) would reallocate)")
	}
	if rs.pbuf == nil {
		t.Fatal("pbuf should stay non-nil, only its pointee should reset")
	}
	if rs.pbuf.Len() != 0 {
		t.Errorf("pbuf.Len() = %d, want 0 (pointer to a type with Reset() from another package must delegate to it)", rs.pbuf.Len())
	}
}

func TestResetableStruct_Reset_NilReceiver(_ *testing.T) {
	var rs *ResetableStruct
	rs.Reset() // не должен паниковать
}
