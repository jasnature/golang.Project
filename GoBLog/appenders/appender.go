//Appenders control the stream of data from a logger to an output.
//For example, a Console appender outputs log data to stdout.
//Satisfy the Appender interface to implement yourself log appender.

package appenders

import (
	"GoBLog/base"
	"GoBLog/formatters"
)

type Appender interface {
	Write(level levels.LogLevel, message string, args ...interface{})
	Layout() layout.Layout
	SetLayout(layout.Layout)
}
