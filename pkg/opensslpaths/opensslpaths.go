package opensslpaths

// Eğer libcrypto.so yüklenebilirse, bu fonksiyon işaretçileri doldurulacak
type libcryptoFonksiyonlar struct {
	X509_get_default_cert_dir      func() string // varsayılan sertifika dizini
	X509_get_default_cert_dir_env  func() string // yukarıdakini kontrol eden ortam değişkeninin adı
	X509_get_default_cert_file     func() string // varsayılan sertifika dosyası
	X509_get_default_cert_file_env func() string // yukarıdakini kontrol eden ortam değişkeninin adı
}

// OpenSSL için yapılandırılmış varsayılan sertifika dosyasını al, eğer OpenSSL yüklü değilse veya yüklenemiyorsa boş string döner
func VarsayilanSertifikaDosyasi() string {
	defer recover()
	if lib := libcrypto(); lib != nil {
		return lib.X509_get_default_cert_file()
	}
	return ""
}

// Varsayılan sertifika dizinini kontrol eden ortam değişkeninin adını al, eğer OpenSSL yüklü değilse veya yüklenemiyorsa boş string döner
func VarsayilanSertifikaDosyasiEnv() string {
	defer recover()
	if lib := libcrypto(); lib != nil {
		return lib.X509_get_default_cert_file_env()
	}
	return ""
}

// OpenSSL için yapılandırılmış varsayılan sertifika dizinini al, eğer OpenSSL yüklü değilse veya yüklenemiyorsa boş string döner
func VarsayilanSertifikaDizini() string {
	defer recover()
	if lib := libcrypto(); lib != nil {
		return lib.X509_get_default_cert_dir()
	}
	return ""
}

// Varsayılan sertifika dizinini kontrol eden ortam değişkeninin adını al, eğer OpenSSL yüklü değilse veya yüklenemiyorsa boş string döner
func VarsayilanSertifikaDiziniEnv() string {
	defer recover()
	if lib := libcrypto(); lib != nil {
		return lib.X509_get_default_cert_dir_env()
	}
	return ""
}
