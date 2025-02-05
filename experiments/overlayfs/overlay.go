package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"
	"golang.org/x/sys/unix"
)

func Ana() error {
	var args struct {
		Komut []string `arg:"positional"`
	}
	arg.MustParse(&args)

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	altDizin := filepath.Join(cwd, "alt")
	ustDizin := filepath.Join(cwd, "ust")
	calismaDizin := filepath.Join(cwd, "calisma")
	birlesikDizin := filepath.Join(cwd, "birlesik")

	for _, dir := range []string{altDizin, ustDizin, calismaDizin, birlesikDizin} {
		_ = os.MkdirAll(dir, os.ModeDir)
	}

	// bir overlay dosya sistemi bağla
	// sudo mount -t overlay overlay -olowerdir=$(pwd)/lower,upperdir=$(pwd)/upper,workdir=$(pwd)/work $(pwd)/merged
	baglantiSecenekleri := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", altDizin, ustDizin, calismaDizin)
	err = unix.Mount("overlay", birlesikDizin, "overlay", 0, baglantiSecenekleri)
	if err != nil {
		return fmt.Errorf("overlay dosya sistemi bağlanırken hata oluştu: %w", err)
	}

	return nil
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	err := Ana()
	if err != nil {
		log.Fatal(err)
	}
}
