// consoleAppender
package appenders

import (
	"GoBLog/base"
	"GoBLog/formatters"
	"fmt"
	"time"
)

type ConsoleAppender struct {
	AppenderBase
}

func NewConsoleAppender() *ConsoleAppender {
	this := &ConsoleAppender{}
	this.formatter = formatters.NewSimpleFormatter()
	return this
}

func (this *ConsoleAppender) WriteString(level base.LogLevel, location string, dtime time.Time, message string, args ...interface{}) {
	fmt.Print(this.Formatter().Format(level, location, dtime, message, args...))
}

func (this *ConsoleAppender) Formatter() formatters.Formatter {
	return this.formatter
}

func (this *ConsoleAppender) SetFormatter(formatter formatters.Formatter) {
	this.formatter = formatter
}

func (this *ConsoleAppender) Dispose() (err error) {
	//none dispose resouce.
	return nil
}
