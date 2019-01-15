package base

type LogLevel int

const (
	OFF LogLevel = iota
	FATAL
	ERROR
	WARN
	INFO
	DEBUG
	TRACE
)

var LogLevelStringMap = map[string]LogLevel{
	"OFF":   OFF,
	"FATAL": FATAL,
	"ERROR": ERROR,
	"WARN":  WARN,
	"INFO":  INFO,
	"DEBUG": DEBUG,
	"TRACE": TRACE,
}

var LogLevelIntMap = map[LogLevel]string{
	OFF:   "OFF",
	FATAL: "FATAL",
	ERROR: "ERROR",
	WARN:  "WARN",
	INFO:  "INFO",
	DEBUG: "DEBUG",
	TRACE: "TRACE",
}

//INHERIT: "INHERIT",
