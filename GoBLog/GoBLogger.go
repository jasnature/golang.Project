package GoBLog

import (
	"GoBLog/appenders"
	"GoBLog/base"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

type ILogger interface {
	appenders.AppenderManager
	base.IDispose
	Name() string
	FullLinkName() string
	Parent() ILogger
	Level() string
	SetLevel(base.LogLevel)
	Log(level base.LogLevel, formate string, params ...interface{})
	NewChildLogger(logName string) (ILogger, error)
	ChildMapList() map[string]ILogger
	GetChildLogger(name string) ILogger

	Fatal(params ...interface{})
	Fatalf(formate string, params ...interface{})
	Error(params ...interface{})
	Errorf(formate string, params ...interface{})
	Warn(params ...interface{})
	Warnf(formate string, params ...interface{})
	Info(params ...interface{})
	Infof(formate string, params ...interface{})
	Debug(params ...interface{})
	Debugf(formate string, params ...interface{})
	Trace(params ...interface{})
	Tracef(formate string, params ...interface{})
}

//full implement ILogger interface
type GoBLogger struct {
	ILogger
	level       base.LogLevel
	logName     string
	appender    appenders.Appender
	childrens   map[string]ILogger
	parent      ILogger
	ExitOnFatal bool
}

// New a default debug level Logger by output console and then reset some field.
// logName can set empty,logname use to differentiate and division logger and children logger
func NewGoBLogger(logName string) ILogger {
	obj := ILogger(&GoBLogger{
		level:       base.DEBUG,
		logName:     logName,
		appender:    appenders.NewConsoleAppender(),
		childrens:   make(map[string]ILogger, 3),
		parent:      nil,
		ExitOnFatal: true,
	})
	obj.SetLevel(base.DEBUG)
	return obj
}

//New child logger ,level inherit parent can reset, appender inherit, parent point to upper
func (this *GoBLogger) NewChildLogger(logName string) (ILogger, error) {
	if strings.TrimSpace(logName) == "" {
		return nil, errors.New("logname is null.")
	}
	child := ILogger(&GoBLogger{
		parent:    this,
		level:     base.LogLevelStringMap[this.Level()],
		logName:   logName,
		appender:  nil,
		childrens: make(map[string]ILogger, 0),
	})
	if _, ok := this.childrens[logName]; ok {
		return nil, errors.New("logname it is exist.")
	}
	this.childrens[logName] = child
	return child, nil
}

func (this *GoBLogger) GetChildLogger(name string) ILogger {
	return this.childrens[name]
}

func (this *GoBLogger) Log(level base.LogLevel, formate string, params ...interface{}) {
	//None=0 FATAL ERROR WARN INFO DEBUG TRACE
	if this.level < level {
		return
	}
	app := this.Appender()

	if app == nil {
		return
	}
	_, file, line, ok := runtime.Caller(2)

	if !ok {
		file = "not found file."
		line = 0
	}

	if app != nil {
		app.WriteString(level, fmt.Sprintf("%s , %d", file, line), time.Now(), formate, params...)
	}

	if this.ExitOnFatal && level == base.FATAL {
		//fmt.Println("ExitOnFatal")
		os.Exit(1)
	}
}

func (this *GoBLogger) Level() string {
	return base.LogLevelIntMap[this.level]
}

func (this *GoBLogger) SetLevel(level base.LogLevel) {
	this.level = level
}

func (this *GoBLogger) ChildMapList() map[string]ILogger {
	return this.childrens
}

func (this *GoBLogger) Parent() ILogger {
	return this.parent
}

func (this *GoBLogger) Name() string {
	return this.logName
}

func (this *GoBLogger) FullLinkName() string {
	n := this.logName
	if this.parent != nil {
		p := this.parent.FullLinkName()
		if len(p) > 0 {
			n = this.parent.FullLinkName() + "<-" + n
		}
	}
	return n
}

func (this *GoBLogger) SetAppender(appender appenders.Appender) {
	this.appender = appender
}

func (this *GoBLogger) Appender() appenders.Appender {
	if ap := this.appender; ap != nil {
		return ap
	}
	if this.parent != nil {

		if ap := this.parent.Appender(); ap != nil {
			//if find then init it to child.
			this.appender = ap
			return ap
		}
	}
	return nil
}

func (this *GoBLogger) String() string {
	TypeName := "ChildNode"
	if this.parent == nil {
		TypeName = "RootNode"
	}
	return fmt.Sprintf(" { [LoggerName]=%s [FullName]=%s [Type]=%s [ChildCount]=%d [Level]=%s } ", this.logName, this.FullLinkName(), TypeName, len(this.childrens), this.Level())
}

func (this *GoBLogger) Dispose() error {
	ref, ok := this.appender.(base.IDispose)
	if ok {
		ref.Dispose()
	}
	return nil
}

func (this *GoBLogger) Fatal(params ...interface{}) { this.Log(base.FATAL, "", params...) }
func (this *GoBLogger) Fatalf(formate string, params ...interface{}) {
	this.Log(base.FATAL, formate, params...)
}
func (this *GoBLogger) Error(params ...interface{}) { this.Log(base.ERROR, "", params...) }
func (this *GoBLogger) Errorf(formate string, params ...interface{}) {
	this.Log(base.ERROR, formate, params...)
}
func (this *GoBLogger) Warn(params ...interface{}) { this.Log(base.WARN, "", params...) }
func (this *GoBLogger) Warnf(formate string, params ...interface{}) {
	this.Log(base.WARN, formate, params...)
}
func (this *GoBLogger) Info(params ...interface{}) { this.Log(base.INFO, "", params...) }
func (this *GoBLogger) Infof(formate string, params ...interface{}) {
	this.Log(base.INFO, formate, params...)
}
func (this *GoBLogger) Debug(params ...interface{}) { this.Log(base.DEBUG, "", params...) }
func (this *GoBLogger) Debugf(formate string, params ...interface{}) {
	this.Log(base.DEBUG, formate, params...)
}
func (this *GoBLogger) Trace(params ...interface{}) { this.Log(base.TRACE, "", params...) }
func (this *GoBLogger) Tracef(formate string, params ...interface{}) {
	this.Log(base.TRACE, formate, params...)
}
