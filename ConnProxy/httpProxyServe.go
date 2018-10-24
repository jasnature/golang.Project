// httpProxyServe
package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func (proxy *ProxyServer) InitProxy(allip string) {
	fmt.Println("ProxyServer init..")
	proxy.allowIp = make(map[string]string, 5)
	proxy.curIpLink = make(map[string]int, 10)
	proxy.allowIp["."] = "def"
	proxy.allowIp["[::1]"] = "def"
	proxy.allowIp["localhost"] = "def"
	proxy.allowIp["127.0.0.1"] = "def"
	if allowIp != "" {

		spstr := strings.Split(allip, ",")
		for _, spitem := range spstr {
			proxy.allowIp[spitem] = spitem
		}
	}
}

type ProxyServer struct {
	linkCount int
	port      string
	addr      string
	printLog  bool
	allowIp   map[string]string
	curIpLink map[string]int
}

func (proxy *ProxyServer) wLog(format string, a ...interface{}) {
	if proxy.printLog {
		if a != nil {
			fmt.Fprintf(os.Stdout, "\r\n"+format, a)
		} else {
			fmt.Fprintln(os.Stdout, format)
		}
	}
}

func (proxy *ProxyServer) wErrlog(a ...interface{}) {

	fmt.Fprintln(os.Stdout, "\r\n[Error] ", a)

}

func (this_proxy *ProxyServer) StartProxy() {

	addrStr := strings.Trim(this_proxy.addr, " ") + ":" + this_proxy.port

	link, err := net.Listen("tcp", addrStr)

	defer link.Close()

	if err != nil {
		this_proxy.wErrlog("Listen link", err.Error())
	}
	fmt.Printf("\r\nlister success : %+v \r\naddress: %s \r\n", this_proxy, addrStr)
	go func() {

		for {
			time.Sleep(time.Second * 10)
			fmt.Printf("\r\nSum Process Count -> %d,Current Link Address list-> %v", this_proxy.linkCount, this_proxy.curIpLink)

		}

	}()
	for {

		conn, accerr := link.Accept()
		//fmt.Println("link->", conn.RemoteAddr())
		reip := conn.RemoteAddr().String()

		if reip == "" {
			go this_proxy.DeferCallClose(conn)
			continue
		}
		if accerr != nil {
			this_proxy.wErrlog("Accept conn", err.Error())
			go this_proxy.DeferCallClose(conn)
			continue
		}
		i := strings.LastIndex(reip, ":")
		reip = reip[:i]
		//fmt.Println("link->", reip)
		if _, ok := this_proxy.allowIp[reip]; !ok {
			//fmt.Println("disallow->", reip)
			conn.Write([]byte("HTTP/1.1 403 Forbidden\r\n\r\n"))
			go this_proxy.DeferCallClose(conn)
			continue
		}

		if count, ok := this_proxy.curIpLink[reip]; ok {
			this_proxy.curIpLink[reip] = count + 1
		} else {
			this_proxy.curIpLink[reip] = 1
		}

		this_proxy.wLog("handle conn info: %+v", conn.RemoteAddr().String())
		this_proxy.linkCount++
		go this_proxy.handleConnection(conn, accerr)
	}

}

func (this_proxy *ProxyServer) handleConnection(clientConn net.Conn, err error) {
	defer this_proxy.DeferCallClose(clientConn)
	clientConn.SetDeadline(time.Now().Add(time.Second * 30))

	bufread := bufio.NewReader(clientConn)
	request, err := http.ReadRequest(bufread)
	//	if request != nil {
	//		this_proxy.wLog("\r\n handleConnection,%+v", *request)
	//	}
	//user, pwd, ok := request.BasicAuth()
	//this_proxy.wLog("BasicAuth,user= %s,pwd= %s,ok=%s", user, pwd, ok)
	if err != nil {
		return
	}
	this_proxy.wLog("Dial proxy connection,host= %s,URL= %s", request.Host, request.URL)
	host := request.Host

	if ppindex := strings.LastIndex(host, ":"); ppindex >= 0 {

		//		if host[ppindex:] == ":443" {
		//			host = host[:ppindex]
		//			host += ":80"
		//		}

	} else {
		host += ":80"
	}

	this_proxy.wLog("----------------%s", host)

	proDialConn, err := net.DialTimeout("tcp", host, time.Second*20)

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

	var completedChan chan int = make(chan int)

	//if clientConn have new request then read clientConn write proDialConn
	go func() {

		var buf []byte = make([]byte, 4096)
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
		var buf []byte = make([]byte, 4096)
		io.CopyBuffer(clientConn, proDialConn, buf)
		completedChan <- 2
	}()

	defer this_proxy.DeferCallClose(proDialConn)

	var result int = 0
	for {
		this_proxy.wLog("completedChan=%d", result)
		result = <-completedChan
		this_proxy.wLog("completedChan=%d", result)
		if result == 2 {
			this_proxy.wLog("handleConnection end")
			break
		}
	}
}

func (this_proxy *ProxyServer) DeferCallClose(closer io.Closer) {

	this_proxy.wLog("Close call=%+v", closer)
	closer.Close()
}
