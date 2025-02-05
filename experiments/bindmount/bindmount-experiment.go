package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/alexflint/go-arg"
	"github.com/monasticacademy/httptap/pkg/bindfiles"
)

// Main fonksiyonu, ana işlevi yerine getirir
func Main() error {
	var args struct {
		Command []string `arg:"positional"`
	}
	arg.MustParse(&args)

	// Eğer komut verilmemişse, varsayılan olarak /bin/sh kullan
	if len(args.Command) == 0 {
		args.Command = []string{"/bin/sh"}
	}

	// Bu goroutine'i tek bir OS iş parçacığına kilitle, çünkü ad alanları iş parçacığına özgüdür
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Yeni bir mount ad alanına git ve bu ad alanında kök dosya sistemini üst üste bind et
	mount, err := bindfiles.Mount(
		bindfiles.File("/etc/resolv.conf", []byte("hello bindfiles\n")),
	)
	if err != nil {
		return fmt.Errorf("bind-mounting hatası: %w", err)
	}
	defer mount.Remove()

	// Bir alt süreç başlat -- zaten ad alanındayız, bu yüzden burada CLONE_NS gerekmez
	cmd := exec.Command(args.Command[0])
	cmd.Args = args.Command
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = []string{"PS1=MOUNTNAMESPACE # "}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("alt süreç başlatma hatası: %w", err)
	}

	// Alt sürecin tamamlanmasını bekle
	err = cmd.Wait()
	if err != nil {
		exitError, isExitError := err.(*exec.ExitError)
		if isExitError {
			return fmt.Errorf("alt süreç %d kodu ile çıktı", exitError.ExitCode())
		} else {
			return fmt.Errorf("alt süreç çalıştırma hatası: %v", err)
		}
	}
	return nil
}

// Ana fonksiyon
func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	err := Main()
	if err != nil {
		log.Fatal(err)
	}
}
