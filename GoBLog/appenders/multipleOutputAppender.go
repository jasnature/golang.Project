package appenders

import (
	"GoBLog/base"
	//"fmt"
	"time"
)

//can support mutiple type appender.
type multipleOutputAppender struct {
	Appender
	base.IDispose
	appenderList []Appender

	syncChan chan chanMsg
}

type chanMsg struct {
	level    base.LogLevel
	location string
	dtime    time.Time
	message  string
	args     []interface{}
}

func NewMultipleAppender(maxQueue int, childAppenders ...Appender) Appender {
	if maxQueue <= 0 || maxQueue >= 1000 {
		maxQueue = 100
	}
	obj := &multipleOutputAppender{
		appenderList: childAppenders,
		syncChan:     make(chan chanMsg, maxQueue),
	}
	go obj.processWriteString()
	return obj
}

func (this *multipleOutputAppender) processWriteString() {
	for data := range this.syncChan {
		for _, appender := range this.appenderList {
			appender.WriteString(data.level, data.location, data.dtime, data.message, data.args...)
		}
	}
}

func (this *multipleOutputAppender) WriteString(level base.LogLevel, location string, dtime time.Time, message string, args ...interface{}) {
	this.syncChan <- chanMsg{
		level,
		location,
		dtime,
		message,
		args,
	}
}

func (this *multipleOutputAppender) Dispose() error {
	for try := 10; try > 0; try-- {
		if len(this.syncChan) <= 0 {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}
	for _, appender := range this.appenderList {
		ref, ok := appender.(base.IDispose)
		if ok {
			//fmt.Println("multipleOutputAppender Dispose")
			ref.Dispose()
		}
	}
	return nil
}
