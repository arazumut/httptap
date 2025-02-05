package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
)

func AnaFonksiyon() error {
	buf := buffer.New(1 << 15) // 32 KB tampon
	r, w := nio.Pipe(buf)

	// Her 1 saniyede bir arka planda okumaya başla
	go func() {
		buf := make([]byte, 12)
		for range time.Tick(1 * time.Second) {
			n, err := r.Read(buf)
			if err != nil {
				break
			}
			log.Printf("okunan: %q", string(buf[:n]))
		}
	}()

	// Her 2 saniyede bir yaz
	var i int
	for range time.Tick(2 * time.Second) {
		log.Println("yazmaya hazırlanıyor...")
		begin := time.Now()
		fmt.Fprintf(w, "merhaba nio %d "+strings.Repeat("=", 50), i)
		log.Printf("yazma işlemi %v sürdü", time.Since(begin))
		i++
	}
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
