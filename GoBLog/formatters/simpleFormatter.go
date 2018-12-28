// simpleFormatter.go
package formatters

import (
	"GoBLog/base"
	"fmt"
)

type SimpleFormatter struct {
	Formatter
}

func NewSimpleFormatter() *SimpleFormatter {
	return &SimpleFormatter{}
}

func (this *SimpleFormatter) Format(level base.LogLevel, message string, args ...interface{}) string {
	//return fmt.Sprintf("%s %s", base.DefaultUtil().NowTimeStr(), fmt.Sprintf(message, args...))
	return fmt.Sprintf(message, args...)
}
