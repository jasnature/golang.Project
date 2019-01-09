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

//standard txt log roll file of FileAppender,if hope use other type log,please used json jsonFileAppender/xmlFileAppender
//if set maxBackupDayIndex then auto roll flush and backup file,first index=1 it is latest backup,max index it is old backup.
type FileAppender struct {
	Appender
	base.IDispose
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
	//return NewFileCustomAppender(filepathAndName, appendModel, base.Byte*300, base.Byte*700, 3)
	return NewFileCustomAppender(filepathAndName, appendModel, base.KB*64, base.MB*6, 10)
}

//filepathAndName: filename or full path,if set "" then auto filename base on date name.
//appendModel false: if set false with reopen file then clear content and write log,if have not close file stream then append log. true: always append log to file
//bufferSize: write memory size if full then flush to disk.
//maxfilesize: sigle file max size.
//maxbakIndex: if > maxfilesize then auto backup to other name by day.
func NewFileCustomAppender(filepathAndName string, appendModel bool, bufferSize base.ByteSize, maxfilesize base.ByteSize, maxbakRollIndexByDay int) (obj *FileAppender, err error) {
	//fmt.Println("NewFileCustomAppender")
	err = nil
	if maxfilesize < 50 {
		maxfilesize = base.MB * 3
	}
	if maxbakRollIndexByDay <= 0 {
		maxbakRollIndexByDay = 20
	}
	if bufferSize <= 0 {
		//128KB
		bufferSize = 1024 * 128
	}
	obj = &FileAppender{
		formatter:         formatters.DefaultPatternFormatter(),
		maxFileSize:       int32(maxfilesize),
		maxBackupDayIndex: maxbakRollIndexByDay,
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

//write msg no lock model but rotateFile has lock,goBLogFactory will be use channel to control queue of write msg function.
func (this *FileAppender) WriteString(level base.LogLevel, message string, args ...interface{}) {
	msg := this.Formatter().Format(level, message, args...)
	//fmt.Println(msg)
	//len([]rune(msg)) this len is count num
	realBytes := []byte(msg)

	var lencount int32 = int32(len(realBytes))
	atomic.AddInt32(&this.writtenBytes, lencount)

	fmt.Printf("WriteString Print: current=%d total=%d max=%d \r\n", len(realBytes), this.writtenBytes, this.maxFileSize)

	_, err := this.bufferIO.WriteString(msg)
	fmt.Printf("->write file:%s error=%+v \r\n", this.currentfilePath, err)

	if this.writtenBytes >= this.maxFileSize {
		result := this.rotateFile()
		if !result {
			_, err = this.bufferIO.WriteString(msg)
			fmt.Printf("->wait has been rotate,write msg. error=%s \r\n", err)
		}
	}

}

func (this *FileAppender) rotateFile() bool {
	defer this.mu_lock.Unlock()
	this.mu_lock.Lock()

	if this.writtenBytes >= this.maxFileSize {
		fmt.Println("->rotate File")

		this.bufferIO.Flush()

		this.closeFileStream()
		defer this.openFileStream()

		dayname := base.DefaultUtil().NowTimeStr(2)

		var oldF, newF string
		for n := this.maxBackupDayIndex - 1; n > 0; n-- {

			if !this.logNameAuto {
				oldF = fmt.Sprintf(this.template_ByName, this.currentfileName, dayname, n, this.currentfileExt)
				newF = fmt.Sprintf(this.template_ByName, this.currentfileName, dayname, (n + 1), this.currentfileExt)
			} else {
				oldF = fmt.Sprintf(this.template_ByDate, this.currentfileName, n, this.currentfileExt)
				newF = fmt.Sprintf(this.template_ByDate, this.currentfileName, (n + 1), this.currentfileExt)
			}
			//fmt.Println("->rename= old:" + oldF + " new:" + newF)
			if err := os.Rename(oldF, newF); err != nil {
				v, ok := err.(*os.LinkError)
				if ok && strings.Contains(v.Error(), "Access is denied") {
					fmt.Printf("\r\n==============%v\r\n%+v", reflect.TypeOf(v), v.Error())
					os.Rename(oldF, newF+".rename")
				}
			}
		}
		var ccname string
		if !this.logNameAuto {
			ccname = fmt.Sprintf(this.template_ByName, this.currentfileName, dayname, 1, this.currentfileExt)
		} else {
			ccname = fmt.Sprintf(this.template_ByDate, this.currentfileName, 1, this.currentfileExt)
		}
		os.Rename(this.currentfilePath, ccname)

		defer atomic.StoreInt32(&this.writtenBytes, 0)

		fmt.Println("->rotate File end.")

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

func (this *FileAppender) Dispose() (err error) {
	if this != nil {
		err = this.bufferIO.Flush()
		fmt.Printf("Dispose Flush=%v \r\n", err)
		if err != nil {
			return err
		}
		err = this.closeFileStream()
		return err
	}
	return nil
}

func (this *FileAppender) closeFileStream() (err error) {
	if this.fileStream != nil {
		err = this.fileStream.Close()
		if err == nil {
			this.fileStream = nil
		}
		fmt.Printf("->closeFileStream,%s,err=%v \r\n", this.currentfilePath, err)
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
	if this.bufferIO != nil {
		this.bufferIO.Reset(this.fileStream)
	}
	fmt.Printf("->openFileStream,%s,err=%v \r\n", this.currentfilePath, err)
	return err
}
