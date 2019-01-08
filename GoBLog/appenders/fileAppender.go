package appenders

import (
	"GoBLog/base"
	"GoBLog/formatters"
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

//standard txt log file of FileAppender,if hope use other type log,please used json jsonFileAppender/xmlFileAppender
type FileAppender struct {
	Appender
	formatters.FormatterManager
	formatter formatters.Formatter

	maxFileSize       int32
	maxBackupDayIndex int

	appendModel bool

	//Has been write bytes
	writtenBytes int32
	bufferIO     *bufio.Writer
	fileStream   *os.File
	mu_lock      sync.Mutex

	currentfilePath string
	currentfileName string
	currentfileExt  string
	template_ByName string
	template_ByDate string
	//auto set name base on date yyyy-MM-dd
	logNameAuto bool
}

//params see of NewFileCustomAppender function
func NewFileAppender(filepathAndName string, appendModel bool) (obj *FileAppender, err error) {
	//fmt.Println("NewFileAppender")
	return NewFileCustomAppender(filepathAndName, appendModel, base.Byte*300, base.Byte*700, 3)
	//return NewFileCustomAppender(filepathAndName, appendModel, base.KB*64, base.MB*6, 10)
}

//filepathAndName: filename or full path,if set "" then auto filename base on date name.
//appendModel false: if set false with reopen file then clear content and write log,if have not close file stream then append log. true: always append log to file
//bufferSize: write memory size if full then flush to disk.
//maxfilesize: sigle file max size.
//maxbakIndex: if > maxfilesize then auto backup to other name by day.
func NewFileCustomAppender(filepathAndName string, appendModel bool, bufferSize base.ByteSize, maxfilesize base.ByteSize, maxbakIndexByDay int) (obj *FileAppender, err error) {
	//fmt.Println("NewFileCustomAppender")
	err = nil
	if maxfilesize < 50 {
		maxfilesize = base.MB * 3
	}
	if maxbakIndexByDay <= 0 {
		maxbakIndexByDay = 20
	}
	if bufferSize <= 0 {
		//128KB
		bufferSize = 1024 * 128
	}
	obj = &FileAppender{
		formatter:         formatters.DefaultPatternFormatter(),
		maxFileSize:       int32(maxfilesize),
		maxBackupDayIndex: maxbakIndexByDay,
		currentfilePath:   filepathAndName,
		appendModel:       appendModel,
		writtenBytes:      0,
	}

	if strings.TrimSpace(filepathAndName) == "" {
		obj.currentfilePath = base.DefaultUtil().NowTimeStr(2) + ".log"
		obj.logNameAuto = true
	}

	err = obj.ResetFilename(obj.currentfilePath)
	if err != nil {
		obj = nil
		return nil, err
	}
	obj.currentfileName, obj.currentfileExt = base.DefaultUtil().GetFileNamAndExt(obj.currentfilePath)
	//fmt.Println(obj.currentfileName)
	obj.bufferIO = bufio.NewWriterSize(obj.fileStream, int(bufferSize))

	obj.template_ByName = "%s_%s_bak%d%s"
	obj.template_ByDate = "%s_bak%d%s"
	//fmt.Println("NewFileCustomAppender end")
	return obj, err
}

func (this *FileAppender) WriteString(level base.LogLevel, message string, args ...interface{}) {
	msg := this.Formatter().Format(level, message, args...)

	//len([]rune(msg)) this len is count num
	realBytes := []byte(msg)

	var lencount int32 = int32(len(realBytes))
	atomic.AddInt32(&this.writtenBytes, lencount)

	fmt.Printf("WriteString Print: current=%d total=%d max=%d \r\n", len(realBytes), this.writtenBytes, this.maxFileSize)

	if this.writtenBytes < this.maxFileSize {
		fmt.Println("->write file:" + this.currentfilePath)
		this.bufferIO.WriteString(msg)
	} else {
		if !this.rotateFile() {
			fmt.Println("->wait has been rotate,write msg.")
			this.bufferIO.WriteString(msg)
		}
	}

}

func (this *FileAppender) rotateFile() bool {
	defer this.mu_lock.Unlock()
	this.mu_lock.Lock()

	if this.writtenBytes >= this.maxFileSize {
		fmt.Println("->rotate File")

		this.bufferIO.Flush()
		atomic.AddInt32(&this.writtenBytes, -this.writtenBytes)

		this.closeFileStream()
		defer this.openFileStream()

		dayname := base.DefaultUtil().NowTimeStr(2)
		//removed expire file,(last)

		var lastFile string

		if !this.logNameAuto {
			lastFile = fmt.Sprintf(this.template_ByName, this.currentfileName, dayname, this.maxBackupDayIndex, this.currentfileExt)
		} else {
			lastFile = fmt.Sprintf(this.template_ByDate, this.currentfileName, this.maxBackupDayIndex, this.currentfileExt)
		}

		if _, err := os.Stat(lastFile); err == nil {
			fmt.Println("->deleted lastFile=" + lastFile)
			os.Remove(lastFile)
		}

		var oldF, newF string
		for n := this.maxBackupDayIndex - 1; n > 0; n-- {

			if !this.logNameAuto {
				oldF = fmt.Sprintf(this.template_ByName, this.currentfileName, dayname, n, this.currentfileExt)
				newF = fmt.Sprintf(this.template_ByName, this.currentfileName, dayname, (n + 1), this.currentfileExt)
			} else {
				oldF = fmt.Sprintf(this.template_ByDate, this.currentfileName, n, this.currentfileExt)
				newF = fmt.Sprintf(this.template_ByDate, this.currentfileName, (n + 1), this.currentfileExt)
			}
			fmt.Println("->rename= old:" + oldF + " new:" + newF)
			if err := os.Rename(oldF, newF); err != nil {
				v, _ := err.(*os.LinkError)
				fmt.Printf("\r\n==============%v\r\n%+v", reflect.TypeOf(v), v)

				os.Rename(oldF, newF+".rename")
			}
		}
		var ccname string
		if !this.logNameAuto {
			ccname = fmt.Sprintf(this.template_ByName, this.currentfileName, dayname, 1, this.currentfileExt)
		} else {
			ccname = fmt.Sprintf(this.template_ByDate, this.currentfileName, 1, this.currentfileExt)
		}
		os.Rename(this.currentfilePath, ccname)

		return true
	}
	return false
}

func (this *FileAppender) ResetFilename(filepath string) error {
	defer this.mu_lock.Unlock()
	this.mu_lock.Lock()
	if this.currentfilePath != filepath || this.fileStream == nil {
		err := this.closeFileStream()
		if err == nil {
			this.currentfilePath = filepath
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
	fs, err := os.OpenFile(this.currentfilePath, mode, 0666)
	this.fileStream = fs
	return err
}
