// dialcontext, bir http isteğinin bağlamının http.Transport üzerindeki DialContext fonksiyonuna
// geçip geçmediğini araştırmak için bir programdır.

package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/alexflint/go-arg"
)

type contextKey string

var fooKey contextKey = "dialcontext.foo"

func Main() error {
	ctx := context.Background()

	var args struct{}
	arg.MustParse(&args)

	// İlk olarak bir http transport oluştur
	transport := http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			log.Printf("dialcontext'te, değer %q alındı", ctx.Value(fooKey))
			return net.Dial(network, address)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          5,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}

	// Şimdi bir foo anahtarı içeren bir bağlam oluştur
	ctx = context.WithValue(ctx, fooKey, "merhaba dialcontext!")

	// Şimdi sıradan bir http isteği oluştur
	req, err := http.NewRequest("GET", "https://www.monasticacademy.org", nil)
	if err != nil {
		return err
	}

	// İsteğe bağlamı ekle
	req = req.WithContext(ctx)

	// İsteği gönder
	_, err = transport.RoundTrip(req)
	if err != nil {
		return err
	}

	log.Println("tamamlandı")
	return nil
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	err := Main()
	if err != nil {
		log.Fatal(err)
	}
}
