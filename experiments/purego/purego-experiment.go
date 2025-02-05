package main

import (
	"log"
	"os"

	"github.com/ebitengine/purego"
)

const (
	OPENSSL_INIT_ADD_ALL_CIPHERS = 0x00000004
	OPENSSL_INIT_ADD_ALL_DIGESTS = 0x00000008
)

func AnaFonksiyon() error {
	libcrypto, err := purego.Dlopen("libcrypto.so", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return err
	}

	var VarsayılanSertifikaDizinOrtamDeğişkeni func() string
	var VarsayılanSertifikaDizini func() string

	purego.RegisterLibFunc(&VarsayılanSertifikaDizinOrtamDeğişkeni, libcrypto, "X509_get_default_cert_dir_env")
	purego.RegisterLibFunc(&VarsayılanSertifikaDizini, libcrypto, "X509_get_default_cert_dir")

	log.Println(VarsayılanSertifikaDizinOrtamDeğişkeni())
	log.Println(VarsayılanSertifikaDizini())

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
