package main

import (
	"log"
	"runtime/debug"
)

// Panik durumlarını ele alan fonksiyon
func panikYakala() {
	if r := recover(); r != nil {
		log.Printf("Panik: %v", r)
		log.Printf("Yığın izi: %s", debug.Stack())
	}
}

// Bir fonksiyonu panik yakalama ile çalıştıran fonksiyon
func guvenliCalistir(f func() error) {
	defer panikYakala()
	err := f()
	if err != nil {
		log.Printf("Bir goroutine hata ile sonlandı: %v", err)
	}
}
