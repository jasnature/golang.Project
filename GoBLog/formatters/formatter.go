//Formatters control the formatting of data into a printable log string.
//For example, the Basic basicFormatter.go passes the log message and arguments
//through fmt.Sprintf.
//Satisfy the Formatter interface to implement yourself log layout.

package formatters

import (
	"GoBLog/base"
)

type Formatter interface {
	Format(level base.LogLevel, message string, args ...interface{}) string
}

//Return a default format
func DefaultFormatter() Formatter {
	return NewSimpleFormatter()
}

func DefaultPatternFormatter() Formatter {
	return NewPatternFormatter("[%p] %d %l %n LogInfo-> %m")
}
