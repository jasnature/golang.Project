// util
package base

import (
	"GoBLog"
	logbase "GoBLog/base"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var Log GoBLog.ILogger
var DefUtil *Util

func init() {
	if Log == nil {
		Log = GoBLog.DefaultLogFactory.GetLoggerByName("connLog", logbase.FileOutput)
		Log.SetLevel(logbase.DEBUG)
		fmt.Println("init log write..")
	}
	if DefUtil == nil {
		DefUtil = &Util{}
	}
}

/*
util toolkit to help quick call method collection.
*/
type Util struct {
}

func (u *Util) GetExecutePath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	spath, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	pathchar := "/"
	if runtime.GOOS == "windows" {
		pathchar = "\\"
	}
	si := strings.LastIndex(spath, pathchar)

	if si < 0 {
		return "", errors.New(`error:can't find "/" or "\" split path.`)
	}

	return string(spath[0 : si+1]), err
}

/*
PathOrFileExists

mode 0 path, >=1 path and file
*/
func (u *Util) PathOrFileExists(path string, mode int) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {

		//if double check of file,pls set mode >=1,either
		if mode >= 1 && strings.LastIndex(path, ".") < 0 {
			return false, errors.New("file is not exists.")
		}

		return true, nil
	}
	//	if os.IsNotExist(err) {
	//		return false, nil
	//	}
	return false, err
}

/*
traceInfo must use defer call this function and end part need add (),
etc: defer TraceMethodInfo("xxx",data1,data2,...)()
*/
func (u *Util) TraceMethodInfo(funcname string, data ...interface{}) func() {
	n := time.Now()
	fmt.Println("[Start record]:", funcname)

	if data != nil {
		for i, v := range data {
			fmt.Printf("\r\n%d-Before value:%+v \r\n", i+1, v)
		}
	}

	return func() {

		if data != nil {

			for i, v := range data {
				fmt.Printf("\r\n%d-After value:%+v \r\n", i+1, v)
			}
		}

		fmt.Println("\r\n[End record]: the trace method cost time= ", time.Since(n))

	}
}

// CopyBuffer is identical to Copy except that it stages through the
// provided buffer (if one is required) rather than allocating a
// temporary one. If buf is nil, one is allocated; otherwise if it has
// zero length, CopyBuffer panics.
// copyBuffer is the actual implementation of Copy and CopyBuffer.
// if buf is nil, one is allocated.
func (u *Util) CopyBufferForRollTimeout(dst net.Conn, src net.Conn, buf []byte, timeout time.Duration) (written int64, err error) {
	if buf != nil && len(buf) == 0 {
		panic("empty buffer in io.CopyBuffer")
	}

	if buf == nil {
		//32K
		size := 32 * 1024
		//if buf it is Limited Reader then buffer size full set.
		//		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
		//			if l.N < 1 {
		//				size = 1
		//			} else {
		//				size = int(l.N)
		//			}
		//		}
		buf = make([]byte, size)
	}
	var ticktime = timeout - (time.Second * 2)

	ticker := time.NewTicker(ticktime)

	defer ticker.Stop()

	var (
		er error
		nr int
	)
	for {
		select {
		case <-ticker.C:
			if er != nil {
				return written, err
			}
			src.SetDeadline(time.Now().Add(timeout))
			//dst.SetDeadline(time.Now().Add(timeout))
			fmt.Printf("NewTicker enter timeoutvalue=%d timedate=%s\r\n", ticktime/time.Second, time.Now().Add(timeout).String())
		default:
			{
				nr, er = src.Read(buf)
				if nr > 0 {
					nw, ew := dst.Write(buf[0:nr])
					if nw > 0 {
						written += int64(nw)
					}
					if ew != nil {
						err = ew
						return
					}
					if nr != nw {
						err = io.ErrShortWrite
						return
					}
				}
				if er != nil {
					err = er
					if er == io.EOF {
						err = nil
					}
					return
				}
			}
		}

	}
	return written, err
}
