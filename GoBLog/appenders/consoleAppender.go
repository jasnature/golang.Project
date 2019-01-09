// consoleAppender
package appenders

import (
	"GoBLog/base"
	"GoBLog/formatters"
	"fmt"
)

type ConsoleAppender struct {
	Appender
	formatters.FormatterManager
	formatter formatters.Formatter
}

func NewConsoleAppender() *ConsoleAppender {
	this := &ConsoleAppender{
		formatter: formatters.NewSimpleFormatter(),
	}
	return this
}

func (this *ConsoleAppender) WriteString(level base.LogLevel, message string, args ...interface{}) {
	fmt.Println(this.Formatter().Format(level, message, args...))
}

func (this *ConsoleAppender) Formatter() formatters.Formatter {
	return this.formatter
}

func (this *ConsoleAppender) SetFormatter(formatter formatters.Formatter) {
	this.formatter = formatter
}
