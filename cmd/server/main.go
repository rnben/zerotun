package main

import (
	"flag"
	"net"

	"tunnel/log"
	"tunnel/protos"
	"tunnel/tun"

	"github.com/songgao/water/waterutil"
	"github.com/vmihailenco/msgpack/v5"
)

// key client tun0 ip, value client public Addr
var connPool = map[string]*net.UDPAddr{}

var (
	ifceName string
	ifceAddr string
)

func init() {
	flag.StringVar(&ifceName, "name", "tun0", "ifce name")
	flag.StringVar(&ifceAddr, "ip", "172.16.0.1/24", "ifce ip")
	flag.Parse()
}

func main() {
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 38796})
	if err != nil {
		log.Fatalln(err)
	}

	go tun.Read(ifceAddr, ifceName, func(b []byte) (int, error) {
		dst := waterutil.IPv4Destination(b).String()

		ip, _, _ := net.ParseCIDR(ifceAddr)
		msg, err := msgpack.Marshal(protos.Message{
			Type:    protos.MessageTypeData,
			Data:    b,
			Address: ip.String(),
		})
		if err != nil {
			return 0, err
		}

		remoteAddr, ok := connPool[dst]
		if !ok {
			if waterutil.IPv4Protocol(b) == waterutil.ICMP {
				log.Info("icmp")
			}

			return 0, nil
		}

		log.Debugf("udp write [%s -> %s], len:%d", listener.LocalAddr().String(), remoteAddr.String(), len(msg))
		return listener.WriteToUDP(msg, remoteAddr)
	})

	data := make([]byte, 4096)
	for {
		n, remoteAddr, err := listener.ReadFromUDP(data)
		if err != nil {
			log.Fatalf("error during read: %s", err)
		}

		log.Debugf("udp read [%s -> %s], len:%d", remoteAddr, listener.LocalAddr(), n)

		var msg protos.Message
		err = msgpack.Unmarshal(data[:n], &msg)
		if err != nil {
			log.Errorf("parse msg err: %s", err)
			continue
		}

		switch msg.Type {
		case protos.MessageTypeHello:
			connPool[msg.Address] = remoteAddr
			log.Infof("client %s(%s) connected!", msg.Address, remoteAddr.IP.String())

			ip, _, _ := net.ParseCIDR(ifceAddr)
			b, _ := msgpack.Marshal(protos.Message{
				Type:    protos.MessageTypeHello,
				Address: ip.String(),
				Data:    msg.Data,
			})
			_, err = listener.WriteToUDP(b, remoteAddr)
		case protos.MessageTypeData:
			_, err = tun.Write(msg.Data)
		}

		if err != nil {
			log.Errorf("%s ", err)
		}
	}
}
