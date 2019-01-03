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
	formatters.FormatterManager
	formatter formatters.Formatter

	MaxFileSize    int
	MaxBackupIndex int
	fileFullPath   string
	appendModel    bool
	bufferSize     int

	//Has been write bytes
	writtenBytes int
	buffer       []byte
	fileStream   *os.File
	mu           sync.Mutex
}

//filepath: filename or full path
//appendModel false: if set false with reopen file then clear content and write log,if have not close file stream then append log. true: always append log to file
func NewFileAppender(filepath string, appendModel bool, bufferSize int) (obj *FileAppender, err error) {
	err = nil

	obj = &FileAppender{
		formatter:      formatters.DefaultPatternFormatter(),
		MaxFileSize:    50,
		MaxBackupIndex: 50,
		fileFullPath:   filepath,
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

func (this *FileAppender) WriteString(level base.LogLevel, message string, args ...interface{}) {
	//fmt.Println(this.Formatter().Format(level, message, args...))
	m := this.Formatter().Format(level, message, args...)

	this.fileStream.Write([]byte(m))

	//this.writeMutex.Lock()
	//	this.bytesWritten += int64(len(m))
	//	if this.bytesWritten >= this.MaxFileSize {
	//		this.bytesWritten = 0
	//		this.rotateFile()
	//	}

	//	this.writeMutex.Unlock()
}

//func (this *FileAppender) rotateFile() {
//	this.closeFile()

//	lastFile := this.filename + "." + strconv.Itoa(this.MaxBackupIndex)
//	if _, err := os.Stat(lastFile); err == nil {
//		os.Remove(lastFile)
//	}

//	for n := this.MaxBackupIndex; n > 0; n-- {
//		f1 := this.filename + "." + strconv.Itoa(n)
//		f2 := this.filename + "." + strconv.Itoa(n+1)
//		os.Rename(f1, f2)
//	}

//	os.Rename(this.filename, this.filename+".1")

//	this.openFile()
//}

func (this *FileAppender) ResetFilename(filepath string) error {
	defer this.mu.Unlock()
	this.mu.Lock()
	if this.fileFullPath != filepath || this.fileStream == nil {
		err := this.closeFileStream()
		if err == nil {
			this.fileFullPath = filepath
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
	fs, err := os.OpenFile(this.fileFullPath, mode, 0666)
	this.fileStream = fs
	return err
}
