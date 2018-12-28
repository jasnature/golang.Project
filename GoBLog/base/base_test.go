package base

import (
	"fmt"
	"testing"
)

func TestLevels(t *testing.T) {
	var str string
	for k, v := range LogLevelStringMap {
		str += fmt.Sprintf("%v=%v ", k, v)
	}
	t.Log(str)
	str = ""
	for k, v := range LogLevelIntMap {
		str += fmt.Sprintf("%v=%v ", k, v)
	}
	t.Log(str)
	var a LogLevel = TRACE
	b := TRACE
	t.Log("LogLevel =", a, "Single Declare:", a == b)
}
func TestUtil(t *testing.T) {
	t.Log(DefaultUtil() == &Util{})
	t.Log(DefaultUtil() == DefaultUtil())
	t.Log(DefaultUtil().NowTimeStr())
}
