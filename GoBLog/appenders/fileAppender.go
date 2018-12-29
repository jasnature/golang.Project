package appenders

import (
	"GoBLog/base"
	"GoBLog/formatters"

	//"fmt"
	"os"
)

type FileAppender struct {
	Appender
	formatter formatters.Formatter

	MaxFileSize    int
	MaxBackupIndex int

	currentWFileName string
	fileStream       *os.File
	appendModel      bool
	writeSycnChan    chan string

	bufferSize int
	//Has been write bytes
	writtenBytes int

	buffer []byte
}

func NewFileAppender(fileName string, appendModel bool, bufferSize int) *FileAppender {

	this := &FileAppender{
		formatter:      formatters.DefaultFormatter(),
		MaxFileSize:    50,
		MaxBackupIndex: 50,
		appendModel:    appendModel,
		writeSycnChan:  make(chan string, 100),
		bufferSize:     bufferSize,
	}
	if bufferSize <= 0 {
		//128KB
		bufferSize = 1024 * 128
	}

	this.buffer = make([]byte, this.bufferSize)

	return this
}

func (this *FileAppender) Write(level base.LogLevel, message string, args ...interface{}) {
	//fmt.Println(this.Formatter().Format(level, message, args...))

}

func (this *FileAppender) closeFile() {
	if this.fileStream != nil {
		this.fileStream.Close()
		this.fileStream = nil
	}
}
func (this *FileAppender) openFile() error {
	mode := os.O_WRONLY | os.O_APPEND | os.O_CREATE
	if !this.appendModel {
		mode = os.O_WRONLY | os.O_CREATE
	}
	//4-r 2-w 1-x linux
	f, err := os.OpenFile(this.currentWFileName, mode, 0666)
	this.fileStream = f
	return err
}

func (this *FileAppender) Formatter() formatters.Formatter {
	return this.formatter
}

func (this *FileAppender) SetFormatter(formatter formatters.Formatter) {
	this.formatter = formatter
}
