package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
)

func AnaFonksiyon() error {
	var argumanlar struct {
		Adres   string `arg:"positional,required"`
		HostAdi string `arg:"positional,required"`
	}
	arg.MustParse(&argumanlar)

	// Sunucuya bağlan
	baglanti, err := net.Dial("tcp", argumanlar.Adres)
	if err != nil {
		return fmt.Errorf("sunucuya bağlanırken hata: %w", err)
	}

	tlsBaglanti := tls.Client(baglanti, &tls.Config{ServerName: argumanlar.HostAdi})

	err = tlsBaglanti.Handshake()
	if err != nil {
		return fmt.Errorf("tls el sıkışması başarısız: %w", err)
	}

	err = tlsBaglanti.VerifyHostname(argumanlar.HostAdi)
	if err != nil {
		return fmt.Errorf("tls host adı doğrulanamadı: %w", err)
	}

	// HTTP isteği oluştur
	istek, err := http.NewRequest("GET", "https://"+argumanlar.HostAdi, nil)
	if err != nil {
		return err
	}

	// İsteği TLS bağlantısına yaz
	err = istek.Write(tlsBaglanti)
	if err != nil {
		return fmt.Errorf("tls üzerinden http isteği gönderilirken hata: %w", err)
	}

	// TLS bağlantısından yanıtı oku
	yanit, err := http.ReadResponse(bufio.NewReader(tlsBaglanti), istek)
	if err != nil {
		return fmt.Errorf("tls üzerinden http yanıtı okunurken hata: %w", err)
	}
	defer yanit.Body.Close()

	// Tüm gövdeyi oku
	govde, err := io.ReadAll(yanit.Body)
	if err != nil {
		return fmt.Errorf("tls üzerinden http gövdesi okunurken hata: %w", err)
	}

	// Sonucu logla
	log.Println(strings.TrimSpace(string(govde)))

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
