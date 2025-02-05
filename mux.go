package main

import (
	"net"
	"strings"
	"sync"
)

// Bir dinleme desenini HOST:PORT biçimindeki bir adres dizesiyle eşleştirir
func desenEşleşiyorMu(desen string, addr net.Addr) bool {
	if desen == "*" {
		return true
	}
	if strings.HasPrefix(desen, ":") && strings.HasSuffix(addr.String(), desen) {
		return true
	}
	return false
}

// mux, ağ bağlantılarını desenlere göre dinleyicilere yönlendirir
type mux struct {
	mu          sync.Mutex
	tcpHandlers []*tcpMuxEntry
	udpHandlers []*udpMuxEntry
}

// tcpHandlerFunc, TCP bağlantılarını alan bir işlevdir
type tcpHandlerFunc func(net.Conn)

// tcpRequestHandlerFunc, TCP bağlantı isteklerini alan ve kabul edip etmeme seçeneğine sahip bir işlevdir.
type tcpRequestHandlerFunc func(TCPRequest)

// tcpMuxEntry, tcp yığını için mux tablosunda kullanılmak üzere bir desen ve ilgili işleyicidir
type tcpMuxEntry struct {
	desen   string
	handler tcpRequestHandlerFunc
}

// udpHandlerFunc, UDP paketlerini alan bir işlevdir. w.Write çağrısının her biri, orijinal paketin gönderildiği hedeften geliyormuş gibi görünen bir UDP paketi gönderir.
type udpHandlerFunc func(w udpResponder, packet *udpPacket)

// udpMuxEntry, udp yığını için mux tablosunda kullanılmak üzere bir desen ve ilgili işleyicidir
type udpMuxEntry struct {
	handler udpHandlerFunc
	desen   string
}

// ListenTCP, bir filtre desenine göre bağlantıları kesen bir net.Listener döndürür.
//
// Desen bir ana bilgisayar adı, bir :port, bir ana bilgisayar adı:port veya "*" her şey için olabilir. Örneğin:
//   - "example.com"
//   - "example.com:80"
//   - ":80"
//   - "*"
//
// Daha sonra bu net.ListenTCP gibi olacak
func (s *mux) ListenTCP(desen string) net.Listener {
	s.mu.Lock()
	defer s.mu.Unlock()

	listener := tcpListener{desen: desen, connections: make(chan net.Conn, 64)}
	s.HandleTCPRequest(desen, func(r TCPRequest) {
		conn, err := r.Accept()
		if err != nil {
			return
		}
		listener.connections <- conn
	})
	return &listener
}

// HandleTCP, verilen filtre desenine uyan her yeni bağlantı kesildiğinde çağrılacak bir işleyici kaydeder.
//
// Desen bir ana bilgisayar adı, bir :port, bir ana bilgisayar adı:port veya "*" her şey için olabilir. Örneğin:
//   - "example.com"
//   - "example.com:80"
//   - ":80"
//   - "*"
func (s *mux) HandleTCP(desen string, handler tcpHandlerFunc) {
	s.HandleTCPRequest(desen, func(r TCPRequest) {
		conn, err := r.Accept()
		if err != nil {
			errorf("bağlantı kabul edilirken hata: %v", err)
			return
		}
		handler(conn)
	})
}

// HandleTCPRequest, verilen filtre desenine uyan her yeni bağlantı kesildiğinde çağrılacak bir işleyici kaydeder. HandleTCP'den farklı olarak, işleyici bağlantının kabul edilip edilmeyeceğini kontrol edebilir, bu da SYN+ACK veya SYN+RST ile yanıt vermek anlamına gelir.
//
// Desen bir ana bilgisayar adı, bir :port, bir ana bilgisayar adı:port veya "*" her şey için olabilir. Örneğin:
//   - "example.com"
//   - "example.com:80"
//   - ":80"
//   - "*"
func (s *mux) HandleTCPRequest(desen string, handler tcpRequestHandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tcpHandlers = append(s.tcpHandlers, &tcpMuxEntry{desen: desen, handler: handler})
}

// HandleUDP, hedef IP ve/veya porta göre UDP paketleri için bir işleyici kaydeder
//
// Desen bir ana bilgisayar adı, bir port, bir ana bilgisayar adı:port veya "*" her şey için olabilir. Portlar iki nokta üst üste ile başlar. Geçerli desenler:
//   - "example.com"
//   - "example.com:80"
//   - ":80"
//   - "*"
//
// Daha sonra bu net.Listen gibi olacak
func (s *mux) HandleUDP(desen string, handler udpHandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.udpHandlers = append(s.udpHandlers, &udpMuxEntry{desen: desen, handler: handler})
}

// notifyTCP, yeni bir akış oluşturulduğunda çağrılır. Verilen akışı kabul edecek ilk dinleyiciyi bulur. Asla engellemez.
func (s *mux) notifyTCP(req TCPRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entry := range s.tcpHandlers {
		if desenEşleşiyorMu(entry.desen, req.LocalAddr()) {
			go entry.handler(req)
			return
		}
	}

	verbosef("tcp için %v'ye kimse dinlemiyor, düşüyor", req.LocalAddr())
}

// notifyUDP, yeni bir paket geldiğinde çağrılır. Paketi eşleşen bir desene sahip ilk işleyiciyi bulur ve paketi ona teslim eder
func (s *mux) notifyUDP(w udpResponder, packet *udpPacket) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entry := range s.udpHandlers {
		if desenEşleşiyorMu(entry.desen, packet.dst) {
			go entry.handler(w, packet)
			return
		}
	}

	verbosef("udp için %v'ye kimse dinlemiyor, düşüyor!", packet.dst)
}

// udpResponder, UDP paketlerini geri yazmak için arayüzdür
type udpResponder interface {
	// alt sürece bir UDP paketi geri yaz
	Write(payload []byte) (n int, err error)
}

// tcpListener, bir mux tarafından yönlendirilen bağlantılar için net.Listener uygular
type tcpListener struct {
	desen       string
	connections chan net.Conn
}

// Accept, kesilen bir bağlantıyı kabul eder. Bu, net.Listener.Accept'i uygular
func (l *tcpListener) Accept() (net.Conn, error) {
	stream := <-l.connections
	if stream == nil {
		// bu, kanalın kapalı olduğu anlamına gelir, bu da tcpStack'in kapatıldığı anlamına gelir
		return nil, net.ErrClosed
	}
	return stream, nil
}

// net.Listener arayüzü için
func (l *tcpListener) Close() error {
	// TODO: yığından kaydını sil, sonra close(l.connections)
	verbose("tcpListener.Close() uygulanmadı, göz ardı ediliyor")
	return nil
}

// net.Listener arayüzü için, bağlantımızın tarafını döndürür
func (l *tcpListener) Addr() net.Addr {
	verbose("tcpListener.Addr() çağrıldı, sahte adres 0.0.0.0:0 döndürülüyor")
	// gerçekte gerçek bir adresimiz yok -- herhangi bir yere giden her şeyi dinliyoruz
	return &net.TCPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 0}
}
