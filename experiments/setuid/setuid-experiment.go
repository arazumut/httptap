package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/alexflint/go-arg"
	"golang.org/x/sys/unix"
)

func Ana() error {
	var args struct {
		Kullanici int
		Grup      int
		Komut     []string `arg:"positional"`
	}
	arg.MustParse(&args)

	// UID değiştirdikten sonra kendi GID'imizi değiştirme yeteneğimizi kaybedeceğiz,
	// ancak ters yönde çalışıyor, bu yüzden önce GID'yi ayarlayın
	err := unix.Setgid(args.Grup)
	if err != nil {
		log.Printf("Grup %v'ye geçerken hata: %v", args.Grup, err)
	}

	err = unix.Setuid(args.Kullanici)
	if err != nil {
		log.Printf("Kullanıcı %q'ya geçerken hata: %v", args.Kullanici, err)
	}

	// Bir alt süreç başlat -- zaten ağ ad alanındayız, bu yüzden burada özel bir şey yok
	cmd := exec.Command(args.Komut[0])
	cmd.Args = args.Komut
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = []string{"PS1=SETUID # ", "HTTPTAP=1"}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("Alt süreç başlatılırken hata: %w", err)
	}

	// Alt sürecin tamamlanmasını bekle
	err = cmd.Wait()
	if err != nil {
		exitError, isExitError := err.(*exec.ExitError)
		if isExitError {
			return fmt.Errorf("Alt süreç %d kodu ile çıktı", exitError.ExitCode())
		} else {
			return fmt.Errorf("Alt süreç çalıştırılırken hata: %v", err)
		}
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
