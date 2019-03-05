// GoBLogFactory
package GoBLog

import (
	"GoBLog/appenders"
	"sync"
)

func init() {
	//fmt.Println("init LogFactory!")
	if DefaultLogFactory == nil {
		DefaultLogFactory = &LogFactory{
			defaultName: "DefaultLog",
			loggerPool:  make(map[string]ILogger, 2),
		}
	}
}

type RunOutputModel int

const (
	None          RunOutputModel = 0
	ConsoleOutput RunOutputModel = 1
	FileOutput    RunOutputModel = 2
)

var DefaultLogFactory *LogFactory

type LogFactory struct {
	defaultName   string
	defaultLogger ILogger
	loggerPool    map[string]ILogger
	mu            sync.Mutex
}

//Get default name logger and output use file stream.
func (this *LogFactory) GetLogger() ILogger {
	return this.GetLoggerByName("", FileOutput)
}

//name filename and put in logs folder.
//runModel[None-0 ConsoleOutput-1 FileOutput-2 or comb flag]
func (this *LogFactory) GetLoggerByName(name string, runModel RunOutputModel) (newlogger ILogger) {
	this.mu.Lock()
	defer this.mu.Unlock()

	if name == "" {
		if this.defaultLogger == nil {
			//fmt.Println("new defaultLogger")
			this.defaultLogger = NewGoBLogger(this.defaultName)
			newlogger = this.defaultLogger
		} else {
			return this.defaultLogger
		}
	} else {
		//fmt.Printf("get Logger %s \r\n", name)
		logger, ok := this.loggerPool[name]
		if !ok {
			//fmt.Printf("new Logger of return %s\r\n", name)
			logger = NewGoBLogger(name)
			this.loggerPool[name] = logger
		}
		newlogger = logger
	}

	//appender
	switch runModel {
	case None:
		newlogger.SetAppender(nil)
	case FileOutput:
		fapp := this.createFileAppender(name)
		newlogger.SetAppender(fapp)
	case ConsoleOutput | FileOutput:
		capp := appenders.NewConsoleAppender()
		fapp := this.createFileAppender(name)
		newlogger.SetAppender(appenders.NewMultipleAppender(50, fapp, capp))
	default:
		//default it is console
	}
	return newlogger
}

func (this *LogFactory) createFileAppender(name string) appenders.Appender {
	var fapp appenders.Appender

	if name == "" {
		fapp, _ = appenders.DefaultFileAppender()
	} else {
		fapp, _ = appenders.NewFileAppender("./logs/"+name, true)
	}
	return fapp
}

func (this *LogFactory) LoggerPoolList() []*ILogger {
	loggers := make([]*ILogger, 0)
	for _, v := range this.loggerPool {
		loggers = append(loggers, &v)
	}
	return loggers
}

func (this *LogFactory) DisposeAll() {
	for _, v := range this.loggerPool {
		v.Dispose()
	}
}
