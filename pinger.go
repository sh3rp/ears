package ears

import (
	"math"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Pinger struct {
	Pings     map[string]*Ping
	pingsLock *sync.Mutex
	timeout   int64
	stopLoop  bool
}

func NewPinger(timeout int64) *Pinger {
	pinger := &Pinger{
		Pings:     make(map[string]*Ping),
		pingsLock: &sync.Mutex{},
		timeout:   timeout,
		stopLoop:  false,
	}
	pinger.Start()

	return pinger
}

func (p *Pinger) Start() {
	p.stopLoop = false
	go p.ReadLoop()
}

func (p *Pinger) done() bool {
	p.pingsLock.Lock()
	defer p.pingsLock.Unlock()

	for _, v := range p.Pings {
		if time.Now().Before(v.SentTime.Add(time.Millisecond * time.Duration(p.timeout))) {
			return false
		}
	}

	p.stopLoop = true

	return true
}

func (p *Pinger) PingIPHosts(intf string) {
	ipChannel := make(chan string, 10)

	for i := 0; i < 20; i++ {
		go func() {
			for ip := range ipChannel {
				t := time.Now()
				id, _ := ulid.New(ulid.Timestamp(t), rand.New(rand.NewSource(t.UnixNano())))
				p.pingsLock.Lock()
				p.Pings[id.String()] = &Ping{
					ID:       id.String(),
					Alive:    false,
					SentTime: time.Now(),
					IP:       ip,
				}
				p.pingsLock.Unlock()
				p.PingHost(id.String(), ip)
			}
		}()
	}

	for _, host := range p.GetNetworkIPAddrs(intf) {
		ipChannel <- host
	}

	close(ipChannel)

	for !p.done() {
		time.Sleep(time.Millisecond * 50)
	}
}

func (p *Pinger) GetIPForIntf(intf string) (*net.IP, *net.IPNet, error) {
	i, err := net.InterfaceByName(intf)

	if err != nil {
		return nil, nil, err
	}

	addrs, err := i.Addrs()

	if err != nil {
		return nil, nil, err
	}

	for _, addr := range addrs {
		if strings.Contains(addr.String(), ".") {
			ip, n, err := net.ParseCIDR(addr.String())

			if err != nil {
				return nil, nil, err
			}

			return &ip, n, nil
		}
	}

	return nil, nil, nil
}

func (p *Pinger) GetNetworkIPAddrs(intf string) []string {
	var iplist []string

	ip, network, err := p.GetIPForIntf(intf)

	if err != nil {
		return nil
	}

	ones, bits := ip.DefaultMask().Size()

	broadcast := math.Pow(2, float64(bits-ones)) - 1

	for ip := ip.Mask(network.Mask); network.Contains(ip) && broadcast > 0; inc(ip) {
		iplist = append(iplist, ip.String())
		broadcast--
	}

	return iplist
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func (p *Pinger) PingHost(id string, ipStr string) {
	c, err := net.ListenPacket("ip4:icmp", "0.0.0.0")

	if err != nil {
		log.Error().Msgf("ListenPacket: %v", err)
		return
	}

	defer c.Close()

	c.SetDeadline(time.Now().Add(3 * time.Second))

	pkt := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID:   1234,
			Data: []byte(id),
		},
	}
	data, err := pkt.Marshal(nil)

	if err != nil {
		log.Error().Msgf("Marshaling error: %v", err)
		return
	}

	ips, err := net.LookupIP(ipStr)
	ipAddr := &net.IPAddr{
		IP: ips[0],
	}

	c.WriteTo(data, ipAddr)
}

type Ping struct {
	ID       string
	Alive    bool
	SentTime time.Time
	Latency  time.Duration
	IP       string
}

func (p *Pinger) ReadLoop() {
	c, err := net.ListenPacket("ip4:icmp", "0.0.0.0")

	if err != nil {
		log.Error().Msgf("ListenPacket: %v", err)
		return
	}

	defer c.Close()

	for !p.stopLoop {
		inData := make([]byte, 1500)

		n, _, err := c.ReadFrom(inData)

		if err != nil {
			log.Error().Msgf("ReadLoop: %v", err)
			continue
		}

		icmpResponse, err := icmp.ParseMessage(1, inData[:n])

		switch icmpResponse.Type {
		case ipv4.ICMPTypeEchoReply:
			id := string(icmpResponse.Body.(*icmp.Echo).Data)
			p.pingsLock.Lock()
			if _, ok := p.Pings[id]; ok {
				p.Pings[id].Alive = true
				p.Pings[id].Latency = time.Now().Sub(p.Pings[id].SentTime) / time.Millisecond
			}
			p.pingsLock.Unlock()
		case ipv4.ICMPTypeDestinationUnreachable:
			id := string(icmpResponse.Body.(*icmp.DstUnreach).Data)
			p.pingsLock.Lock()
			if _, ok := p.Pings[id]; ok {
				p.Pings[id].Alive = false
			}
			p.pingsLock.Unlock()
		default:
			log.Error().Msgf("Unknown ICMP message type: %v", icmpResponse.Type)
		}
	}
}
