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

	closerConnNotify chan string
}

func (this *ProxyServer) initProxy() {
	this.wLog("ProxyServer init..")

	tempConfig, esl := configMgr.LoadConfig()

	if esl == nil {
		fmt.Printf("\r\nload local xml config file[%s] init success!\r\n", configMgr.FileName)
		this.config = tempConfig
	} else {
		fmt.Println("cannot find local xml config, use default inner params init.")
	}

	//esl := configMgr.SaveConfig(&proxy.config)
	//fmt.Println("save", esl)

	this.allowIpMap = make(map[string]string, 5)
	this.curIpLink = make(map[string]int, 10)

	this.allowIpMap["."] = "1"
	this.allowIpMap["[::1]"] = "1"
	this.allowIpMap["localhost"] = "1"
	this.allowIpMap["127.0.0.1"] = "1"

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
				fmt.Printf("Sum Process Count -> %d,Current Process Count-> %d,Current Link Address list-> %v \r\n", this.totalEnterCounter, this.linkingCount, this.curIpLink)

			}

		}()
	}

	if this.config.BuffSize <= 0 {
		this.config.BuffSize = 1024 * 16
	}

	if this.config.AllowMaxConn <= 0 {
		this.config.AllowMaxConn = 100
	}

	this.closerConnNotify = make(chan string, int(this.config.AllowMaxConn/2))

	go func() {
		for removeIpPort := range this.closerConnNotify {
			this.wLog("removeIpPort= %s", removeIpPort)
			delete(this.curIpLink, removeIpPort)
		}
	}()
}

func (this *ProxyServer) wLog(format string, a ...interface{}) {
	if this.config.PrintLog {
		if a != nil {
			fmt.Fprintf(os.Stdout, "\r\n"+format+"\r\n", a)
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
	addrStr := ":" + this.config.Port

	link, err := net.Listen("tcp", addrStr)

	defer link.Close()

	if err != nil {
		this.wErrlog("Listen link", err.Error())
	}
	fmt.Printf("\r\nlister success : %+v \r\naddress: %s \r\n", this, addrStr)

	for {

		conn, accerr := link.Accept()
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
	this.linkingCount = atomic.AddInt32(&this.linkingCount, 1)
	defer func() {
		this.linkingCount = atomic.AddInt32(&this.linkingCount, -1)
		if p := recover(); p != nil {
			this.wErrlog("##Recover Info:##", p)
			errbuf := make([]byte, 1<<20)
			ernum := runtime.Stack(errbuf, false)
			this.wErrlog("##Recover Stack:##\r\n", string(errbuf[:ernum]))
		}
	}()

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
		this.wLog("Request Build,host= %s,URL= %s", requestBuild.Host, requestBuild.URL)

		dialHost = requestBuild.Host

		if ppindex := strings.LastIndex(dialHost, ":"); ppindex < 0 {
			dialHost += ":80"
		}
	}

	this.wLog("Call DialTimeout:%s", dialHost)
	proDialConn, err := net.DialTimeout("tcp", dialHost, time.Second*10)

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

	var result int = 0
	for {
		result += <-completedChan
		this.wLog("<-completedChan=%d", result)
		if result >= 2 {

			close(completedChan)
			this.wLog("closed all channel Connection end,linkingCount = %d", this.linkingCount)
			break
		}
	}

	defer func(re int) {
		if completedChan != nil {
			if re < 2 {
				this.wLog("if not closed then again to close Chan. linkingCount = %d", this.linkingCount)
				close(completedChan)
			}
		}
	}(result)
}

func (this *ProxyServer) processParams(clientConn net.Conn, accerr error) bool {

	reip_port := clientConn.RemoteAddr().String()

	if reip_port == "" {
		go this.DeferCallClose(clientConn)
		return false
	}
	if accerr != nil {
		this.wErrlog("Accept conn", accerr.Error())
		go this.DeferCallClose(clientConn)
		return false
	}

	if !this.allowAllIp {
		i := strings.LastIndex(reip_port, ":")
		reip := reip_port[:i]
		if _, ok := this.allowIpMap[reip]; !ok {
			this.wLog("disallow->%s", reip)
			clientConn.Write([]byte("HTTP/1.1 403 Forbidden  \r\nServer: JProxy-1.0 \r\nContent-Type: text/html \r\nConnection:keep-alive \r\nContent-Length: 13 \r\n\r\n Deny access."))
			go this.DeferCallClose(clientConn)
			return false
		}
	}

	if count, ok := this.curIpLink[reip_port]; ok {
		this.curIpLink[reip_port] = count + 1
	} else {
		this.curIpLink[reip_port] = 1
	}

	return true
}

func (this *ProxyServer) DeferCallClose(closer net.Conn) {
	if closer != nil {
		reip := closer.RemoteAddr().String()
		this.wLog("Close call=%s", reip)

		//if conn, ok := closer.(net.Conn); ok {
		this.closerConnNotify <- reip

		closer.Close()
	}
}
