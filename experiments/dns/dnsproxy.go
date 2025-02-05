package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/miekg/dns"
)

const upstreamDNS = "1.1.1.1:53" // TODO: resolv.conf ve nsswitch.conf dosyalarından al

// handle fonksiyonu IPv4 hostlarını net.DefaultResolver kullanarak çözer

func handle(requestMsg *dns.Msg) ([]dns.RR, error) {
	if len(requestMsg.Question) == 0 {
		return nil, nil // bu, cevap yok, hata yok demektir, bu da sorun değil
	}

	soru := requestMsg.Question[0]
	log.Printf("dns isteği alındı: %v", soru.Name)

	ctx := context.Background()

	// isteği kendimiz işleyelim
	switch soru.Qtype {
	case dns.TypeA:
		ipler, err := net.DefaultResolver.LookupIP(ctx, "ip4", soru.Name)
		if err != nil {
			return nil, fmt.Errorf("varsayılan çözücü dedi ki: %w", err)
		}

		var rrs []dns.RR
		for _, ip := range ipler {
			rrline := fmt.Sprintf("%s A %s", soru.Name, ip)
			rr, err := dns.NewRR(rrline)
			if err != nil {
				return nil, fmt.Errorf("rr oluşturulurken hata: %w", err)
			}
			rrs = append(rrs, rr)
		}
		return rrs, nil
	}

	log.Println("istek proxyleniyor...")

	// isteği başka bir sunucuya proxyle
	queryMsg := new(dns.Msg)
	requestMsg.CopyTo(queryMsg)
	queryMsg.Question = []dns.Question{soru}

	dnsClient := new(dns.Client)
	dnsClient.Net = "udp"
	cevap, _, err := dnsClient.Exchange(queryMsg, upstreamDNS)
	if err != nil {
		return nil, err
	}

	log.Printf("upstream dns sunucusundan %d cevap alındı", len(cevap.Answer))

	if len(cevap.Answer) > 0 {
		return cevap.Answer, nil
	}
	return nil, fmt.Errorf("bulunamadı")
}

func Main() error {
	var args struct {
		Port string `arg:"positional"`
	}
	arg.MustParse(&args)

	dns.HandleFunc(".", func(w dns.ResponseWriter, req *dns.Msg) {
		switch req.Opcode {
		case dns.OpcodeQuery:
			rrs, err := handle(req)
			if err != nil {
				log.Printf("dns başarısız oldu: %s, hata: %v, devam ediliyor...", req, err.Error())
				// burada durma, devam et
			}

			resp := new(dns.Msg)
			resp.SetReply(req)
			resp.Answer = rrs
			w.WriteMsg(resp)
		}
	})

	server := &dns.Server{Addr: args.Port, Net: "udp"}
	server.ListenAndServe()
	log.Printf("dinleniyor: %v...", server.Addr)
	return server.ListenAndServe()
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	err := Main()
	if err != nil {
		log.Fatal(err)
	}
}
