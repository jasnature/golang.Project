package base

import (
	"net"
	"testing"
)

func TestLog(t *testing.T) {
	ip := net.ParseIP("192.168.1.230")
	start := net.ParseIP("192.168.1.1")
	end := net.ParseIP("192.168.255.200")
	result := DefUtil.CheckIpInRange(ip, start, end)
	t.Logf("CheckIpInRange Result:%v", result)
}
