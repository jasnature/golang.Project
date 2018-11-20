// httpProxyServe
package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	_ "reflect"
	"runtime"
	"strings"
	"time"

	"connProxy/base"
)

var configMgr *base.ConfigManager

func init() {
	fmt.Println("httpProxyServe init")
	configMgr, _ = base.NewConfigManager()

}

type ProxyServer struct {
	config base.ProxyConfig

	linkCount int
	curIpLink map[string]int

	//start ip control
	allowIpMap map[string]string
	allowAllIp bool
	//end ip control
}

func (proxy *ProxyServer) initProxy() {
	proxy.wLog("ProxyServer init..")

	tempConfig, esl := configMgr.LoadConfig()
	//fmt.Println("load", esl)
	if esl == nil {
		fmt.Printf("\r\nload local xml config file[%s] init success!\r\n", configMgr.FileName)
		proxy.config = tempConfig
		//configMgr.SaveConfig(&proxy.config)
	} else {
		fmt.Println("cannot find local xml config, use default inner params init.")
	}

	//esl := configMgr.SaveConfig(&proxy.config)
	//fmt.Println("save", esl)

	proxy.allowIpMap = make(map[string]string, 5)
	proxy.curIpLink = make(map[string]int, 10)

	proxy.allowIpMap["."] = "def"
	proxy.allowIpMap["[::1]"] = "def"
	proxy.allowIpMap["localhost"] = "def"
	proxy.allowIpMap["127.0.0.1"] = "def"

	if proxy.config.AllowIpStr != "" {

		if strings.TrimSpace(proxy.config.AllowIpStr) == "*" {
			proxy.allowAllIp = true
			proxy.allowIpMap = nil
		} else {
			proxy.allowAllIp = false

			spstr := strings.Split(proxy.config.AllowIpStr, ",")
			for _, spitem := range spstr {
				proxy.allowIpMap[spitem] = spitem
			}
		}
	}

	if proxy.config.PrintIpSummary {

		go func() {

			for {
				time.Sleep(time.Second * 10)
				fmt.Printf("\r\nSum Process Count -> %d,Current Link Address list-> %v", proxy.linkCount, proxy.curIpLink)

			}

		}()
	}

	if proxy.config.BuffSize <= 0 {
		proxy.config.BuffSize = 1024 * 16
	}
}

func (proxy *ProxyServer) wLog(format string, a ...interface{}) {
	if proxy.config.PrintLog {
		if a != nil {
			fmt.Fprintf(os.Stdout, "\r\n"+format, a)
		} else {
			fmt.Fprintln(os.Stdout, format)
		}
	}
}

func (proxy *ProxyServer) wErrlog(a ...interface{}) {

	fmt.Fprintf(os.Stdout, "\r\n[Error]\r\n %s \r\n---------------------------", a)

}

func (this_proxy *ProxyServer) StartProxy() {
	this_proxy.initProxy()
	addrStr := strings.Trim(this_proxy.config.Addr, " ") + ":" + this_proxy.config.Port

	link, err := net.Listen("tcp", addrStr)

	defer link.Close()

	if err != nil {
		this_proxy.wErrlog("Listen link", err.Error())
	}
	fmt.Printf("\r\nlister success : %+v \r\naddress: %s \r\n", this_proxy, addrStr)

	for {

		conn, accerr := link.Accept()
		//fmt.Println("link->", conn.RemoteAddr())
		//this_proxy.wLog("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<processParams")
		result := this_proxy.processParams(conn, accerr)

		if !result {
			this_proxy.wLog("Deny %s ip enter.", conn.RemoteAddr().String())
			continue
		}

		this_proxy.wLog("handle conn info: %+v", conn.RemoteAddr().String())
		this_proxy.linkCount++

		go this_proxy.handleConnection(conn, accerr)
	}

}

