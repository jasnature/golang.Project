package appenders

import (
	"GoBLog/base"
	"GoBLog/formatters"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"time"
)

//standard txt log roll file of FileAppender,if hope use other type log,please used json jsonFileAppender/xmlFileAppender
//if set maxBackupDayIndex then auto roll flush and backup file,first index=1 it is latest backup,max index it is old backup.
type FileAppender struct {
	AppenderBase

	maxFileSize       int64
	maxBackupDayIndex int

	appendModel bool

	//Has been write bytes
	writtenBytes     int64
	bufferIO         *bufio.Writer
	fileStream       *os.File
	fileLastOpenTime time.Time

	currentfilePath string
	currentfileName string
	currentfileExt  string
	template_ByName string
	template_ByDate string
	//auto set name base on date yyyy-MM-dd
	logNameAuto bool

	tickCheckTimer     *time.Ticker
	autoFlushDuration  time.Duration
	notifyFlushChan    chan byte
	notifyContinueChan chan byte
	isFlushing         bool
	isWriteing         bool
}

//params see of NewFileCustomAppender function
func NewFileAppender(filepathAndName string, appendModel bool) (obj *FileAppender, err error) {
	//fmt.Println("NewFileAppender")
	//return NewFileCustomAppender(filepathAndName, appendModel, base.Byte*300, base.KB*60, 3)

	return NewFileCustomAppender(filepathAndName, appendModel, base.KB*256, base.MB*6, 10)
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
		maxFileSize:        int64(maxfilesize),
		maxBackupDayIndex:  maxbakRollIndexByDay,
		currentfilePath:    filepathAndName,
		appendModel:        appendModel,
		writtenBytes:       0,
		autoFlushDuration:  time.Second * 50,
		notifyFlushChan:    make(chan byte),
		notifyContinueChan: make(chan byte),
	}

	obj.formatter = formatters.DefaultPatternFormatter()
	obj.bufferChan = make(chan string, 1000)
	if strings.TrimSpace(filepathAndName) == "" {
		obj.currentfilePath = base.DefaultUtil().NowTimeStr(time.Now(), 2) + ".log"
		obj.logNameAuto = true
	}

	if !filepath.IsAbs(obj.currentfilePath) {
		obj.currentfilePath, _ = filepath.Abs(obj.currentfilePath)
	}

	obj.currentfileName, obj.currentfileExt = base.DefaultUtil().GetFileNamAndExt(obj.currentfilePath)
	if strings.TrimSpace(obj.currentfileExt) == "" {
		obj.currentfileExt = ".log"
		obj.currentfilePath += ".log"
	}

	err = obj.ResetFilename(obj.currentfilePath)
	if err != nil {
		obj = nil
		return nil, err
	}

	//fmt.Printf("currentfilePath=%s | %s | %s \r\n", obj.currentfilePath, obj.currentfileName, obj.currentfileExt)

	obj.bufferIO = bufio.NewWriterSize(obj.fileStream, int(bufferSize))

	obj.template_ByName = "%s_%s_bak%d%s"
	obj.template_ByDate = "%s_bak%d%s"
	//fmt.Println("NewFileCustomAppender end")
	go obj.processWriteChan()
	obj.isDispose = false

	obj.tickCheckTimer = time.NewTicker(obj.autoFlushDuration)
	go obj.tickerAutoCheck()
	obj.isFlushing = false

	return obj, err
}

//write use channel control for msg and rotateFile has lock
func (this *FileAppender) WriteString(level base.LogLevel, location string, dtime time.Time, message string, args ...interface{}) {
	if !this.isDispose {
		//fmt.Printf("FileAppender->WriteString->writeStringChan %s \r\n", message)
		this.bufferChan <- this.Formatter().Format(level, location, dtime, message, args...)
	}
}

//It is sync method
func (this *FileAppender) processWriteChan() {

	for msg := range this.bufferChan {
		this.isWriteing = true

		//auto flush
		if this.isFlushing {
			this.isFlushing = false
			//fmt.Println("wait auto flush...")
			this.notifyFlushChan <- 1
			select {
			case <-time.Tick(time.Second * 15): //timeout
			case <-this.notifyContinueChan:
			}
			//fmt.Println("wait flush end...")
		}

		//fmt.Println(msg)
		//len([]rune(msg)) this len is count num
		realBytes := []byte(msg)

		var lencount int64 = int64(len(realBytes))
		atomic.AddInt64(&this.writtenBytes, lencount)

		//fmt.Printf("WriteString Print: current=%d total=%d max=%d \r\n", len(realBytes), this.writtenBytes, this.maxFileSize)

		this.bufferIO.WriteString(msg)
		//_, err :=
		//fmt.Printf("->Write bufferIO:%s error=%+v \r\n", this.currentfilePath, err)

		if this.writtenBytes >= this.maxFileSize {
			result := this.rotateFile()
			if !result {
				this.bufferIO.WriteString(msg)
				//_, err =
				//fmt.Printf("->Wait has been rotate,write msg. error=%s \r\n", err)
			}
		}
		//time.Sleep(time.Millisecond * 500)

		this.isWriteing = false
	}
}

