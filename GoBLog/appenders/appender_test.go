package appenders

import (
	"GoBLog/base"
	"GoBLog/formatters"
	"testing"
)

func TestConsole(t *testing.T) {
	console := NewConsoleAppender()
	console.Write(base.DEBUG, "TestConsole=%d", 666)

	console.SetFormatter(formatters.DefaultPatternFormatter())
	console.Write(base.TRACE, "TestConsole=%d", 666)
}

func TestFileRolling(t *testing.T) {
	console := NewFileAppender("goblog.log", true, 1024)
	console.Write(base.DEBUG, "TestFileRolling=%d", 666)

	console.SetFormatter(formatters.DefaultPatternFormatter())
	console.Write(base.TRACE, "TestFileRolling=%d", 666)
}
