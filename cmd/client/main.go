package main

import (
	"flag"
	"net"
	"runtime"

	"tunnel/log"
	"tunnel/protos"
	"tunnel/tun"

	"github.com/vmihailenco/msgpack/v5"
)

var ifceName string
var ifceAddr string
var serverAddr string
var serverPort int

func init() {
	flag.StringVar(&ifceName, "name", "tun0", "ifce name")
	flag.StringVar(&ifceAddr, "ip", "172.16.0.2/24", "ifce ip")
	flag.StringVar(&serverAddr, "server", "192.168.123.106", "server ip")
	flag.IntVar(&serverPort, "port", 38796, "server udp port")
	flag.Parse()
}

func main() {
	conn, err := net.DialUDP("udp",
		&net.UDPAddr{IP: net.IPv4zero, Port: serverPort},
		&net.UDPAddr{IP: net.ParseIP(serverAddr), Port: serverPort},
	)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	go tun.Read(ifceAddr, ifceName, func(b []byte) (int, error) {
		msg, err := msgpack.Marshal(protos.Message{
			Type: protos.MessageTypeData,
			Data: b,
		})
		if err != nil {
			return 0, err
		}

		log.Debugf("udp write [%s -> %s], len:%d", conn.LocalAddr().String(), conn.RemoteAddr().String(), len(msg))
		return conn.Write(msg)
	})

	// hello
	ip, _, _ := net.ParseCIDR(ifceAddr)
	b, _ := msgpack.Marshal(protos.Message{
		Type:    protos.MessageTypeHello,
		Data:    []byte("hello"),
		Address: ip.String(),
	})
	log.Info("client say hello")

	_, err = conn.Write(b)
	if err != nil {
		log.Fatalln(err)
	}

	data := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(data)
		if err != nil {
			log.Fatalf("error during read: %s", err)
		}

		log.Debugf("udp read [%s -> %s], len:%d", remoteAddr, conn.LocalAddr(), n)

		var msg protos.Message
		err = msgpack.Unmarshal(data[:n], &msg)
		if err != nil {
			log.Errorf("parse msg err: %s", err)
			continue
		}

		switch msg.Type {
		case protos.MessageTypeHello:
			if runtime.GOOS == "darwin" {
				tun.RunCmd("ifconfig", tun.GetIfceName(), ip.String(), msg.Address, "up")
			}
			log.Info("client success!")

		case protos.MessageTypeData:
			_, err = tun.Write(msg.Data)
		}

		if err != nil {
			log.Errorf("%s", err)
		}
	}
}
