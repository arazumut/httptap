package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/alexflint/go-arg"
)

func Ana() error {
	var argumanlar struct {
		Host string `arg:"positional,required"`
	}
	arg.MustParse(&argumanlar)

	ctx := context.Background()
	ipler, hata := net.DefaultResolver.LookupIP(ctx, "ip4", argumanlar.Host)
	if hata != nil {
		return fmt.Errorf("varsayılan çözücü dedi ki: %w", hata)
	}
	log.Println(ipler)
	return nil
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	hata := Ana()
	if hata != nil {
		log.Fatal(hata)
	}
}
