// ApiTestPro project main.go
package main

import (
	"flag"
	"fmt"
)

var (
	pt      string
	ip      string
	plog    int
	allowIp string
)

func main() {
	fmt.Println("Start proxy...!")

	flag.StringVar(&ip, "ip", " ", "plase input listen ip. def(all ip)")
	flag.StringVar(&pt, "pt", "9696", "plase input listen port.def(9696)")
	flag.IntVar(&plog, "plog", 0, "plase set print log status. def(0-disable 1-enable)")
	flag.StringVar(&allowIp, "allip", "10.21.30.159,10.21.30.151", "allow access ip address list(use , split(etc. 10.21.30.159,10.21.30.151).)")
	flag.Parse()
	proxy := &ProxyServer{linkCount: 0, addr: ip, port: pt, printLog: plog == 1}

	proxy.InitProxy(allowIp)
	proxy.StartProxy()

	fmt.Println("End proxy...!", proxy)
}
