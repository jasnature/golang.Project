//Appenders control the stream of data from a logger to an output.
//For example, a Console appender outputs log data to stdout.
//Satisfy the Appender interface to implement yourself log appender.

package appenders

import (
	"GoBLog/base"
)

type Appender interface {
	WriteString(level base.LogLevel, message string, args ...interface{})
}

//auto date filename and append model
//buffer 64KB, file size max 6MB
func DefaultFileAppender() (Appender, error) {
	return NewFileAppender("", true)
}
