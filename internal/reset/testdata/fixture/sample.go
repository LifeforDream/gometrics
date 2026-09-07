package fixture

import (
	"bytes"
	"time"
)

type namedList []int

type namedMap map[string]string

// generate:reset
type ResetableStruct struct {
	i      int
	str    string
	strP   *string
	s      []int
	s5     [5]int
	m      map[string]string
	il     namedList
	im     namedMap
	child  *ResetableStruct
	t      time.Time
	t2     time.Time
	rchild ResettableChild
	buf    bytes.Buffer
	pbuf   *bytes.Buffer
}

// NotAnnotated не должна затрагиваться генератором: метод Reset()
// для неё генерироваться не должен.
type NotAnnotated struct {
	X int
}

// ResettableChild позволяет убедиться, что для дочерних структур,
// не используемых по указателю, вызывается Reset().
type ResettableChild struct {
	s []int
}

// Reset в данном случае правильно обрабатывает очистку слайса.
func (rc *ResettableChild) Reset() {
	rc.s = rc.s[:0]
}
