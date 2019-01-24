//Appenders control the stream of data from a logger to an output.
//For example, a Console appender outputs log data to stdout.
//Satisfy the Appender interface to implement yourself log appender.

package appenders

import (
	"GoBLog/base"
	"GoBLog/formatters"
	"sync"
	"time"
)

type Appender interface {
	WriteString(level base.LogLevel, location string, dtime time.Time, message string, args ...interface{})
}

type AppenderManager interface {
	Appender() Appender
	SetAppender(appender Appender)
}

//auto filename by date and append log model
//buffer 256KB,max size 6MB for each file , auto flush buffer each 60s.
func DefaultFileAppender() (Appender, error) {
	return NewFileAppender("", true)
}

//Appender common base
type AppenderBase struct {
	Appender
	base.IDispose
	formatters.FormatterManager
	formatter formatters.Formatter

	isDispose  bool
	mu_lock    sync.Mutex
	bufferChan chan string
}
