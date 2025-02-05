package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func AnaFonksiyon() error {
	var args struct {
		Cihaz string         `default:"lo"`
		Port  layers.UDPPort `default:"9000"`
	}
	arg.MustParse(&args)

	handle, err := pcap.OpenLive(args.Cihaz, 1500, false, pcap.BlockForever)
	if err != nil {
		return err
	}

	eth := layers.Ethernet{
		EthernetType: layers.EthernetTypeIPv4,
		SrcMAC:       net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		DstMAC:       net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	_ = eth

	// Loopback arayüzü için kullanılır
	lo := layers.Loopback{
		Family: layers.ProtocolFamilyIPv4,
	}
	_ = lo

	ip := layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    net.IP{127, 0, 0, 1},
		DstIP:    net.IP{127, 0, 0, 1},
		Protocol: layers.IPProtocolUDP,
	}

	udp := layers.UDP{
		SrcPort: 62003,
		DstPort: args.Port,
	}
	err = udp.SetNetworkLayerForChecksum(&ip)
	if err != nil {
		return err
	}

	payload := []byte("merhaba düşük seviyeli ağ!\n")

	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	buffer := gopacket.NewSerializeBuffer()

	if err = gopacket.SerializeLayers(buffer, options,
		&eth,
		&ip,
		&udp,
		gopacket.Payload(payload),
	); err != nil {
		return fmt.Errorf("paket serileştirilirken hata: %w", err)
	}
	gidenPaket := buffer.Bytes()

	if err = handle.WritePacketData(gidenPaket); err != nil {
		return fmt.Errorf("paket gönderilirken hata: %w", err)
	}

	log.Println("işlem tamam")

	return nil
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	err := AnaFonksiyon()
	if err != nil {
		log.Fatal(err)
	}
}
