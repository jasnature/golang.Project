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
	"time"
)

/*
util toolkit to help quick call method collection.
*/
type util struct {
}

func (u *util) GetExecutePath() (string, error) {
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
func (u *util) PathOrFileExists(path string, mode int) (bool, error) {
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
func (u *util) TraceMethodInfo(funcname string, data ...interface{}) func() {
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
