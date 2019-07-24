// httpProxyServe
package main

import (
	logbase "GoBLog/base"
	"bufio"
	"connProxy/base"
	"connProxy/protocol"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

var configMgr *base.ConfigManager

//calc second result of timeout
var timeout_seconds_dur time.Duration

const ipRangeMapName = "IPRangeMap"

func init() {
	fmt.Println("httpProxyServe init")
	base.Log.Info("httpProxyServe init")
	configMgr, _ = base.NewConfigManager()
}

/*
An proxy server that support http,https,socket5 protocol proxy and reverse proxy and more infomation.
*/
type ProxyServer struct {
	config base.ProxyConfig

	linkListenerMap map[string]net.Listener

	socketHandle *protocol.SocketParser

	linkingCount       int32
	totalAcceptCounter int64
	curIpLink          map[string]int

	wholeDelaySeconds int

	//s-ReversePorxy
	currentReverseProxy string
	reverseProxyMap     map[string]*base.Server
	//e-ReversePorxy

	//start ip control
	allowIpMap map[string][]base.IpRange
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
	this.wholeDelaySeconds = 6
	tempConfig, esl := configMgr.LoadConfig()

	if esl == nil {
		fmt.Printf("\r\nLoad local xml config file[%s] init success!\r\n", configMgr.FileName)
		base.Log.Warnf("Load local xml config file[%s] init success!", configMgr.FileName)
		this.config = tempConfig
	} else {
		fmt.Println("Cannot find local xml config, use default inner params init.")
		base.Log.Warnf("Cannot find local xml config, use default inner params init. err=%v", esl)
	}
	if this.config.LogLevel != "" {
		val, ok := logbase.LogLevelStringMap[this.config.LogLevel]
		if ok {
			fmt.Printf("Log Level=%s\r\n", this.config.LogLevel)
			base.Log.SetLevel(val)
		}
	}
	this.linkListenerMap = make(map[string]net.Listener, 2)
	this.socketHandle = &protocol.SocketParser{}
	this.socketHandle.Config = &this.config.Socket
	//esl := configMgr.SaveConfig(&proxy.config)
	//fmt.Println("save", esl)

	this.allowIpMap = make(map[string][]base.IpRange, 5)
	this.curIpLink = make(map[string]int, 10)

	this.allowIpMap["."] = nil
	this.allowIpMap["[::1]"] = nil
	this.allowIpMap["localhost"] = nil
	this.allowIpMap["127.0.0.1"] = nil

	if this.config.AllowIpStr != "" {

		if strings.TrimSpace(this.config.AllowIpStr) == "*" {
			this.allowAllIp = true
			this.allowIpMap = nil
		} else {
			this.allowAllIp = false

			spstr := strings.Split(this.config.AllowIpStr, ",")
			var ranglist []base.IpRange

			for _, spitem := range spstr {
				if strings.Index(spitem, "-") > 0 {
					//ip range
					rangitem := strings.Split(spitem, "-")
					ranglist = append(ranglist, base.IpRange{
						Start: net.ParseIP(rangitem[0]),
						End:   net.ParseIP(rangitem[1]),
					})
				} else {
					//single ip
					this.allowIpMap[spitem] = nil
				}
			}
			if ranglist != nil && len(ranglist) > 0 {
				this.allowIpMap[ipRangeMapName] = ranglist
				//fmt.Printf("IPRangeMap,%+v \r\n", ranglist)
			}
		}
	}

	if this.config.PrintSummary {

		go func() {

			for {
				time.Sleep(time.Second * time.Duration(this.wholeDelaySeconds))
				fmt.Println("\r\n============Print Summary Start==============")

				fmt.Printf("Sum Process Sum:-> %d ,Current Process Count:-> %d ,Current Link Address list:-> %v ", this.totalAcceptCounter, this.linkingCount, this.curIpLink)
				fmt.Printf("\r\nAvailable Conn Count:-> %d", len(this.enterConnectionNotify))
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

	if this.config.TimeoutModel == "" {
		this.config.TimeoutModel = "auto"
	} else {
		this.config.TimeoutModel = strings.ToLower(this.config.TimeoutModel)
	}

	if this.config.Timeout < 3 {
		this.config.Timeout = 10
	}
	timeout_seconds_dur = time.Second * time.Duration(this.config.Timeout)

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
			base.Log.Info("Notice:not found Reverse Proxys config,enable direct proxy model.")
		}
	}
}

func (this *ProxyServer) wLog(format string, a ...interface{}) {
	if this.config.PrintConsoleLog {
		if a != nil {
			fmt.Fprintf(os.Stdout, "\r\n["+time.Now().String()[:23]+"] "+format+"\r\n", a...)
		} else {
			fmt.Fprintln(os.Stdout, "["+time.Now().String()[:23]+"] "+format)
		}
	}
}

func (this *ProxyServer) wErrlog(a ...interface{}) {

	fmt.Fprintf(os.Stdout, "\r\n ["+time.Now().String()[:23]+"][Error]\r\n %s \r\n---------------------------", a)

}

func (this *ProxyServer) StartProxy() {
	var (
		err error
	)

	defer func() {
		if p := recover(); p != nil {

			base.Log.Errorf("Recover StartProxy Error:%s", p)
			this.wErrlog("##Recover StartProxy Error:##", p)

			time.Sleep(time.Second * 5)
			this.wLog("Find Server exception Restart call StartProxy->")
			this.StartProxy()
		}

		if this.linkListenerMap != nil {
			this.Dispose()
		}
	}()

	this.wLog("StartProxy->")
	this.initProxy()

	addrStr := ":" + this.config.Port
	this.wLog("[Print config] %+v", this.config)
	base.Log.Debugf("[Print config] %+v", this.config)

	go this.enterMaxConnControl()
	switch strings.ToLower(this.config.Prototype) {

	case "http":
		this.AcceptTcpRequest(addrStr, 1, err)
	case "socket":
		this.AcceptTcpRequest(addrStr, 2, err)
	default:
		ports := strings.Split(this.config.Port, ",")
		go this.AcceptTcpRequest(":"+ports[1], 2, err)
		this.AcceptTcpRequest(":"+ports[0], 1, err)
	}

}

//protype 1 web 2 socket
func (this *ProxyServer) AcceptTcpRequest(addrStr string, protype byte, err error) {

	linkListener, err := net.Listen("tcp", addrStr)
	this.linkListenerMap[addrStr] = linkListener

	if err != nil {
		this.wErrlog("Port has been used.", err.Error())
		base.Log.Errorf("Port has been used:%s", err)
		return
	}

	fmt.Printf("\r\n[Lister success info]:protype: %d %+v \r\n\r\n", protype, addrStr)
	base.Log.Infof("[Lister success info]: %d %+v", addrStr, addrStr)

	var currentWait *int32 = new(int32)
	*currentWait = 0
	for {

		conn, accerr := linkListener.Accept()
		if conn == nil {
			continue
		}
		this.wLog("Accept conn: %s", conn.RemoteAddr().String())
		base.Log.Debugf("Accept conn: %s", conn.RemoteAddr().String())

		result := this.processParams(conn, accerr)

		if !result {
			this.wLog("Deny %s ip enter.", conn.RemoteAddr().String())
			base.Log.Debugf("Deny %s ip enter.", conn.RemoteAddr().String())
			continue
		}

		atomic.AddInt64(&this.totalAcceptCounter, 1)

		//max wait conn control
		if this.config.AllowMaxWait > 0 && this.config.AllowMaxWait < *currentWait {
			this.DeferCallClose(conn)
			this.wLog("overtake max wait,closed and continue.")
			base.Log.Debug("overtake max wait,closed and continue.")
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
				this.proxyConnectionHandle(conn, protype)
			case <-time.After((timeout_seconds_dur + (time.Second * 5))): //add 5 seconds for wait conn
				this.wLog("wait conn timeout: %s", conn.RemoteAddr().String())
				base.Log.Debugf("wait conn timeout: %s", conn.RemoteAddr().String())
				defer func() {
					atomic.AddInt32(currentWait, -1)
					this.wLog("currentWait release after=%d", *currentWait)
					base.Log.Debugf("currentWait release after=%d", *currentWait)
				}()
				defer this.DeferCallClose(conn)
			}
		}()

	}

}

func (this *ProxyServer) processParams(clientConn net.Conn, accerr error) bool {

	reip_port := clientConn.RemoteAddr().String()

	if reip_port == "" {
		go this.DeferCallClose(clientConn)
		return false
	}
	if accerr != nil {
		this.wErrlog("Accept conn", accerr.Error())
		base.Log.Warnf("Accept conn error:%s", accerr.Error())
		go this.DeferCallClose(clientConn)
		return false
	}

	if !this.allowAllIp {

		reip := reip_port[:strings.LastIndex(reip_port, ":")]

		if _, ok := this.allowIpMap[reip]; !ok {

			iprangResult := false

			if rangMap, ok := this.allowIpMap[ipRangeMapName]; ok {
				ipv4 := net.ParseIP(reip)
				for _, iprange := range rangMap {
					if base.DefUtil.CheckIpInRange(ipv4, iprange.Start, iprange.End) {
						iprangResult = true
						//fmt.Printf("CheckIpInRange result:%v checkip:%v \r\n", iprangResult, ipv4)
						base.Log.Debugf("Find IP allow range->%s rang->%v", reip, iprange)
						break
					}
				}
			}

			if !iprangResult {
				this.wLog("disallow->%s", reip)
				base.Log.Infof("Disallow IP enter->%s", reip)
				clientConn.Write([]byte("HTTP/1.1 403 Forbidden  \r\nServer: JProxy-1.0 \r\nContent-Type: text/html \r\nConnection:keep-alive \r\nContent-Length: 13 \r\n\r\n Deny access."))
				go this.DeferCallClose(clientConn)
				return false
			}
		}
	}

	if count, ok := this.curIpLink[reip_port]; ok {
		this.curIpLink[reip_port] = count + 1
	} else {
		this.curIpLink[reip_port] = 1
	}

	return true
}

//max connection control by chan buffer
func (this *ProxyServer) enterMaxConnControl() {
	var i int32
	//fill full can be enter
	for i = 0; i < this.config.AllowMaxConn; i++ {
		this.enterConnectionNotify <- 1
	}
	this.wLog("enter max connection control->init:%d", i)
	base.Log.Debugf("enter max connection control->init:%d", i)
	for {
		//this.wLog("wait -> outConnectionNotify...")
		<-this.outConnectionNotify
		this.enterConnectionNotify <- 1
		//this.wLog("readd -> enterConnectionNotify")
	}
}

//protype 1 web 2 socket
func (this *ProxyServer) proxyConnectionHandle(clientConn net.Conn, protype byte) {
	//v, ok := clientConn.(io.Seeker)
	this.wLog("ConnectionHandle->: %s", clientConn.RemoteAddr().String())
	base.Log.Debugf("ConnectionHandle->: %s", clientConn.RemoteAddr().String())
	var err error
	defer func() {
		atomic.AddInt32(&this.linkingCount, -1)
		this.wLog("release one linkingCount:%d", this.linkingCount)
		base.Log.Debugf("release one linkingCount:%d", this.linkingCount)
		this.outConnectionNotify <- 1
		if p := recover(); p != nil {
			if clientConn != nil {
				this.DeferCallClose(clientConn)
			}
			this.wErrlog("##Recover Info:##", p)
			//errbuf := make([]byte, 1<<20)
			//ernum := runtime.Stack(errbuf, false)
			//this.wErrlog("##Recover Stack:##\r\n", string(errbuf[:ernum]))

			base.Log.Errorf("Recover Info:%s Recover Stack:%s", p)
		}
	}()

	defer this.DeferCallClose(clientConn)

	clientConn.SetDeadline(time.Now().Add(timeout_seconds_dur))

	var dialHost string
	var requestBuild *http.Request = nil

	var useReverseProxys bool = strings.TrimSpace(this.currentReverseProxy) != ""

	if useReverseProxys {
		//auto random if have some server.
		dialHost = this.currentReverseProxy
		if v, ok := this.reverseProxyMap[dialHost]; ok {
			v.HandleSum++
		}
	} else {

		//1 web 2 socket
		switch protype {

		case 1:

			//Parse HTTP request url
			bufread := bufio.NewReader(clientConn)
			requestBuild, err = http.ReadRequest(bufread)

			if err != nil {
				this.wLog("Parse http error ->: %s %s", clientConn.RemoteAddr().String(), err)
				base.Log.Errorf("Parse http error->: %s %s", clientConn.RemoteAddr().String(), err)
				return
			}

			this.wLog("Request Build,host= %s", requestBuild.Host)
			base.Log.Debugf("Request Build,host= %s", requestBuild.Host)
			dialHost = requestBuild.Host

			if ppindex := strings.LastIndex(dialHost, ":"); ppindex < 0 {
				dialHost += ":80"
			}

		case 2:
			//socker5
			sockethead, err := this.socketHandle.ConnectAndParse(clientConn)
			if err != nil {
				this.wLog("Parse socket error ->: %s %s", clientConn.RemoteAddr().String(), err)
				base.Log.Errorf("Parse socket error->: %s %s", clientConn.RemoteAddr().String(), err)
				return
			}
			dialHost = sockethead.Join_HOST_PORT
		}
	}

	this.wLog("Call DialByTimeout by:%s for clientip:%s", dialHost, clientConn.RemoteAddr().String())
	base.Log.Debugf("Call DialByTimeout by:%s for clientip:%s", dialHost, clientConn.RemoteAddr().String())

	proDialConn, err := net.DialTimeout("tcp", dialHost, timeout_seconds_dur)

	if proDialConn != nil {
		proDialConn.SetDeadline(time.Now().Add(timeout_seconds_dur))
	}

	if err != nil {
		this.wErrlog("ClientIp=" + clientConn.RemoteAddr().String() + "  DialError:" + err.Error())
		base.Log.Warnf("ClientIp=%s  DialError:%s", clientConn.RemoteAddr().String(), err.Error())
		if useReverseProxys {
			if v, ok := this.reverseProxyMap[dialHost]; ok {
				v.ErrorCount++
			}
		}
		return
	}

	defer this.DeferCallClose(proDialConn)

	//if it is http/s request and do not use reverse server
	if requestBuild != nil && !useReverseProxys {
		if requestBuild.Method == "CONNECT" { //https
			clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
			//this.wLog("clientip=%s,WriteString:%s,", clientConn.RemoteAddr().String(), "Connection Established\r\n")
			base.Log.Debugf("clientip=%s,WriteString:%s,", clientConn.RemoteAddr().String(), "Connection Established\r\n")
			if err != nil {
				this.wLog("WriteString Error")
				base.Log.Errorf("WriteString Error:%s", err)
				return
			}
		} else {
			//write has been head request to dial conn
			requestBuild.Write(proDialConn)
			this.wLog("WriteRequestHeaders,clientip=%s,", clientConn.RemoteAddr().String())
			base.Log.Debugf("WriteRequestHeaders,clientip=%s", clientConn.RemoteAddr().String())
		}
	}

	var completedChan chan int = make(chan int)

	//if clientConn have new request then read clientConn write proDialConn
	go func() {

		var buf []byte = make([]byte, this.config.BuffSize)
		var readwriteError error
		if this.config.TimeoutModel == "auto" {
			_, readwriteError = base.DefUtil.CopyBufferForRollTimeout(proDialConn, clientConn, buf, timeout_seconds_dur)
		} else {
			_, readwriteError = io.CopyBuffer(proDialConn, clientConn, buf)
		}
		this.wLog("read clientConn->write proDialConn end,clientIP:%s dialToIp:%s error:%s", clientConn.RemoteAddr().String(), proDialConn.RemoteAddr().String(), readwriteError)
		base.Log.Debugf("read clientConn->write proDialConn end,clientIP:%s dialToIp:%s error:%s", clientConn.RemoteAddr().String(), proDialConn.RemoteAddr().String(), readwriteError)

		completedChan <- 1
	}()

	//if proDialConn have new respone then read proDialConn write clientConn
	go func() {

		var buf []byte = make([]byte, this.config.BuffSize)
		var readwriteError error
		if this.config.TimeoutModel == "auto" {
			_, readwriteError = base.DefUtil.CopyBufferForRollTimeout(clientConn, proDialConn, buf, timeout_seconds_dur)
		} else {
			_, readwriteError = io.CopyBuffer(clientConn, proDialConn, buf)
		}

		this.wLog("read proDialConn->write clientConn end,clientIP:%s dialToIp:%s error:%s", clientConn.RemoteAddr().String(), proDialConn.RemoteAddr().String(), readwriteError)
		base.Log.Debugf("read proDialConn->write clientConn end,clientIP:%s dialToIp:%s error:%s", clientConn.RemoteAddr().String(), proDialConn.RemoteAddr().String(), readwriteError)

		completedChan <- 1
	}()

	//exit counter
	var result int = 0
	for {
		result += <-completedChan
		this.wLog("<-completedChan=%d", result)
		base.Log.Debugf("<-completedChan=%d", result)
		if result >= 2 {
			close(completedChan)
			this.wLog("closed all channel Connection end,clientIP=%s", clientConn.RemoteAddr().String())
			base.Log.Debugf("closed all channel Connection end,clientIP=%s", clientConn.RemoteAddr().String())
			break
		}
	}

	//keep must be close chan
	defer func(re int) {
		if completedChan != nil {
			if re < 2 {
				close(completedChan)
				this.wLog("if not closed then again to close Chan,clientIP=%s", clientConn.RemoteAddr().String())
				base.Log.Infof("if not closed then again to close Chan,clientIP=%s", clientConn.RemoteAddr().String())
			}
		}
	}(result)
}

func (this *ProxyServer) DeferCallClose(closer net.Conn) {
	defer func() {
		if p := recover(); p != nil {
			this.wErrlog("##DeferCallClose Recover Info:##", p)
			base.Log.Errorf("DeferCallClose Recover Info:%s", p)
		}
	}()
	if closer != nil {
		reip := closer.RemoteAddr().String()
		this.wLog("Close call=%s", reip)
		base.Log.Debugf("Close call=%s", reip)
		//if conn, ok := closer.(net.Conn); ok {
		this.closerConnNotify <- reip

		closer.Close()
	}
}

func (this *ProxyServer) Dispose() {
	base.Log.Debugf("Dispose call start...")
	for _, link := range this.linkListenerMap {
		if link != nil {
			link.Close()
		}
	}
	base.Log.Debugf("Dispose call end...")
	if base.Log != nil {
		time.Sleep(time.Millisecond * 500)
		base.Log.Dispose()
	}
}
