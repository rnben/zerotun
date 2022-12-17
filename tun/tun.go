package tun

import (
	"fmt"
	"net"
	"os/exec"

	"tunnel/log"

	"github.com/songgao/water"
	"github.com/songgao/water/waterutil"
)

var ifce *water.Interface

func Read(ip string, ifceName string, writeFunc func([]byte) (int, error)) {
	var err error

	ifce, err = water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		log.Fatalln(err)
	}

	if ifce.Name() == ifceName {
		RunCmd("ip", "addr", "add", ip, "dev", ifce.Name())
		RunCmd("ip", "link", "set", "dev", ifce.Name(), "up")
	}

	log.Infof("Interface Name: %s", ifce.Name())

	packet := make([]byte, 4096)
	for {
		n, err := ifce.Read(packet)
		if err != nil {
			log.Errorf("")
		}

		src, dest := parseAddr(packet[:n])
		log.Debugf("tun read [%s -> %s], len %d", src, dest, len(packet[:n]))

		_, err = writeFunc(packet[:n])
		if err != nil {
			log.Error(err)
		}
	}
}

func Write(packet []byte) (int, error) {
	src, dest := parseAddr(packet)
	log.Debugf("tun write [%s -> %s], len %d", src, dest, len(packet))

	return ifce.Write(packet)
}

func GetIfceName() string {
	if ifce == nil {
		return ""
	}

	return ifce.Name()
}

func RunCmd(cmd ...string) {
	if len(cmd) == 0 {
		return
	}
	b, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		log.Errorf("run %v failed, output: %s, err: %s", cmd, string(b), err)
	}
}

func parseAddr(packet []byte) (string, string) {
	src := fmt.Sprintf("%s:%d", waterutil.IPv4Source(packet), waterutil.IPv4SourcePort(packet))
	dest := fmt.Sprintf("%s:%d", waterutil.IPv4Destination(packet), waterutil.IPv4DestinationPort(packet))
	return src, dest
}

func DealCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); ip_tools(ip) {
		ips = append(ips, ip.String())
	}
	return ips[1 : len(ips)-1], nil
}

func ip_tools(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
