package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/alexflint/go-arg"
	"github.com/monasticacademy/httptap/pkg/overlayroot"
)

func Ana() error {
	var args struct {
		Komut []string `arg:"positional"`
	}
	arg.MustParse(&args)

	if len(args.Komut) == 0 {
		args.Komut = []string{"/bin/sh"}
	}

	// Bu goroutine'i tek bir OS iş parçacığına kilitle, çünkü ad alanları iş parçacığına özeldir
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Yeni bir bağlama ad alanına git ve bu ad alanında kök dosya sistemini kapla
	mount, err := overlayroot.Pivot(
		overlayroot.File("/a/b/test", []byte("merhaba kaplama kök\n")),
	)
	if err != nil {
		return fmt.Errorf("kök dosya sistemini kaplarken hata: %w", err)
	}
	defer mount.Remove()

	// Bir alt süreç başlat -- zaten ad alanındayız, bu yüzden burada CLONE_NS gerekmez
	cmd := exec.Command(args.Komut[0])
	cmd.Args = args.Komut
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = []string{"PS1=MOUNTNAMESPACE # "}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("alt süreci başlatırken hata: %w", err)
	}

	// Alt sürecin tamamlanmasını bekle
	err = cmd.Wait()
	if err != nil {
		exitError, isExitError := err.(*exec.ExitError)
		if isExitError {
			return fmt.Errorf("alt süreç %d kodu ile çıktı", exitError.ExitCode())
		} else {
			return fmt.Errorf("alt süreci çalıştırırken hata: %v", err)
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
