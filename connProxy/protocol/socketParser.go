// util
package protocol

import (
	connbase "connProxy/base"
	"errors"
	"net"
	"strconv"
)

//========client to server====
// 第一个字段VER代表Socket的版本，Soket5默认为0x05，其固定长度为1个字节
// 第二个字段NMETHODS表示第三个字段METHODS的长度，它的长度也是1个字节
// 第三个METHODS表示客户端支持的验证方式，可以有多种，他的尝试是1-255个字节。
// X’00’ NO AUTHENTICATION REQUIRED（不需要验证）
// X’01’ GSSAPI
// X’02’ USERNAME/PASSWORD（用户名密码）
// X’03’ to X’7F’ IANA ASSIGNED
// X’80’ to X’FE’ RESERVED FOR PRIVATE METHODS
// X’FF’ NO ACCEPTABLE METHODS（都不支持，没法连接了）

//=====server to client=====
// VER	METHOD
// 1	1
// 第一个字段VER代表Socket的版本，Soket5默认为0x05，其值长度为1个字节
// 第二个字段METHOD代表需要服务端需要客户端按照此验证方式提供验证信息，其值长度为1个字节，选择为上面的六种验证方式。

//========client to server====
// VER	CMD	RSV		ATYP	DST.ADDR	DST.PORT
// 1	1	X’00’	1		Variable	2
// VER代表Socket协议的版本，Soket5默认为0x05，其值长度为1个字节
// CMD代表客户端请求的类型，值长度也是1个字节，有三种类型
// CONNECT X’01’
// BIND X’02’
// UDP ASSOCIATE X’03’
// RSV保留字，值长度为1个字节
// ATYP代表请求的远程服务器地址类型，值长度1个字节，有三种类型
// IP V4 address: X’01’
// DOMAINNAME: X’03’
// IP V6 address: X’04’
// DST.ADDR代表远程服务器的地址，根据ATYP进行解析，值长度不定。
// DST.PORT代表远程服务器的端口，要访问哪个端口的意思，值长度2个字节

//=====server to client=====
// VER	REP	RSV	ATYP	BND.ADDR	BND.PORT
// 1	1	X’00’	1	Variable	2

// VER代表Socket协议的版本，Soket5默认为0x05，其值长度为1个字节
// REP代表响应状态码，值长度也是1个字节，有以下几种类型
// X’00’ succeeded
// X’01’ general SOCKS server failure
// X’02’ connection not allowed by ruleset
// X’03’ Network unreachable
// X’04’ Host unreachable
// X’05’ Connection refused
// X’06’ TTL expired
// X’07’ Command not supported
// X’08’ Address type not supported
// X’09’ to X’FF’ unassigned
// RSV保留字，值长度为1个字节
// ATYP代表请求的远程服务器地址类型，值长度1个字节，有三种类型
// IP V4 address: X’01’
// DOMAINNAME: X’03’
// IP V6 address: X’04’
// BND.ADDR表示绑定地址，值长度不定。
// BND.PORT表示绑定端口，值长度2个字节

var (
	no_auth   = []byte{0x05, 0x00}
	with_auth = []byte{0x05, 0x02}

	auth_success = []byte{0x05, 0x00}
	auth_failed  = []byte{0x05, 0x01}

	connect_success = []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

type SocketParser struct {
	Config *connbase.SocketItem
}

type SockerHead struct {
	//0x05 Soket5
	VER byte
	// IP V4 address: X’01’
	// DOMAINNAME: X’03’
	// IP V6 address: X’04’
	ATYP           byte
	DST_ADDR       string
	DST_PORT       string
	Join_HOST_PORT string
}

func (this *SocketParser) ConnectAndParse(client net.Conn) (result SockerHead, err error) {

	if client == nil {
		return SockerHead{}, errors.New("[ConnectAndParse] client it is nil")
	}

	result = SockerHead{}

	var b [1024]byte
	n, err := client.Read(b[:])

	if err != nil {
		return SockerHead{}, err
	}

	if b[0] == 0x05 { //Socket5
		result = SockerHead{}
		//direct response client -> not auth

		if this.Config != nil && this.Config.Socket_Auth {

			client.Write(with_auth)

			n, err = client.Read(b[:])

			if err != nil {
				return SockerHead{}, err
			}

			user_length := int(b[1])
			user := string(b[2:(2 + user_length)])
			pass := string(b[(2 + user_length):])

			//fmt.Printf("len=%+v, u=%s p=%s", b[:n], user, pass)

			if this.Config.Socket_UID == user && this.Config.Socket_PWD == pass {
				client.Write(auth_success)
			} else {
				client.Write(auth_failed)
				return SockerHead{}, errors.New("[ConnectAndParse] auth failed.")
			}

		} else {
			client.Write(no_auth)
		}

		n, err = client.Read(b[:])
		var host, port string
		result.VER = b[0]
		result.ATYP = b[3]

		switch result.ATYP {
		case 0x01: //IP V4
			host = net.IPv4(b[4], b[5], b[6], b[7]).String()
		case 0x03: //domain
			host = string(b[5 : n-2]) //b[4]->domain length
		case 0x04: //IP V6
			host = net.IP{b[4], b[5], b[6], b[7], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15], b[16], b[17], b[18], b[19]}.String()
		}
		port = strconv.Itoa(int(b[n-2])<<8 | int(b[n-1]))

		result.DST_ADDR = host
		result.DST_PORT = port
		result.Join_HOST_PORT = net.JoinHostPort(host, port)

		//response client connect success
		client.Write(connect_success) //响应客户端连接成功

		//next need each copy request buffer
		return result, nil
	}

	return SockerHead{}, errors.New("[ConnectAndParse] it not socket protocol.")
}
