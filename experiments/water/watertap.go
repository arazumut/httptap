package main

import (
	"log"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/songgao/packets/ethernet"
	"github.com/songgao/water"
)

func Ana() error {
	var argumanlar struct {
		Cihaz string `default:"httptap"`
	}
	arg.MustParse(&argumanlar)

	konfig := water.Config{
		DeviceType: water.TAP,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: argumanlar.Cihaz,
		},
	}

	ifce, err := water.New(konfig)
	if err != nil {
		log.Fatal(err)
	}
	var cerceve ethernet.Frame

	log.Printf("Yeni cihaz %q üzerinde dinleniyor...", argumanlar.Cihaz)

	for {
		cerceve.Resize(1500)
		n, err := ifce.Read([]byte(cerceve))
		if err != nil {
			log.Fatal(err)
		}
		cerceve = cerceve[:n]
		log.Printf("Hedef: %s\n", cerceve.Destination())
		log.Printf("Kaynak: %s\n", cerceve.Source())
		log.Printf("Ethertype: % x\n", cerceve.Ethertype())
		log.Printf("Yük: % x\n", cerceve.Payload())
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
