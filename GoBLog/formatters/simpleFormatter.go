// simpleFormatter.go
package formatters

import (
	"GoBLog/base"
	"fmt"
	"time"
)

type SimpleFormatter struct {
	Formatter
}

func NewSimpleFormatter() *SimpleFormatter {
	return &SimpleFormatter{}
}

func (this *SimpleFormatter) Format(level base.LogLevel, location string, dtime time.Time, message string, args ...interface{}) string {
	strL := ""
	if message != "" {
		strL = fmt.Sprintf(message, args...)
	} else {
		strL = fmt.Sprint(args...)
	}

	return fmt.Sprintf("[%s] %s %s \n", base.LogLevelIntMap[level], base.DefaultUtil().NowTimeStr(0), strL)
}
