package appenders

import (
	"GoBLog/base"
	"GoBLog/formatters"
	"sync"

	//"fmt"
	"os"
)

type FileAppender struct {
	Appender
	formatter formatters.Formatter

	MaxFileSize    int
	MaxBackupIndex int
	fileName       string
	appendModel    bool
	bufferSize     int

	//Has been write bytes
	writtenBytes int
	buffer       []byte
	fileStream   *os.File
	mu           sync.Mutex
}

//filepath: filename or full path
//appendModel append log to file
func NewFileAppender(filepath string, appendModel bool, bufferSize int) (obj *FileAppender, err error) {
	err = nil

	obj = &FileAppender{
		formatter:      formatters.DefaultFormatter(),
		MaxFileSize:    50,
		MaxBackupIndex: 50,
		fileName:       filepath,
		appendModel:    appendModel,
		bufferSize:     bufferSize,
		writtenBytes:   0,
	}

	err = obj.ResetFilename(filepath)
	if err != nil {
		obj = nil
		return nil, err
	}

	if bufferSize <= 0 {
		//128KB
		bufferSize = 1024 * 128
	}

	obj.buffer = make([]byte, obj.bufferSize)

	return obj, err
}

func (this *FileAppender) Write(level base.LogLevel, message string, args ...interface{}) {
	//fmt.Println(this.Formatter().Format(level, message, args...))

}

func (this *FileAppender) ResetFilename(filepath string) error {
	defer this.mu.Unlock()
	this.mu.Lock()
	if this.fileName != filepath || this.fileStream == nil {
		err := this.closeFileStream()
		if err == nil {
			this.fileName = filepath
			err = this.openFileStream()
		}
		return err
	}
	return nil
}

func (this *FileAppender) Formatter() formatters.Formatter {
	return this.formatter
}

func (this *FileAppender) SetFormatter(formatter formatters.Formatter) {
	this.formatter = formatter
}

func (this *FileAppender) closeFileStream() (err error) {
	if this.fileStream != nil {
		err = this.fileStream.Close()
		if err == nil {
			this.fileStream = nil
		}
	}
	return err
}
func (this *FileAppender) openFileStream() error {
	mode := os.O_WRONLY | os.O_APPEND | os.O_CREATE
	if !this.appendModel {
		mode = os.O_WRONLY | os.O_CREATE
	}
	//4-r 2-w 1-x linux
	fs, err := os.OpenFile(this.fileName, mode, 0666)
	this.fileStream = fs
	return err
}
