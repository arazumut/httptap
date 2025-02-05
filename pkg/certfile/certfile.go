package certfile

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"software.sslmate.com/src/go-pkcs12"
)

// WritePEM bir x509 sertifikasını PEM dosyasına yazar
func WritePEM(dosyaYolu string, sertifika *x509.Certificate) error {
	dosya, err := os.Create(dosyaYolu)
	if err != nil {
		return err
	}
	defer dosya.Close()

	return pem.Encode(dosya, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: sertifika.Raw,
	})
}

// WritePKCS12 bir x509 sertifikasını PKCS12 dosyasına yazar
func WritePKCS12(dosyaYolu string, sertifika *x509.Certificate) error {
	truststore, err := pkcs12.EncodeTrustStore([]*x509.Certificate{sertifika}, "")
	if err != nil {
		return fmt.Errorf("sertifika otoritesini pkcs12 formatında kodlarken hata: %w", err)
	}

	return os.WriteFile(dosyaYolu, truststore, os.ModePerm)
}
