package GoBLog

import (
	"GoBLog/appenders"
	"GoBLog/base"
	"errors"
	"strings"
)

type ILogger interface {
	appenders.AppenderManager
	Name() string
	FullLinkName() string
	Enabled() map[base.LogLevel]bool
	Parent() ILogger
	Level() string
	SetLevel(base.LogLevel)
	Log(level base.LogLevel, formate string, params ...interface{})
	NewChildLogger(logName string) (ILogger, error)
	ChildList() []ILogger
	GetChildLogger(name string) ILogger

	Fatal(params ...interface{})
	Fatalf(params ...interface{})
	Error(params ...interface{})
	Errorf(params ...interface{})
	Warn(params ...interface{})
	Warnf(params ...interface{})
	Info(params ...interface{})
	Infof(params ...interface{})
	Debug(params ...interface{})
	Debugf(params ...interface{})
	Trace(params ...interface{})
	Tracef(params ...interface{})
}

//full implement ILogger interface
type GoBLogger struct {
	ILogger
	level       base.LogLevel
	logName     string
	enabled     map[base.LogLevel]bool
	appender    appenders.Appender
	childrens   []ILogger
	parent      ILogger
	ExitOnFatal bool
}

// New a default debug level Logger and then reset some field.
// logName can set empty,logname use to differentiate and division logger and children logger
func NewGoBLogger(logName string) ILogger {
	obj := ILogger(&GoBLogger{
		level:       base.DEBUG,
		logName:     logName,
		enabled:     make(map[base.LogLevel]bool),
		appender:    appenders.NewConsoleAppender(),
		childrens:   make([]ILogger, 3),
		parent:      nil,
		ExitOnFatal: true,
	})
	obj.SetLevel(base.DEBUG)
	return obj
}

//New child logger ,level is base.INHERIT, appender inherit, parent point to upper
func (this *GoBLogger) NewChildLogger(logName string) (ILogger, error) {
	if strings.TrimSpace(logName) == "" {
		return nil, errors.New("logname is null.")
	}
	child := ILogger(&GoBLogger{
		level:     base.INHERIT,
		logName:   logName,
		enabled:   make(map[base.LogLevel]bool),
		appender:  nil,
		childrens: make([]ILogger, 0),
		parent:    this,
	})
	this.childrens = append(this.childrens, child)
	return child, nil
}

func (this *GoBLogger) Level() string {
	if this.level == base.INHERIT {
		return this.parent.Level()
	}
	return base.LogLevelIntMap[this.level]
}

func (this *GoBLogger) SetLevel(level base.LogLevel) {
	this.level = level
	for k := range base.LogLevelIntMap {
		if k <= level {
			this.enabled[k] = true
		} else {
			this.enabled[k] = false
		}
	}
}

func (this *GoBLogger) Children() []ILogger {
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
			return ap
		}
	}
	return nil
}
