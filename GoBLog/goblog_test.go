package GoBLog

import (
	"GoBLog/appenders"
	"GoBLog/base"
	"testing"
)

func TestLogFactory(t *testing.T) {

}

func TestGoBLogger(t *testing.T) {
	parentLogger := NewGoBLogger("Main")
	parentLogger.SetLevel(base.DEBUG)
	childlo1, _ := parentLogger.NewChildLogger("child")

	t.Logf("parentLogger name=%s,childlo1 name=%s ,parentLogger == childlo1.Parent?%v", parentLogger.Name(), childlo1.Name(), parentLogger == childlo1.Parent())

	childlo2, _ := childlo1.NewChildLogger("child2")
	childlo2.SetLevel(base.INFO)
	childlo3, _ := childlo2.NewChildLogger("child3")

	t.Logf("childlo3 Appender=%+v childlo3 Appender=parentLogger Appender?%v", childlo3.Appender(), childlo3.Appender() == parentLogger.Appender())

	t.Logf("Level list: parentLogger=%v c1=%v c2=%v c3=%v", parentLogger.Level(), childlo1.Level(), childlo2.Level(), childlo3.Level())

	t.Logf("childlo3 FullLinkName %s", childlo3.FullLinkName())

	parentLogger.Log(base.TRACE, "", 234)
	childlo1.Log(base.DEBUG, "", 234)
	childlo2.Log(base.WARN, "", 234)

	t.Logf("parentLogger GetLogger %s childlo1==parentLogger GetLogger?%v", parentLogger.GetChildLogger("child"), parentLogger.GetChildLogger("child") == childlo1)
	t.Logf("parentLogger GetLogger %s", parentLogger.GetChildLogger("123"))

}

func TestGoBLoggerToFile(t *testing.T) {
	parentLogger := NewGoBLogger("Main")
	app, _ := appenders.DefaultFileAppender()
	fileapp := app.(*appenders.FileAppender)
	//fmt.Println(reflect.TypeOf(app))
	parentLogger.SetAppender(app)
	var cc chan int = make(chan int)
	go func() {
		for i := 0; i < 10; i++ {
			parentLogger.Infof("#####################=%d", i)
			parentLogger.Debugf("#####################=%d", i)
			parentLogger.Errorf("#####################=%d", i)
		}
		cc <- 1
	}()
	<-cc
	fileapp.Dispose()
}
