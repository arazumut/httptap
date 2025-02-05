package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/alexflint/go-arg"
	"github.com/songgao/water"
)

// numThreads fonksiyonu, mevcut süreçteki iş parçacığı sayısını döndürür
func numThreads() int {
	st, err := os.Stat("/proc/self/task")
	if err != nil {
		log.Fatal("stat: ", err)
	}
	return int(st.Sys().(*syscall.Stat_t).Nlink)
}

// Main fonksiyonu, ana işlevi yerine getirir
func Main() error {
	var args struct {
		Command []string `arg:"positional"`
	}
	arg.MustParse(&args)

	log.Println(os.Args)

	var err error

	// İlk olarak, kendimizi yeni bir kullanıcı ad alanında yeniden çalıştırıyoruz
	if os.Args[0] != "/proc/self/exe" {
		log.Println("Yeniden çalıştırılıyor...")
		// Alt süreç başlat -- zaten ağ ad alanındayız, bu yüzden burada özel bir şey yok
		cmd := exec.Command("/proc/self/exe")
		cmd.Args = append([]string{"/proc/self/exe"}, os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWNET | syscall.CLONE_NEWUSER,
			UidMappings: []syscall.SysProcIDMap{{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			}},
			GidMappings: []syscall.SysProcIDMap{{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			}},
		}
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("kendimizi yeni bir kullanıcı ad alanında yeniden çalıştırma hatası: %w", err)
		}
		return nil
	}

	log.Println("İç seviyedeyiz, bir tun cihazı oluşturuluyor...")

	// Yeni ad alanında bir tun cihazı oluştur
	tun, err := water.New(water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: "tun",
		},
	})
	if err != nil {
		return fmt.Errorf("tun cihazı oluşturma hatası: %w", err)
	}

	_ = tun

	// Alt süreç çalıştır
	// Alt süreç için ortam değişkenlerini ayarla
	env := append(
		os.Environ(),
		"PS1=HTTPTAP # ",
		"HTTPTAP=1",
	)

	log.Println("Çalıştırılacak komut:", args.Command)

	// Alt süreç başlat -- zaten ağ ad alanındayız, bu yüzden burada özel bir şey yok
	cmd := exec.Command(args.Command[0])
	cmd.Args = args.Command
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("alt süreci başlatma hatası: %w", err)
	}

	return nil
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	err := Main()
	if err != nil {
		log.Fatal(err)
	}
}
