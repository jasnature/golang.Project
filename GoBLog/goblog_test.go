package GoBLog

import (
	_ "GoBLog/appenders"
	"GoBLog/base"
	"testing"
	_ "time"
)

func TestLogFactory(t *testing.T) {
	var d1 = DefaultLogFactory.GetLogger()
	var d2 = DefaultLogFactory.GetLogger()
	t.Logf("d1=d2?%v", d1 == d2)
	d1.Debug("11111111111111111111111")
	d2.Debug("2222222222222222222222222")
	d1.Dispose()

	var d3 = DefaultLogFactory.GetLoggerByName("xxoo", base.ConsoleOutput|base.FileOutput)
	var d4 = DefaultLogFactory.GetLoggerByName("xxoo1", base.ConsoleOutput|base.FileOutput)
	t.Logf("d3=d4?%v", d3 == d4)
	d3.Debug("333333333333333333333333")
	d4.Debug("4444444444444444444444444444")
	d3.Dispose()
	d4.Dispose()
	var ps = DefaultLogFactory.LoggerPoolList()
	t.Logf("LoggerPools:%v len=%d", ps, len(ps))
	//time.Sleep(time.Second * 3)
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

//func TestGoBLoggerToFile(t *testing.T) {
//	parentLogger := NewGoBLogger("Main")
//	app, _ := appenders.DefaultFileAppender()
//	fileapp := app.(*appenders.FileAppender)
//	//fmt.Println(reflect.TypeOf(app))
//	parentLogger.SetAppender(app)
//	var cc chan int = make(chan int)
//	go func() {
//		for i := 0; i < 2; i++ {
//			parentLogger.Infof("#####################=%d", i)
//			parentLogger.Info("my name is info.")
//			parentLogger.Debugf("#####################=%d", i)
//			parentLogger.Debug("My name is debug.")
//			parentLogger.Errorf("#####################=%d", i)
//			parentLogger.Error("my name is error.")
//		}
//		cc <- 1
//	}()
//	<-cc
//	fileapp.Dispose()
//}
