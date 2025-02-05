package main

import (
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// UDP paketi

type udpPacket struct {
	src     net.Addr
	dst     net.Addr
	payload []byte
}

// udpStack, gopacket ile UDP paketlerini ayrıştırır ve bunları bir mux üzerinden yönlendirir
type udpStack struct {
	toSubprocess chan []byte // bu kanala gönderilen veri, alt sürece ham IPv4 paketi olarak gider
	buf          gopacket.SerializeBuffer
	app          *mux
}

func newUDPStack(app *mux, link chan []byte) *udpStack {
	return &udpStack{
		toSubprocess: link,
		buf:          gopacket.NewSerializeBuffer(),
		app:          app,
	}
}

func (s *udpStack) handlePacket(ipv4 *layers.IPv4, udp *layers.UDP, payload []byte) {
	replyudp := layers.UDP{
		SrcPort: udp.DstPort,
		DstPort: udp.SrcPort,
	}

	replyipv4 := layers.IPv4{
		Version:  4, // IPv4'ü belirtir
		TTL:      ttl,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    ipv4.DstIP,
		DstIP:    ipv4.SrcIP,
	}

	w := udpStackResponder{
		stack:      s,
		udpheader:  &replyudp,
		ipv4header: &replyipv4,
	}

	// veriyi uygulama seviyesindeki dinleyicilere ilet
	verbosef("got %d udp bytes to %v:%v, delivering to application", len(udp.Payload), ipv4.DstIP, udp.DstPort)

	src := net.UDPAddr{IP: ipv4.SrcIP, Port: int(udp.SrcPort)}
	dst := net.UDPAddr{IP: ipv4.DstIP, Port: int(udp.DstPort)}
	s.app.notifyUDP(&w, &udpPacket{&src, &dst, payload})
}

// serializeUDP, bir UDP paketini serileştirir
func serializeUDP(ipv4 *layers.IPv4, udp *layers.UDP, payload []byte, tmp gopacket.SerializeBuffer) ([]byte, error) {
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	tmp.Clear()

	// her katman *önceden* eklenir, mevcut tampon veriyi yük olarak kabul eder
	p, err := tmp.AppendBytes(len(payload))
	if err != nil {
		return nil, fmt.Errorf("error appending UDP payload to packet (%d bytes): %w", len(payload), err)
	}
	copy(p, payload)

	err = udp.SerializeTo(tmp, opts)
	if err != nil {
		return nil, fmt.Errorf("error serializing UDP part of packet: %w", err)
	}

	err = ipv4.SerializeTo(tmp, opts)
	if err != nil {
		errorf("error serializing IP part of packet: %v", err)
	}

	return tmp.Bytes(), nil
}

// summarizeUDP, bir UDP paketini loglama için tek bir satırda özetler
func summarizeUDP(ipv4 *layers.IPv4, udp *layers.UDP, payload []byte) string {
	return fmt.Sprintf("UDP %v:%d => %v:%d - Len %d",
		ipv4.SrcIP, udp.SrcPort, ipv4.DstIP, udp.DstPort, len(udp.Payload))
}

// udpStackResponder, UDP paketlerini bilinen bir göndericiye geri yazar
type udpStackResponder struct {
	stack      *udpStack
	udpheader  *layers.UDP
	ipv4header *layers.IPv4
}

func (r *udpStackResponder) SetSourceIP(ip net.IP) {
	r.ipv4header.SrcIP = ip
}

func (r *udpStackResponder) SetSourcePort(port uint16) {
	r.udpheader.SrcPort = layers.UDPPort(port)
}

func (r *udpStackResponder) SetDestIP(ip net.IP) {
	r.ipv4header.DstIP = ip
}

func (r *udpStackResponder) SetDestPort(port uint16) {
	r.udpheader.DstPort = layers.UDPPort(port)
}

func (r *udpStackResponder) Write(payload []byte) (int, error) {
	// checksum ve uzunlukları ayarla
	r.udpheader.SetNetworkLayerForChecksum(r.ipv4header)

	// logla
	verbosef("sending udp packet to subprocess: %s", summarizeUDP(r.ipv4header, r.udpheader, payload))

	// veriyi serileştir
	packet, err := serializeUDP(r.ipv4header, r.udpheader, payload, r.stack.buf)
	if err != nil {
		return 0, fmt.Errorf("error serializing UDP packet: %w", err)
	}

	// aynı tampon yeniden kullanılacağı için bir kopya oluştur
	cp := make([]byte, len(packet))
	copy(cp, packet)

	// alt sürece kanala bloklamadan gönder
	select {
	case r.stack.toSubprocess <- cp:
	default:
		return 0, fmt.Errorf("channel for sending udp to subprocess would have blocked")
	}

	// gönderilen bayt sayısını değil, iletilen bayt sayısını döndür
	return len(payload), nil
}
