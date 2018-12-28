package formatters

import (
	"GoBLog/base"
	"testing"
)

func TestSimple(t *testing.T) {

	str := NewSimpleFormatter().Format(base.DEBUG, "My type it is float=%f", 123.345)
	t.Log(str)
}

func TestBasic(t *testing.T) {
	var pattern string = "%p %d %l %n LogInfo: %m"

	str := NewPatternFormatter(pattern).Format(base.DEBUG, "NewBasicFormatter log=%d", 123456)
	str1 := DefaultPatternFormatter().Format(base.TRACE, "NewBasicFormatter log=%d", 123456)
	t.Log(str)
	t.Log(str1)
}
