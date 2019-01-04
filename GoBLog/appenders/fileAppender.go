package appenders

import (
	"GoBLog/base"
	"GoBLog/formatters"
	"bufio"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
)

//standard txt log file of FileAppender,if hope use other type log,please used json jsonFileAppender/xmlFileAppender
type FileAppender struct {
	Appender
	formatters.FormatterManager
	formatter formatters.Formatter

	maxFileSize    int32
	maxBackupIndex int
	fileFullPath   string
	appendModel    bool

	//Has been write bytes
	writtenBytes int32
	bufferIO     *bufio.Writer
	fileStream   *os.File
	mu           sync.Mutex
}

//filepath: filename or full path
//appendModel false: if set false with reopen file then clear content and write log,if have not close file stream then append log. true: always append log to file
func NewFileAppender(filepathAndName string, appendModel bool) (obj *FileAppender, err error) {
	return NewFileCustomAppender(filepathAndName, appendModel, base.Byte*300, base.KB*2, 2)
	//return NewFileCustomAppender(filepathAndName, appendModel, base.KB*64, base.MB*6, 2)
}

func NewFileCustomAppender(filepathAndName string, appendModel bool, bufferSize base.ByteSize, maxfilesize base.ByteSize, maxIncreaseNameIndex int) (obj *FileAppender, err error) {
	err = nil
	if maxfilesize < 50 {
		maxfilesize = base.MB * 3
	}
	if maxIncreaseNameIndex <= 0 {
		maxIncreaseNameIndex = 20
	}
	if bufferSize <= 0 {
		//128KB
		bufferSize = 1024 * 128
	}
	obj = &FileAppender{
		formatter:      formatters.DefaultPatternFormatter(),
		maxFileSize:    int32(maxfilesize),
		maxBackupIndex: maxIncreaseNameIndex,
		fileFullPath:   filepathAndName,
		appendModel:    appendModel,
		writtenBytes:   0,
	}

	err = obj.ResetFilename(filepathAndName)
	if err != nil {
		obj = nil
		return nil, err
	}

	obj.bufferIO = bufio.NewWriterSize(obj.fileStream, int(bufferSize))

	return obj, err
}

func (this *FileAppender) WriteString(level base.LogLevel, message string, args ...interface{}) {
	msg := this.Formatter().Format(level, message, args...)

	//len([]rune(msg)) this len is count num
	realBytes := []byte(msg)

	var lencount int32 = int32(len(realBytes))
	atomic.AddInt32(&this.writtenBytes, lencount)

	fmt.Printf("WriteString Print: w= %d t=%d m=%d \r\n", len(realBytes), this.writtenBytes, this.maxFileSize)

	if this.writtenBytes < this.maxFileSize {
		fmt.Println("enter write")
		this.bufferIO.WriteString(msg)
	} else {
		fmt.Println("not write")
		this.bufferIO.Flush()
	}

	//	this.bytesWritten += int64(len(msg))
	//	if this.bytesWritten >= this.MaxFileSize {
	//		this.bytesWritten = 0
	//		this.rotateFile()
	//	}

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