func (this *FileAppender) tickerAutoCheck() {
	//fmt.Println("->tickerAutoCheck init")
	var bufcount int = 0

	for _ = range this.tickCheckTimer.C {
		bufcount = this.bufferIO.Buffered()
		if bufcount > 0 && (this.isWriteing || len(this.bufferChan) > 0) {
			//fmt.Println("->notify write chan will be call Flush...")
			this.isFlushing = true
			select {
			case <-time.Tick(time.Second * 15): //timeout
			case <-this.notifyFlushChan:
			}
			this.bufferIO.Flush()
			//fmt.Println("->notify write and call Flush end")
			this.notifyContinueChan <- 1
			this.isFlushing = false

		} else if bufcount > 0 {
			this.bufferIO.Flush()
			//fmt.Println("->direct call Flush()")
		}
		//day auto change
		if time.Now().Day() != this.fileLastOpenTime.Day() {
			this.rotateFile()
			//fmt.Println("->day check enter....")
		}
		//fmt.Println("->tickerAutoCheck call")
	}
}

func (this *FileAppender) rotateFile() bool {
	defer this.mu_lock.Unlock()
	this.mu_lock.Lock()

	if (this.writtenBytes >= this.maxFileSize) || (time.Now().Day() != this.fileLastOpenTime.Day()) {
		//fmt.Println("->rotate File start...")

		this.bufferIO.Flush()

		this.closeFileStream()
		defer this.openFileStream()

		dayname := base.DefaultUtil().NowTimeStr(this.fileLastOpenTime, 2)

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

		defer atomic.StoreInt64(&this.writtenBytes, 0)

		//fmt.Println("->rotate File end.")

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
	defer this.mu_lock.Unlock()
	this.mu_lock.Lock()

	if this != nil && !this.isDispose {
		this.isDispose = true
		this.tickCheckTimer.Stop()
		for try := 10; try > 0; try-- {
			time.Sleep(time.Millisecond * 200)
			if len(this.bufferChan) <= 0 && this.bufferIO.Buffered() <= 0 {
				//fmt.Printf("FileAppender writeStringChan len=%d,\r\n", len(this.writeStringChan))
				break
			}
			//fmt.Printf("FileAppender Dispose try=%d,\r\n", try)
			err = this.bufferIO.Flush()
		}
		err = this.bufferIO.Flush()
		//fmt.Printf("Dispose Flush=%v \r\n", err)
		if err != nil {
			return err
		}
		err = this.closeFileStream()
		close(this.bufferChan)
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
		//fmt.Printf("->closeFileStream,%s,err=%v \r\n", this.currentfilePath, err)
	}
	return err
}

func (this *FileAppender) openFileStream() error {
	mode := os.O_WRONLY | os.O_APPEND | os.O_CREATE
	if !this.appendModel {
		mode = os.O_WRONLY | os.O_CREATE
	}

	if ok, _ := base.DefaultUtil().PathOrFileExists(this.currentfilePath, 0); !ok {
		//fmt.Println("OpenFile not exists:" + this.currentfilePath)
		os.MkdirAll(filepath.Dir(this.currentfilePath), 0666)
	}
	//4-r 2-w 1-x linux
	fs, err := os.OpenFile(this.currentfilePath, mode, 0666)

	this.fileStream = fs
	if this.bufferIO != nil {
		this.bufferIO.Reset(this.fileStream)
	}
	finfo, _ := fs.Stat()
	if finfo != nil {
		atomic.StoreInt64(&this.writtenBytes, finfo.Size())
	}
	this.fileLastOpenTime = time.Now()
	//fmt.Printf("->openFileStream,%s,size=%d,err=%v fileLastOpenTime=%v\r\n", this.currentfilePath, this.writtenBytes, err, this.fileLastOpenTime)
	return err
}
