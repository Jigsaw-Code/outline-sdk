package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/Jigsaw-Code/outline-internal-sdk/network"
	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
	"github.com/Jigsaw-Code/outline-internal-sdk/transport/shadowsocks"
	"github.com/Jigsaw-Code/outline-internal-sdk/tun2socks/lwip"
	"github.com/songgao/water"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// Compile with `go build -ldflags="-extldflags=-static"` on Linux
// We only support Linux for now

const OUTLINE_TUN_NAME = "outline233"
const OUTLINE_TUN_IP = "10.233.233.1"
const OUTLINE_TUN_MTU = 1500 // todo: we can read this from netlink
const OUTLINE_TUN_SUBNET = "10.233.233.1/32"
const OUTLINE_GW_SUBNET = "10.233.233.2/32"
const OUTLINE_GW_IP = "10.233.233.2"
const OUTLINE_ROUTING_PRIORITY = 23333
const OUTLINE_ROUTING_TABLE = 233

// ./app
//
//	<svr-ip>     : the outline server IP (e.g. 111.111.111.111)
//	<svt-port>   : the outline server port (e.g. 21532)
//	<svr-pass>   : the outline server password
func main() {
	fmt.Println("OutlineVPN CLI (experimental-05092000)")

	svrIp := os.Args[1]
	svrIpCidr := svrIp + "/32"
	svrPass := os.Args[3]
	svrPort, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return
	}
	if svrPort < 1000 || svrPort > 65535 {
		fmt.Printf("fatal error: server port out of range\n")
		return
	}

	// wait for go routine to terminate
	defer time.Sleep(1 * time.Second)

	tun, err := setupTunDevice()
	if err != nil {
		return
	}
	defer cleanUpTunDevice(tun)

	if err := showTunDevice(); err != nil {
		return
	}
	if err := configureTunDevice(); err != nil {
		return
	}
	if err := showTunDevice(); err != nil {
		return
	}

	err = setupRouting()
	if err != nil {
		return
	}
	defer cleanUpRouting()

	if err := showRouting(); err != nil {
		return
	}

	r, err := setupIpRule(svrIpCidr)
	if err != nil {
		return
	}
	defer cleanUpRule(r)

	if err := showAllRules(); err != nil {
		return
	}

	t2s, err := startTun2Socks(tun, svrIp, svrPass, svrPort)
	if err != nil {
		return
	}
	defer stopTun2Socks(t2s)

	go func() {
		fmt.Printf("debug: start receiving data from tun %v\n", tun.Name())
		if _, err := io.Copy(t2s, tun); err != nil {
			fmt.Printf("warning: failed to write data to network stack: %v\n", err)
		} else {
			fmt.Printf("debug: %v -> t2s eof\n", tun.Name())
		}
	}()

	go func() {
		fmt.Printf("debug: start forwarding t2s data to tun %v\n", tun.Name())
		if _, err = io.Copy(tun, t2s); err != nil {
			fmt.Printf("warning: failed to forward t2s data to tun: %v\n", err)
		} else {
			fmt.Printf("debug: t2s -> %v eof\n", tun.Name())
		}
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill, unix.SIGTERM, unix.SIGHUP)
	s := <-sigc
	fmt.Printf("\nReceived %v, cleaning up resources...\n", s)
}

func showTunDevice() error {
	l, err := netlink.LinkByName(OUTLINE_TUN_NAME)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	if tun, ok := l.(*netlink.Tuntap); ok {
		mode := "unknown"
		if tun.Mode == netlink.TUNTAP_MODE_TUN {
			mode = "tun"
		} else if tun.Mode == netlink.TUNTAP_MODE_TAP {
			mode = "tap"
		}
		persist := "persist"
		if tun.NonPersist {
			persist = "non-persist"
		}
		fmt.Printf("\t%v %v %v mtu=%v attr=%v stat=%v\n", tun.Name, mode, persist, tun.MTU, tun.Attrs(), tun.Statistics)
		return nil
	} else {
		fmt.Printf("fatal error: %v is not a tun device\n", OUTLINE_TUN_NAME)
		return fmt.Errorf("tun device not found")
	}
}

func setupTunDevice() (*water.Interface, error) {
	fmt.Println("setting up tun device...")
	conf := water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name:    OUTLINE_TUN_NAME,
			Persist: false,
		},
	}
	r, err := water.New(conf)
	if err == nil {
		fmt.Println("tun device created")
	} else {
		fmt.Printf("fatal error: %v\n", err)
	}
	return r, err
}

func configureTunDevice() error {
	fmt.Println("configuring tun device ip...")
	tun, err := netlink.LinkByName(OUTLINE_TUN_NAME)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	addr, err := netlink.ParseAddr(OUTLINE_TUN_SUBNET)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	if err := netlink.AddrAdd(tun, addr); err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	if err := netlink.LinkSetUp(tun); err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	return nil
}

func cleanUpTunDevice(tun *water.Interface) error {
	fmt.Println("cleaning up tun device...")
	err := tun.Close()
	if err == nil {
		fmt.Println("tun device deleted")
	} else {
		fmt.Printf("clean up error: %v\n", err)
	}
	return err
}

