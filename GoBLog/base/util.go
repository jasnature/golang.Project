// util
package base

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

/*
util toolkit to help quick call method collection.
*/
var mu sync.Mutex

type Util struct {
}

var instanceUtil *Util = nil

func DefaultUtil() *Util {

	if instanceUtil == nil {

		mu.Lock()
		if instanceUtil == nil {
			instanceUtil = &Util{}
		}
		defer mu.Unlock()
	}
	return instanceUtil
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

func (u *Util) NowTimeStr() string {
	nowTime := time.Now().String()
	return nowTime[:19]
}