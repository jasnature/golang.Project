// patternFormatter
package formatters

import (
	"GoBLog/base"
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type PatternFormatter struct {
	Formatter
	Pattern           string
	DefaultTimeLayout string
	reg               *regexp.Regexp
}

//New Pattern format struct,if not set then use default pattern
func NewPatternFormatter(pattern string) *PatternFormatter {
	pattern = strings.TrimRight(pattern, " ")
	//fmt.Printf("[old]=%s ,%v", pattern, strings.HasSuffix(pattern, "\n"))
	if !strings.HasSuffix(pattern, "\n") && !strings.HasSuffix(pattern, "%n") {
		pattern += " \n"
		//fmt.Printf(" [new]=%s", pattern)
	}

	return &PatternFormatter{
		Pattern:           string(pattern),
		DefaultTimeLayout: "2006-01-02 15:04:05.000",
		reg:               regexp.MustCompile("%(\\w|%)(?:{([^}]+)})?"),
	}
}

//%S - Stack info print
//%d or %d{golang Format string,e.g:2006-01-02 15:04:05.000} - The current date-time, using time.Now().Format("DefaultTimeLayout Field ")
//%F - The filename the log statement is in
//%l - The location of the log statement, e.g. file path : 12
//%L - The line number the log statement is on
//%m - The log message and its arguments formatted with fmt.Sprintf
//%n - A new-line character
//%p - Priority - the log level
func (this *PatternFormatter) Format(level base.LogLevel, message string, args ...interface{}) string {

	// TODO
	// %M - function name
	_, file, line, ok := runtime.Caller(2)

	if !ok {
		file = "not found file."
		line = 0
	}
	msg := this.reg.ReplaceAllStringFunc(this.Pattern, func(m string) string {
		//fmt.Println(m)
		parts := this.reg.FindStringSubmatch(m)
		//fmt.Printf("parts=%+v \n", parts)
		switch parts[1] {
		// FIXME
		// %c and %C should probably return the logger name, not the package
		// name, since that's how the logger is created in the first place!
		//		case "c":
		//			return caller.pkg
		//		case "C":
		//			return caller.pkg
		case "d":
			if len(parts) == 3 && strings.TrimSpace(parts[2]) != "" {
				return time.Now().Format(parts[2])
			}
			return time.Now().Format(this.DefaultTimeLayout)
		case "F":
			return file
		case "l":
			return fmt.Sprintf("%s : %d", file, line)
		case "L":
			return strconv.Itoa(line)
		case "m":
			return fmt.Sprintf(message, args...)
		case "n":
			switch runtime.GOOS {
			case "windows":
				return "\r\n"
			default:
				return "\n"
			}
		case "p":
			return base.LogLevelIntMap[level]
		case "S":
			errbuf := make([]byte, 1<<20)
			ernum := runtime.Stack(errbuf, false)
			return string(errbuf[:ernum])
		case "%":
			return "%"
		}
		return m
	})

	return msg
}