func (this_proxy *ProxyServer) handleConnection(clientConn net.Conn, err error) {
	defer func() {
		if p := recover(); p != nil {
			this_proxy.wErrlog("##Recover Info:##", p)
			errbuf := make([]byte, 1<<20)
			ernum := runtime.Stack(errbuf, false)
			this_proxy.wErrlog("##Recover Stack:##\r\n", string(errbuf[:ernum]))
		}
	}()
	//this_proxy.wLog("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<1")
	defer this_proxy.DeferCallClose(clientConn)
	clientConn.SetDeadline(time.Now().Add(time.Second * 30))

	bufread := bufio.NewReader(clientConn)
	request, err := http.ReadRequest(bufread)

	//request.Header.Set("Accept-Encoding", "none")
	//	if request != nil {
	//		this_proxy.wLog("\r\n handleConnection,%+v", *request)
	//	}
	if err != nil {
		return
	}
	this_proxy.wLog("Dial proxy connection,host= %s,URL= %s", request.Host, request.URL)
	host := request.Host

	if ppindex := strings.LastIndex(host, ":"); ppindex < 0 {
		host += ":80"
	}

	this_proxy.wLog("----------------%s", host)

	proDialConn, err := net.DialTimeout("tcp", host, time.Second*20)

	if proDialConn != nil {
		proDialConn.SetDeadline(time.Now().Add(time.Second * 60))
	}
	if err != nil {
		this_proxy.wErrlog("proConn", err.Error())
		return
	}

	if request.Method == "CONNECT" {
		clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		//_, err := io.WriteString(clientConn, )
		this_proxy.wLog("WriteString:%s", "HTTP/1.1 200 Connection Established\r\n")
		if err != nil {
			return
		}
	} else {
		request.Write(proDialConn)
	}
	//this_proxy.wLog("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<2")
	var completedChan chan int = make(chan int)

	//if clientConn have new request then read clientConn write proDialConn
	go func() {
		//this_proxy.wLog("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<3")
		var buf []byte = make([]byte, this_proxy.config.BuffSize)
		io.CopyBuffer(proDialConn, clientConn, buf)

		//		for {
		//			n, e := clientConn.Read(temp)
		//			fmt.Println("temp:", n, e)
		//			if e == io.EOF || n <= 0 {
		//				break
		//			}
		//			proDialConn.Write(temp[:n])
		//		}

		completedChan <- 1
	}()

	//if proDialConn have new respone then read proDialConn write clientConn
	go func() {
		//this_proxy.wLog("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<4")

		var buf []byte = make([]byte, this_proxy.config.BuffSize)
		io.CopyBuffer(clientConn, proDialConn, buf)

		//		var temp []byte = make([]byte, this_proxy.buffSize)
		//		for {
		//			n, e := proDialConn.Read(temp)
		//			fmt.Println("\r\ntemp:", n, e)
		//			if e == io.EOF || n <= 0 {

		//				break
		//			}
		//			//			stemp := string(temp[:n]) + "inject"
		//			//			stemp = strings.Replace(stemp, "1813", "1819", 1)
		//			//fmt.Println(string(temp[:n]))
		//			clientConn.Write(temp[:n])

		//		}

		completedChan <- 1
	}()

	defer this_proxy.DeferCallClose(proDialConn)
	//this_proxy.wLog("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<5")
	var result int = 0
	for {
		result += <-completedChan
		this_proxy.wLog("<-completedChan=%d", result)
		if result >= 2 {
			close(completedChan)
			this_proxy.wLog("  handleConnection end")
			break
		}
	}

	defer func(cr int) {
		if completedChan != nil {
			if cr < 2 {
				this_proxy.wLog("again close completedChan")
				close(completedChan)
			}
		}
	}(result)
}

func (this_proxy *ProxyServer) processParams(clientConn net.Conn, accerr error) bool {

	reip := clientConn.RemoteAddr().String()

	if reip == "" {
		go this_proxy.DeferCallClose(clientConn)
		return false
	}
	if accerr != nil {
		this_proxy.wErrlog("Accept conn", accerr.Error())
		go this_proxy.DeferCallClose(clientConn)
		return false
	}

	if !this_proxy.allowAllIp {
		i := strings.LastIndex(reip, ":")
		reip = reip[:i]
		//fmt.Println("link->", reip)
		if _, ok := this_proxy.allowIpMap[reip]; !ok {
			//fmt.Println("disallow->", reip)
			clientConn.Write([]byte("HTTP/1.1 403 Forbidden\r\n\r\n"))
			go this_proxy.DeferCallClose(clientConn)
			return false
		}
	}

	if count, ok := this_proxy.curIpLink[reip]; ok {
		this_proxy.curIpLink[reip] = count + 1
	} else {
		this_proxy.curIpLink[reip] = 1
	}

	return true
}

func (this_proxy *ProxyServer) DeferCallClose(closer io.Closer) {
	if closer != nil {
		//var me, _ = reflect.TypeOf(closer).MethodByName("RemoteAddr")
		this_proxy.wLog("Close call=%s", closer)
		closer.Close()
	}
}
