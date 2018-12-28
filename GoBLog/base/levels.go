package base

type LogLevel int

const (
	None LogLevel = iota
	FATAL
	ERROR
	WARN
	INFO
	DEBUG
	TRACE
)

var LogLevelStringMap = map[string]LogLevel{
	"None":  None,
	"FATAL": FATAL,
	"ERROR": ERROR,
	"WARN":  WARN,
	"INFO":  INFO,
	"DEBUG": DEBUG,
	"TRACE": TRACE,
}

var LogLevelIntMap = map[LogLevel]string{
	None:  "None",
	FATAL: "FATAL",
	ERROR: "ERROR",
	WARN:  "WARN",
	INFO:  "INFO",
	DEBUG: "DEBUG",
	TRACE: "TRACE",
}
