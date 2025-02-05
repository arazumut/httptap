package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/alexflint/go-arg"
)

// Bu, netcat'in FIN paketleriyle ilgili davranışını araştırmak için sabit bir dize yazan ve ardından bağlantıyı kapatan bir TCP sunucusudur.

func Ana() error {
	var args struct {
		Adres string `arg:"positional" default:":11223"`
	}
	arg.MustParse(&args)

	log.Printf("%v adresinde dinleniyor ...", args.Adres)
	l, err := net.Listen("tcp", args.Adres)
	if err != nil {
		return fmt.Errorf("%v adresinde dinlenirken hata: %w", args.Adres, err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		fmt.Fprintln(conn, "merhaba fin deneyi")
		conn.Close()
		log.Printf("%v adresinden bir bağlantı kabul edildi ve kapatıldı", conn.RemoteAddr())
	}
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	err := Ana()
	if err != nil {
		log.Fatal(err)
	}
}