func showRouting() error {
	filter := netlink.Route{Table: OUTLINE_ROUTING_TABLE}
	routes, err := netlink.RouteListFiltered(netlink.FAMILY_V4, &filter, netlink.RT_FILTER_TABLE)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	fmt.Printf("\tRoutes (@%v): %v\n", OUTLINE_ROUTING_TABLE, len(routes))
	for _, route := range routes {
		fmt.Printf("\t\t%v\n", route)
	}
	return nil
}

func setupRouting() error {
	fmt.Println("configuring outline routing table...")
	tun, err := netlink.LinkByName(OUTLINE_TUN_NAME)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}

	dst, err := netlink.ParseIPNet(OUTLINE_GW_SUBNET)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	r := netlink.Route{
		LinkIndex: tun.Attrs().Index,
		Table:     OUTLINE_ROUTING_TABLE,
		Dst:       dst,
		Src:       net.ParseIP(OUTLINE_TUN_IP),
		Scope:     netlink.SCOPE_LINK,
	}
	fmt.Printf("\trouting only from %v to %v through nic %v...\n", r.Src, r.Dst, r.LinkIndex)
	err = netlink.RouteAdd(&r)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}

	r = netlink.Route{
		LinkIndex: tun.Attrs().Index,
		Table:     OUTLINE_ROUTING_TABLE,
		Gw:        net.ParseIP(OUTLINE_GW_IP),
	}
	fmt.Printf("\tdefault routing entry via gw %v through nic %v...\n", r.Gw, r.LinkIndex)
	err = netlink.RouteAdd(&r)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}

	fmt.Println("routing table has been successfully configured")
	return nil
}

func cleanUpRouting() error {
	fmt.Println("cleaning up outline routing table...")
	filter := netlink.Route{Table: OUTLINE_ROUTING_TABLE}
	routes, err := netlink.RouteListFiltered(netlink.FAMILY_V4, &filter, netlink.RT_FILTER_TABLE)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	var lastErr error = nil
	for _, route := range routes {
		if err := netlink.RouteDel(&route); err != nil {
			fmt.Printf("fatal error: %v\n", err)
			lastErr = err
		}
	}
	if lastErr == nil {
		fmt.Println("routing table has been reset")
	}
	return lastErr
}

func showAllRules() error {
	rules, err := netlink.RuleList(netlink.FAMILY_ALL)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	for _, r := range rules {
		fmt.Printf("\t%v\n", r)
	}
	return nil
}

func setupIpRule(svrIp string) (*netlink.Rule, error) {
	fmt.Println("adding ip rule for outline routing table...")
	dst, err := netlink.ParseIPNet(svrIp)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return nil, err
	}
	rule := netlink.NewRule()
	rule.Priority = OUTLINE_ROUTING_PRIORITY
	rule.Family = netlink.FAMILY_V4
	rule.Table = OUTLINE_ROUTING_TABLE
	rule.Dst = dst
	rule.Invert = true
	fmt.Printf("+from all not to %v via table %v...\n", rule.Dst, rule.Table)
	err = netlink.RuleAdd(rule)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return nil, err
	}
	fmt.Println("ip rule for outline routing table created")
	return rule, nil
}

func cleanUpRule(rule *netlink.Rule) error {
	fmt.Println("cleaning up ip rule of routing table...")
	err := netlink.RuleDel(rule)
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
		return err
	}
	fmt.Println("ip rule of routing table deleted")
	return nil
}

func startTun2Socks(tun *water.Interface, ip, pass string, port int) (network.IPDevice, error) {
	fmt.Println("starting outline-go-tun2socks...")

	cipher, err := shadowsocks.NewCipher("chacha20-ietf-poly1305", pass)
	if err != nil {
		fmt.Printf("fatal error: failed to create Shadowsocks cipher, %v\n", err)
		return nil, err
	}

	proxyIP, err := net.ResolveIPAddr("ip", ip)
	if err != nil {
		fmt.Printf("fatal error: failed to resolve proxy address, %v\n", err)
		return nil, err
	}
	proxyTCPEndpoint := transport.TCPEndpoint{RemoteAddr: net.TCPAddr{IP: proxyIP.IP, Port: port}}
	proxyUDPEndpoint := transport.UDPEndpoint{RemoteAddr: net.UDPAddr{IP: proxyIP.IP, Port: port}}

	sd, err := shadowsocks.NewStreamDialer(proxyTCPEndpoint, cipher)
	if err != nil {
		fmt.Printf("fatal error: failed to create StreamDialer, %v\n", err)
		return nil, err
	}

	pl, err := shadowsocks.NewPacketListener(proxyUDPEndpoint, cipher)
	if err != nil {
		fmt.Printf("fatal error: failed to create PacketListener, %v\n", err)
		return nil, err
	}

	t2s, err := lwip.NewTun2SocksDevice(sd, pl)
	if err != nil {
		fmt.Printf("fatal error: failed to create Tun2Socks device, %v\n", err)
		return nil, err
	}

	fmt.Println("lwIP tun2socks created")
	return t2s, nil
}

func stopTun2Socks(t2s network.IPDevice) error {
	fmt.Println("stopping outline-go-tun2socks...")
	err := t2s.Close()
	if err != nil {
		fmt.Printf("fatal error: %v\n", err)
	}
	fmt.Println("outline-go-tun2socks stopped")
	return err
}
