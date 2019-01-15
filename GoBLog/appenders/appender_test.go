package appenders

import (
	"GoBLog/base"
	"GoBLog/formatters"
	"testing"
	"time"
	//"time"
)

//func TestConsole(t *testing.T) {
//	console := NewConsoleAppender()
//	console.WriteString(base.DEBUG, "TestConsole=%d", 666)

//	console.SetFormatter(formatters.DefaultPatternFormatter())
//	console.WriteString(base.TRACE, "TestConsole=%d", 666)
//}

func TestFileRolling(t *testing.T) {

	//filer, _ := NewFileAppender("goblog.log", true)
	filer, _ := NewFileAppender("", true)

	//filer.WriteString(base.DEBUG, "TestFileRolling=%d %s", 1, "1111")

	filer.SetFormatter(formatters.DefaultPatternFormatter())
	for i := 0; i < 20; i++ {
		filer.WriteString(base.TRACE, "test", time.Now(), "TestFileRolling=[%d] %s", i, "中文测试")
		time.Sleep(time.Millisecond * 100)
	}
	t.Logf("dispose:%v \r\n", filer.Dispose())
	time.Sleep(time.Second * 1)
}

//func TestDefaultFileRolling(t *testing.T) {
//	var ffi, _ = DefaultFileAppender()

//	//1 switch
//	switch this := ffi.(type) {
//	case *FileAppender:
//		this.SetFormatter(formatters.DefaultPatternFormatter())
//	}
//	//2 direct assert
//	ff := ffi.(*FileAppender)
//	ff.SetFormatter(formatters.DefaultPatternFormatter())
//}

//func TestMutiple(t *testing.T) {
//	console := NewConsoleAppender()
//	filer, _ := NewFileAppender("goblog.log", false)

//	mutiple := NewMultipleAppender(100, console, filer)

//	for i := 0; i < 3; i++ {
//		mutiple.WriteString(base.INFO, "TestMutiple=%d %s %d", i, "kkkkkkkkkkk", 123456)
//	}

//	time.Sleep(time.Second * 1)
//}
