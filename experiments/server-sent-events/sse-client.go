package main

import (
	"log"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/r3labs/sse"
)

// Main fonksiyonu, SSE (Server-Sent Events) istemcisini başlatır ve belirtilen URL'den mesajları dinler.
func Main() error {
	var args struct {
		URL string `arg:"positional,required"` // Komut satırından alınacak URL argümanı
	}
	arg.MustParse(&args) // Argümanları ayrıştır

	client := sse.NewClient(args.URL) // Yeni bir SSE istemcisi oluştur
	return client.Subscribe("messages", func(msg *sse.Event) {
		// Veri alındı!
		log.Println("veri alındı: ", string(msg.Data))
	})
}

func main() {
	log.SetOutput(os.Stdout) // Log çıktısını standart çıktıya yönlendir
	log.SetFlags(0)          // Log formatını ayarla
	err := Main()            // Main fonksiyonunu çalıştır
	if err != nil {
		log.Fatal(err) // Hata varsa logla ve çık
	}
}
