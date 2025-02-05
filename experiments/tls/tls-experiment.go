package main

import (
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/joemiller/certin"
)

// Sertifika dosyasını yazan fonksiyon
func sertifikaDosyasiniYaz(cert []byte, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("PEM dosyasını yazmak için açarken hata: %w", err)
	}
	defer f.Close()

	err = pem.Encode(f, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
	if err != nil {
		return fmt.Errorf("CA'yı PEM'e kodlarken hata: %w", err)
	}

	log.Printf("%v oluşturuldu", path)
	return nil
}

func Ana() error {
	var args struct {
		Port string `default:":19870"`
	}
	arg.MustParse(&args)

	root, err := certin.NewCert(nil, certin.Request{CN: "root CA", IsCA: true})
	if err != nil {
		return fmt.Errorf("root CA oluşturulurken hata: %w", err)
	}

	// Sertifika otoritesini geçici bir dosyaya yaz
	err = sertifikaDosyasiniYaz(root.Certificate.Raw, "ca.crt")
	if err != nil {
		return err
	}

	// HTTP sunucusunu başlat
	const metin = "merhaba httptap dünyası"
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, metin)
	}))
	server.TLS = &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			log.Printf("%q için challenge alındı", hello.ServerName)
			onthefly, err := certin.NewCert(root, certin.Request{CN: hello.ServerName})
			if err != nil {
				log.Println("sertifika oluşturulurken hata: %w", err)
				return nil, fmt.Errorf("%q için anında sertifika oluşturulurken hata: %w", hello.ServerName, err)
			}

			err = sertifikaDosyasiniYaz(onthefly.Certificate.Raw, "certificate.crt")
			if err != nil {
				log.Printf("anında sertifika dosyasına yazılırken hata: %v, göz ardı ediliyor", err)
			}

			tlscert := onthefly.TLSCertificate()
			return &tlscert, nil
		},
	}
	server.Listener, err = net.Listen("tcp", args.Port)
	if err != nil {
		return fmt.Errorf("%v üzerinde dinlenemiyor: %w", args.Port, err)
	}

	server.StartTLS()
	defer server.Close()

	// Sunucu ile iletişim kurmak için CA'ya güvenen bir http.Client yapılandır
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    root.CertPool(),
			ServerName: "example.com",
		},
	}
	httpClient := http.Client{
		Transport: transport,
	}

	url := fmt.Sprintf("https://127.0.0.1%v/", args.Port)
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}

	// Yanıtı doğrula
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := strings.TrimSpace(string(respBodyBytes[:]))
	if body != metin {
		return fmt.Errorf("uyumsuzluk, alınan: %q", body)
	}

	log.Printf("Bağlantının yerel olarak çalıştığı doğrulandı, şimdi %v üzerinde dinleniyor ...", server.URL)
	select {}
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	err := Ana()
	if err != nil {
		log.Fatal(err)
	}
}
