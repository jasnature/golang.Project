package appenders

import (
	"GoBLog/base"
	"time"
)

//can support mutiple type appender.
type multipleOutputAppender struct {
	AppenderBase
	appenderList []Appender
	isDispose    bool
	syncChan     chan chanMsg
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
	obj.isDispose = false
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
	if !this.isDispose {
		this.syncChan <- chanMsg{
			level,
			location,
			dtime,
			message,
			args,
		}
	}
}

func (this *multipleOutputAppender) Dispose() error {
	defer this.mu_lock.Unlock()
	this.mu_lock.Lock()
	if !this.isDispose {
		this.isDispose = true
		for try := 10; try > 0; try-- {
			time.Sleep(time.Millisecond * 50)
			if len(this.syncChan) <= 0 {
				//fmt.Println("multipleOutputAppender syncChan=0")
				break
			}
		}
		for _, appender := range this.appenderList {
			ref, ok := appender.(base.IDispose)
			if ok {
				//fmt.Println("multipleOutputAppender Dispose")
				ref.Dispose()
			}
		}
	}
	return nil
}
