package main

import (
	"io"
	"net"
)

// proxyTCP, bir TCP bağlantısında alınan verileri dünyaya ve geri diğer yöne proxyler.
func proxyTCP(dst string, subprocess net.Conn) {
	// bağlantının "LocalAddr"ı aslında diğer tarafın (alt süreç) ulaşmaya çalıştığı adres,
	// bu yüzden proxy yapmak için bu adresi arıyoruz
	world, err := net.Dial("tcp", dst)
	if err != nil {
		// TODO: hedefin ulaşılamaz olmasıyla ilgili olmayan hataları raporla
		subprocess.Close()
		return
	}

	go proxyBytes(subprocess, world)
	go proxyBytes(world, subprocess)
}

// proxyBytes, dünya ile alt süreç arasında veri kopyalar
func proxyBytes(w io.Writer, r io.Reader) {
	buf := make([]byte, 1<<20)
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			// dış dünyaya bittiğimizi nasıl bildirebiliriz?
			return
		}
		if err != nil {
			// dış dünyaya okumanın başarısız olduğunu nasıl bildirebiliriz?
			errorf("proxyBytes içinde okuma hatası: %v, bırakılıyor", err)
			return
		}

		// paketi kanala gönder, başarısızlık durumunda bırak
		_, err = w.Write(buf[:n])
		if err != nil {
			errorf("proxyBytes içinde yazma hatası: %v, %d bayt bırakılıyor", err, n)
		}
	}
}
