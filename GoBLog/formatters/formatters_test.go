package formatters

import (
	"GoBLog/base"
	"testing"
)

func TestSimple(t *testing.T) {

	str := NewSimpleFormatter().Format(base.INFO, "My type it is float=%f", 123.345)
	t.Log(str)
}

func TestBasic(t *testing.T) {
	var pattern string = "%p %d{15:04:05.000} %l %n LogInfo: %m \n"

	str := NewPatternFormatter(pattern).Format(base.DEBUG, "NewBasicFormatter log=%d", 123456)
	str1 := DefaultPatternFormatter().Format(base.TRACE, "NewBasicFormatter log=%d", 123456)
	t.Log(str)
	t.Log(str1)
}
