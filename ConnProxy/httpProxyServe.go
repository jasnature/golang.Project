// httpProxyServe
package main

import (
	"bufio"
	"connProxy/base"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

var configMgr *base.ConfigManager
var timeout_dur time.Duration

func init() {
	fmt.Println("httpProxyServe init")
	configMgr, _ = base.NewConfigManager()

}

type ProxyServer struct {
	config base.ProxyConfig

	linkingCount       int32
	totalAcceptCounter int64
	curIpLink          map[string]int

	wholeDelaySeconds int

	//s-ReversePorxy
	currentReverseProxy string
	reverseProxyMap     map[string]*base.Server
	//e-ReversePorxy

	//start ip control
	allowIpMap map[string]string
	allowAllIp bool
	//end ip control

	//make in buffer by allmaxconn
	enterConnectionNotify chan int
	//make out buffer by allmaxconn
	outConnectionNotify chan int
	//make buffer by allmaxconn
	closerConnNotify chan string

	//implement this method will call switch proxy
	ProxyBalancePlot func(servers []base.Server) string
}

func (this *ProxyServer) initProxy() {
	this.wLog("initProxy->")
	this.wholeDelaySeconds = 6
	tempConfig, esl := configMgr.LoadConfig()

	if esl == nil {
		fmt.Printf("\r\nLoad local xml config file[%s] init success!\r\n", configMgr.FileName)
		this.config = tempConfig
	} else {
		fmt.Println("Cannot find local xml config, use default inner params init.")
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

	if this.config.PrintSummary {

		go func() {

			for {
				time.Sleep(time.Second * time.Duration(this.wholeDelaySeconds))
				fmt.Println("\r\n============Print Summary Start==============\r\n")

				fmt.Printf("Sum Process Count -> %d,Current Process Count-> %d,Current Link Address list-> %v ", this.totalAcceptCounter, this.linkingCount, this.curIpLink)
				if this.reverseProxyMap != nil {
					for _, v := range this.reverseProxyMap {
						fmt.Printf("\r\nreverseProxyStatus-> %+v", *v)
					}
				}
				fmt.Println("\r\n\r\n============Print Summary End==============")
			}

		}()
	}

	if this.config.BuffSize <= 0 {
		this.config.BuffSize = 1024 * 16
	}

	if this.config.AllowMaxConn <= 0 {
		this.config.AllowMaxConn = 100
	}
	this.enterConnectionNotify = make(chan int, this.config.AllowMaxConn)

	this.outConnectionNotify = make(chan int, this.config.AllowMaxConn)

	if this.config.Timeout < 2 {
		this.config.Timeout = 10
	}
	timeout_dur = time.Second * time.Duration(this.config.Timeout)

	this.closerConnNotify = make(chan string, int(this.config.AllowMaxConn))

	go func() {
		for removeIpPort := range this.closerConnNotify {
			this.wLog("removeIpPort= %s", removeIpPort)
			delete(this.curIpLink, removeIpPort)
		}
	}()

	//reverse proxy config
	if this.config.ReverseProxys != nil {
		rp := this.config.ReverseProxys
		lenpp := len(rp.Servers)

		if lenpp > 1 {

			this.reverseProxyMap = make(map[string]*base.Server, lenpp)

			for i := 0; i < lenpp; i++ {
				t := rp.Servers[i]
				//this.wLog("this.reverseProxyMap= %+v", &t)
				this.reverseProxyMap[t.Addr] = &t
			}

			//this.wLog("this.reverseProxyMap= %+v", this.reverseProxyMap)
			go func() {
				//simple switch proxy or have any implement method
				for {
					if this.ProxyBalancePlot != nil {
						this.currentReverseProxy = this.ProxyBalancePlot(rp.Servers)
					} else {
						protemp := rand.Intn(lenpp)
						//this.wLog("protemp= %d", protemp)
						this.currentReverseProxy = rp.Servers[protemp].Addr
					}
					this.wLog("currentReverseProxy= %s", this.currentReverseProxy)

					time.Sleep(time.Second * time.Duration(this.wholeDelaySeconds))
				}
			}()

		} else if lenpp == 1 {
			this.currentReverseProxy = this.config.ReverseProxys.Servers[0].Addr
		} else {
			this.wLog("Notice:not found Reverse Proxys config,enable direct proxy model.")
		}
	}
}

func (this *ProxyServer) wLog(format string, a ...interface{}) {
	if this.config.PrintLog {
		if a != nil {
			fmt.Fprintf(os.Stdout, "\r\n"+format+"\r\n", a...)
		} else {
			fmt.Fprintln(os.Stdout, format)
		}
	}
}

func (this *ProxyServer) wErrlog(a ...interface{}) {

	fmt.Fprintf(os.Stdout, "\r\n[Error]\r\n %s \r\n---------------------------", a)

}

func (this *ProxyServer) StartProxy() {
	var (
		link net.Listener
		err  error
	)

	defer func() {
		link.Close()
		if p := recover(); p != nil {
			this.wErrlog("##Recover StartProxy Error:##", p)
			time.Sleep(time.Second * 5)
			this.wLog("Find Server exception Restart call StartProxy->")
			this.StartProxy()
		}
	}()

	this.wLog("StartProxy->")
	this.initProxy()
	addrStr := ":" + this.config.Port

	link, err = net.Listen("tcp", addrStr)

	if err != nil {
		this.wErrlog("Port has been used.", err.Error())
		return
	}

	fmt.Printf("\r\n[Lister success info]: %+v \r\n\r\n", this.config)

	go this.enterMaxConnControl()

	var currentWait *int32 = new(int32)
	*currentWait = 0
	for {

		conn, accerr := link.Accept()
		this.wLog("Accept conn: %s", conn.RemoteAddr().String())
		result := this.processParams(conn, accerr)

		if !result {
			this.wLog("Deny %s ip enter.", conn.RemoteAddr().String())
			continue
		}

		atomic.AddInt64(&this.totalAcceptCounter, 1)

		//max wait conn control
		if this.config.AllowMaxWait > 0 && this.config.AllowMaxWait < *currentWait {
			this.DeferCallClose(conn)
			this.wLog("overtake max wait,closed and continue.")
			continue
		}

		atomic.AddInt32(currentWait, 1)
		this.wLog("currentWait add=%d", *currentWait)

		//if have wait then auto handle thread timeout control
		go func() {
			select {
			case <-this.enterConnectionNotify:
				atomic.AddInt32(currentWait, -1)
				atomic.AddInt32(&this.linkingCount, 1)
				this.proxyConnectionHandle(conn)
			case <-time.After(timeout_dur):
				this.wLog("wait conn timeout: %s", conn.RemoteAddr().String())
				defer func() {
					atomic.AddInt32(currentWait, -1)
					this.wLog("currentWait release after=%d", *currentWait)
				}()
				defer this.DeferCallClose(conn)
			}
		}()

	}

}

//max connection control by chan buffer
func (this *ProxyServer) enterMaxConnControl() {
	var i int32
	for i = 0; i < this.config.AllowMaxConn; i++ {
		this.enterConnectionNotify <- 1
	}
	this.wLog("enter max connection control->init:%d", i)
	for {
		this.wLog("wait -> outConnectionNotify...")
		<-this.outConnectionNotify
		this.enterConnectionNotify <- 1
		this.wLog("readd -> enterConnectionNotify")
	}
}

func (this *ProxyServer) proxyConnectionHandle(clientConn net.Conn) {
	this.wLog("ConnectionHandle->: %s", clientConn.RemoteAddr().String())
	var err error
	defer func() {
		atomic.AddInt32(&this.linkingCount, -1)
		this.wLog("release one linkingCount:%d", this.linkingCount)
		this.outConnectionNotify <- 1
		if p := recover(); p != nil {
			if clientConn != nil {
				this.DeferCallClose(clientConn)
			}
			this.wErrlog("##Recover Info:##", p)
			errbuf := make([]byte, 1<<20)
			ernum := runtime.Stack(errbuf, false)
			this.wErrlog("##Recover Stack:##\r\n", string(errbuf[:ernum]))
		}
	}()

	defer this.DeferCallClose(clientConn)

	clientConn.SetDeadline(time.Now().Add(timeout_dur))

	var dialHost string
	var requestBuild *http.Request = nil

	var useReverseProxys bool = strings.TrimSpace(this.currentReverseProxy) != ""

	if useReverseProxys {
		dialHost = this.currentReverseProxy
		if v, ok := this.reverseProxyMap[dialHost]; ok {
			v.HandleSum++
		}
	} else {

		bufread := bufio.NewReader(clientConn)
		requestBuild, err = http.ReadRequest(bufread)

		if err != nil {
			return
		}
		this.wLog("Request Build,host= %s", requestBuild.Host)

		dialHost = requestBuild.Host

		if ppindex := strings.LastIndex(dialHost, ":"); ppindex < 0 {
			dialHost += ":80"
		}
	}

	this.wLog("Call DialByTimeout by:%s for clientip:%s", dialHost, clientConn.RemoteAddr().String())
	proDialConn, err := net.DialTimeout("tcp", dialHost, timeout_dur)

	if proDialConn != nil {
		proDialConn.SetDeadline(time.Now().Add(timeout_dur))
	}

	if err != nil {
		this.wErrlog("ClientIp=" + clientConn.RemoteAddr().String() + "  DialError:" + err.Error())
		if useReverseProxys {
			if v, ok := this.reverseProxyMap[dialHost]; ok {
				v.ErrorCount++
			}
		}
		return
	}

	defer this.DeferCallClose(proDialConn)

	//direct dial url
	if requestBuild != nil && !useReverseProxys {
		if requestBuild.Method == "CONNECT" {
			clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
			//_, err := io.WriteString(clientConn, )
			this.wLog("clientip=%s,WriteString:%s,", clientConn.RemoteAddr().String(), "Connection Established\r\n")
			if err != nil {
				this.wLog("WriteString Error")
				return
			}
		} else {
			requestBuild.Write(proDialConn)
			this.wLog("WriteRequestHeaders,clientip=%s", clientConn.RemoteAddr().String())
		}
	}

	var completedChan chan int = make(chan int)

	//if clientConn have new request then read clientConn write proDialConn
	go func() {

		var buf []byte = make([]byte, this.config.BuffSize)
		io.CopyBuffer(proDialConn, clientConn, buf)

		this.wLog("read clientConn->write proDialConn end,clientIP:%s", clientConn.RemoteAddr().String())
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

		this.wLog("read proDialConn->write clientConn end,clientIP:%s", clientConn.RemoteAddr().String())
		completedChan <- 1
	}()

	var result int = 0
	for {
		result += <-completedChan
		this.wLog("<-completedChan=%d", result)
		if result >= 2 {
			close(completedChan)
			this.wLog("closed all channel Connection end,clientIP=%s", clientConn.RemoteAddr().String())
			break
		}
	}

	//keep must be close chan
	defer func(re int) {
		if completedChan != nil {
			if re < 2 {
				close(completedChan)
				this.wLog("if not closed then again to close Chan,clientIP=%s", clientConn.RemoteAddr().String())
			}
		}
	}(result)
}

func (this *ProxyServer) processParams(clientConn net.Conn, accerr error) bool {
	this.wLog("processParams->")
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
	defer func() {
		if p := recover(); p != nil {
			this.wErrlog("##DeferCallClose Recover Info:##", p)
		}
	}()
	if closer != nil {
		reip := closer.RemoteAddr().String()
		this.wLog("Close call=%s", reip)

		//if conn, ok := closer.(net.Conn); ok {
		this.closerConnNotify <- reip

		closer.Close()
	}
}
