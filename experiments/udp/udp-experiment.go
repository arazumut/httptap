package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/mdlayher/packet"
	"golang.org/x/sys/unix"
)

func Main() error {
	var args struct {
		Arayüz  string `arg:"positional,required"`
		Uzak    string `arg:"positional,required"`
		Sayı    int    `default:"2"`
	}
	arg.MustParse(&args)

	// Arayüzü al
	iface, err := net.InterfaceByName(args.Arayüz)
	if err != nil {
		return err
	}

	// Ham IP paketlerini dinle (root yetkisi gerektirir)
	conn, err := packet.Listen(iface, packet.Raw, packet.All nil)
	if err != nil {
		if errors.Is(err, unix.EPERM) {
			return fmt.Errorf("ham paketleri okumak için root yetkisine ihtiyacınız var (%w)", err)
		}
		return fmt.Errorf("ham paket dinlerken hata: %w", err)
	}

	// Promiscuous modu ayarla, böylece her şeyi görebiliriz
	err = conn.SetPromiscuous(true)
	if err != nil {
		return fmt.Errorf("ham paket bağlantısını promiscuous moda ayarlarken hata: %w", err)
	}

	// UDP paketi gönder
	udpconn, err := net.Dial("udp", args.Uzak)
	if err != nil {
		return fmt.Errorf("bağlanırken hata %v: %w", args.Uzak, err)
	}
	udpconn.Write([]byte("udp-deneyi'nden merhaba..."))

	// Paket oku
	buf := make([]byte, iface.MTU)
	for i := 0; i < args.Sayı; i++ {
		n, srcmac, err := conn.ReadFrom(buf)
		if err != nil {
			return fmt.Errorf("ham paket okurken hata: %w", err)
		}
		_ = srcmac

		// gopacket ile decode et
		log.Printf("%d bayt okundu", n)
		packet := gopacket.NewPacket(buf[:n], layers.LayerTypeIPv4, gopacket.NoCopy)
		log.Println(packet.Dump())
	}

	return nil
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)
	err := Main()
	if err != nil {
		log.Fatal(err)
	}
}
