package examples

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/netip"
	"strings"
	"testing"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
)

func TestHanging(t *testing.T) {
	// Server
	listener := server()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(io.Discard, conn)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Client
	c := client()

	var buf = []byte(strings.Repeat("hello world\n", 65536))
	for {
		fmt.Println("write...")
		n, err := c.Write(buf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("written %d B\n", n)
	}

}

func server() *gonet.TCPListener {
	tun, tnet, err := netstack.CreateNetTUN(
		[]netip.Addr{netip.MustParseAddr("192.168.4.29")},
		[]netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("8.8.4.4")},
		1420,
	)
	if err != nil {
		log.Panic(err)
	}
	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelVerbose, ""))
	dev.IpcSet(`private_key=003ed5d73b55806c30de3f8a7bdab38af13539220533055e635690b8b87ad641
listen_port=58120
public_key=f928d4f6c1b86c12f2562c10b07c555c5c57fd00f59e90c8d8d88767271cbf7c
allowed_ip=192.168.4.28/32
persistent_keepalive_interval=25
`)
	dev.Up()

	listener, err := tnet.ListenTCP(&net.TCPAddr{Port: 2000})
	if err != nil {
		log.Panicln(err)
	}
	return listener
}

func client() *gonet.TCPConn {
	tun, tnet, err := netstack.CreateNetTUN(
		[]netip.Addr{netip.MustParseAddr("192.168.4.28")},
		[]netip.Addr{netip.MustParseAddr("8.8.8.8")},
		1420)
	if err != nil {
		log.Panic(err)
	}
	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelVerbose, ""))
	err = dev.IpcSet(`private_key=087ec6e14bbed210e7215cdc73468dfa23f080a1bfb8665b2fd809bd99d28379
public_key=c4c8e984c5322c8184c72265b92b250fdb63688705f504ba003c88f03393cf28
allowed_ip=0.0.0.0/0
endpoint=127.0.0.1:58120
`)
	err = dev.Up()
	if err != nil {
		log.Panic(err)
	}

	serverAddr, err := net.ResolveTCPAddr("tcp", "192.168.4.29:2000")
	if err != nil {
		log.Panic(err)
	}
	client, err := tnet.DialTCP(serverAddr)
	if err != nil {
		log.Panic(err)
	}
	return client
}
