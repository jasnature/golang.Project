package appenders

import (
	"GoBLog/base"
	"GoBLog/formatters"
	"testing"
	"time"
)

func TestConsole(t *testing.T) {
	console := NewConsoleAppender()
	console.WriteString(base.DEBUG, "TestConsole=%d", 666)

	console.SetFormatter(formatters.DefaultPatternFormatter())
	console.WriteString(base.TRACE, "TestConsole=%d", 666)
}

func TestFileRolling(t *testing.T) {
	filer, err := NewFileAppender("goblog.log", true, 1024)
	filer.WriteString(base.DEBUG, "TestFileRolling=%d,err=%v", 666, err)

	filer.SetFormatter(formatters.DefaultPatternFormatter())
	for i := 0; i < 3; i++ {
		filer.WriteString(base.TRACE, "TestFileRolling=%d", i)
	}
}

func TestMutiple(t *testing.T) {
	console := NewConsoleAppender()
	filer, _ := NewFileAppender("goblog.log", true, 1024)

	mutiple := NewMultipleAppender(100, console, filer)

	for i := 0; i < 3; i++ {
		mutiple.WriteString(base.INFO, "TestMutiple=%d %s %d", i, "kkkkkkkkkkk", 123456)
	}

	time.Sleep(time.Second * 1)
}
