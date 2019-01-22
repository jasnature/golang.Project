// ApiTestPro project main.go
package main

import (
	"connProxy/base"
	"flag"
	"fmt"
)

var (
	pt      string
	plog    int
	allowIp string
	buf     int
)

func main() {
	fmt.Println("Start proxy...!")
	base.Log.Info("Start proxy...!")
	flag.StringVar(&pt, "pt", "9696", "please input listen port.def(9696)")
	flag.IntVar(&plog, "plog", 0, "please set print log status. def(0-disable 1-enable)")
	flag.IntVar(&buf, "buf", 1024*16, "please set print log status. def(1024*16k)")
	flag.StringVar(&allowIp, "allip", "*", "allow * or access ip address list(use , split(etc. 10.21.30.159,10.21.30.160,10.21.30.151).)")
	flag.Parse()

	//linkCount: 0, addr: ip, port: pt, printLog: plog == 1, buffSize: buf, allowIpStr: allowIp
	configObj := base.ProxyConfig{Port: pt, PrintConsoleLog: plog == 1, BuffSize: buf, AllowIpStr: allowIp}
	proxy := &ProxyServer{config: configObj}

	proxy.StartProxy()

	//fmt.Println("End proxy...!", proxy)
}
