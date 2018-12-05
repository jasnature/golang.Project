// httpProxyServe
package main

import (
	"bufio"
	"connProxy/base"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

var configMgr *base.ConfigManager

func init() {
	fmt.Println("httpProxyServe init")
	configMgr, _ = base.NewConfigManager()

}

type ProxyServer struct {
	config base.ProxyConfig

	linkingCount      int32
	totalEnterCounter int64
	curIpLink         map[string]int

	//start ip control
	allowIpMap map[string]string
	allowAllIp bool
	//end ip control
}

func (this *ProxyServer) initProxy() {
	this.wLog("ProxyServer init..")

	tempConfig, esl := configMgr.LoadConfig()
	//fmt.Println("load", esl)
	if esl == nil {
		fmt.Printf("\r\nload local xml config file[%s] init success!\r\n", configMgr.FileName)
		this.config = tempConfig
		//configMgr.SaveConfig(&this.config)
	} else {
		fmt.Println("cannot find local xml config, use default inner params init.")
	}

	//esl := configMgr.SaveConfig(&proxy.config)
	//fmt.Println("save", esl)

	this.allowIpMap = make(map[string]string, 5)
	this.curIpLink = make(map[string]int, 10)

	this.allowIpMap["."] = "def"
	this.allowIpMap["[::1]"] = "def"
	this.allowIpMap["localhost"] = "def"
	this.allowIpMap["127.0.0.1"] = "def"

	if this.config.AllowIpStr != "" {

		if strings.TrimSpace(this.config.AllowIpStr) == "*" {
			this.allowAllIp = true
			this.allowIpMap = nil
		} else {
			this.allowAllIp = false

			spstr := strings.Split(this.config.AllowIpStr, ",")
			for _, spitem := range spstr {
				this.allowIpMap[spitem] = spitem
			}
		}
	}

	if this.config.PrintIpSummary {

		go func() {

			for {
				time.Sleep(time.Second * 10)
				fmt.Printf("\r\nSum Process Count -> %d,Current Process Count-> %d,Current Link Address list-> %v", this.totalEnterCounter, this.linkingCount, this.curIpLink)

			}

		}()
	}

	if this.config.BuffSize <= 0 {
		this.config.BuffSize = 1024 * 16
	}

	if this.config.AllowMaxConn <= 0 {
		this.config.AllowMaxConn = 100
	}
}

func (this *ProxyServer) wLog(format string, a ...interface{}) {
	if this.config.PrintLog {
		if a != nil {
			fmt.Fprintf(os.Stdout, "\r\n"+format, a)
		} else {
			fmt.Fprintln(os.Stdout, format)
		}
	}
}

func (this *ProxyServer) wErrlog(a ...interface{}) {

	fmt.Fprintf(os.Stdout, "\r\n[Error]\r\n %s \r\n---------------------------", a)

}

func (this *ProxyServer) StartProxy() {
	this.initProxy()
	addrStr := strings.Trim(this.config.Addr, " ") + ":" + this.config.Port

	link, err := net.Listen("tcp", addrStr)

	defer link.Close()

	if err != nil {
		this.wErrlog("Listen link", err.Error())
	}
	fmt.Printf("\r\nlister success : %+v \r\naddress: %s \r\n", this, addrStr)

	for {

		conn, accerr := link.Accept()
		//fmt.Println("link->", conn.RemoteAddr())
		//this_proxy.wLog("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<processParams")
		result := this.processParams(conn, accerr)

		if !result {
			this.wLog("Deny %s ip enter.", conn.RemoteAddr().String())
			continue
		}

		this.wLog("handle conn info: %+v", conn.RemoteAddr().String())

		this.totalEnterCounter = atomic.AddInt64(&this.totalEnterCounter, 1)
		go this.handleConnection(conn, accerr)
	}

}

func (this *ProxyServer) handleConnection(clientConn net.Conn, err error) {
	defer func() {
		if p := recover(); p != nil {
			this.wErrlog("##Recover Info:##", p)
			errbuf := make([]byte, 1<<20)
			ernum := runtime.Stack(errbuf, false)
			this.wErrlog("##Recover Stack:##\r\n", string(errbuf[:ernum]))
		}
	}()
	//this_proxy.wLog("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<1")
	defer this.DeferCallClose(clientConn)
	clientConn.SetDeadline(time.Now().Add(time.Second * 30))

	var dialHost string
	var requestBuild *http.Request = nil

	if strings.TrimSpace(this.config.PassProxy) != "" {
		dialHost = this.config.PassProxy
	} else {

		bufread := bufio.NewReader(clientConn)
		requestBuild, err = http.ReadRequest(bufread)

		if err != nil {
			return
		}
		this.wLog("Dial proxy connection,host= %s,URL= %s", requestBuild.Host, requestBuild.URL)

		dialHost = requestBuild.Host

		if ppindex := strings.LastIndex(dialHost, ":"); ppindex < 0 {
			dialHost += ":80"
		}

		this.wLog("----------------%s", dialHost)
	}

	this.wLog("call DialTimeout:%s", dialHost)
	proDialConn, err := net.DialTimeout("tcp", dialHost, time.Second*10)
	this.linkingCount = atomic.AddInt32(&this.linkingCount, 1)
	if proDialConn != nil {
		proDialConn.SetDeadline(time.Now().Add(time.Second * 30))
	}

	if err != nil {
		this.wErrlog("proConn", err.Error())
		return
	}

	if requestBuild != nil && strings.TrimSpace(this.config.PassProxy) == "" {
		if requestBuild.Method == "CONNECT" {
			clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
			//_, err := io.WriteString(clientConn, )
			this.wLog("WriteString:%s", "HTTP/1.1 200 Connection Established\r\n")
			if err != nil {
				this.wLog("WriteString Error")
				return
			}
		} else {
			requestBuild.Write(proDialConn)
			this.wLog("WriteRequestHeaders")
		}
	}

	//this_proxy.wLog("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<2")
	var completedChan chan int = make(chan int)

	//if clientConn have new request then read clientConn write proDialConn
	go func() {

		var buf []byte = make([]byte, this.config.BuffSize)
		io.CopyBuffer(proDialConn, clientConn, buf)

		//		var temp []byte = make([]byte, this_proxy.config.BuffSize)
		//		for {
		//			n, e := clientConn.Read(temp)
		//			fmt.Println("temp:", n, e)
		//			if e == io.EOF || n <= 0 {
		//				break
		//			}
		//			proDialConn.Write(temp[:n])
		//			//this_proxy.wLog(string(temp[:n]))
		//		}
		this.wLog("read clientConn->write proDialConn end")
		completedChan <- 1
	}()

	//if proDialConn have new respone then read proDialConn write clientConn
	go func() {
		//this_proxy.wLog("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<4")

		var buf []byte = make([]byte, this.config.BuffSize)
		io.CopyBuffer(clientConn, proDialConn, buf)

		//		var temp []byte = make([]byte, this_proxy.config.BuffSize)
		//		for {
		//			n, e := proDialConn.Read(temp)
		//			fmt.Println("\r\ntemp:", n, e)
		//			if e == io.EOF || n <= 0 {
		//				break
		//			}
		//			clientConn.Write(temp[:n])
		//		}

		this.wLog("read proDialConn->write clientConn end")
		completedChan <- 1
	}()

	defer this.DeferCallClose(proDialConn)
	//this_proxy.wLog("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<5")
	var result int = 0
	for {
		result += <-completedChan
		this.wLog("<-completedChan=%d\r\n", result)
		if result >= 2 {
			this.linkingCount = atomic.AddInt32(&this.linkingCount, -1)
			close(completedChan)
			this.wLog("closed all channel Connection end,linkingCount = %d", this.linkingCount)
			break
		}
	}

	defer func(cr int) {
		if completedChan != nil {
			if cr < 2 {
				this.linkingCount = atomic.AddInt32(&this.linkingCount, -1)
				this.wLog("if not closed then again to close Chan. linkingCount = %d", this.linkingCount)
				close(completedChan)
			}
		}
	}(result)
}

func (this *ProxyServer) processParams(clientConn net.Conn, accerr error) bool {

	reip := clientConn.RemoteAddr().String()

	if reip == "" {
		go this.DeferCallClose(clientConn)
		return false
	}
	if accerr != nil {
		this.wErrlog("Accept conn", accerr.Error())
		go this.DeferCallClose(clientConn)
		return false
	}

	if !this.allowAllIp {
		i := strings.LastIndex(reip, ":")
		reip = reip[:i]
		//fmt.Println("link->", reip)
		if _, ok := this.allowIpMap[reip]; !ok {
			//fmt.Println("disallow->", reip)
			clientConn.Write([]byte("HTTP/1.1 403 Forbidden\r\n\r\n"))
			go this.DeferCallClose(clientConn)
			return false
		}
	}

	if count, ok := this.curIpLink[reip]; ok {
		this.curIpLink[reip] = count + 1
	} else {
		this.curIpLink[reip] = 1
	}

	return true
}

func (this *ProxyServer) DeferCallClose(closer io.Closer) {
	if closer != nil {
		//var me, _ = reflect.TypeOf(closer).MethodByName("RemoteAddr")
		this.wLog("Close call=%s", closer)
		closer.Close()
	}
}
