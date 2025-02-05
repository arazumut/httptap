// bindfiles paketi, bir mount namespace oluşturur ve bu namespace içinde belirli içeriklere sahip
// geçici dosyalara dosyaları bağlar. Bu, mevcut dosya sisteminin belirli değişikliklerle bir görünümünü
// oluşturmak için kullanılabilir ve pivot_root ile tüm dosya sisteminde izin sorunları olmadan yapılabilir.

package bindfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// file, orijinal dosya sistemindeki bir dosyanın içeriğini sabit içeriklerle değiştirir
type file struct {
	path    string
	content []byte
	perm    os.FileMode
}

// File, içeriği bir string ile belirtilen bir dosyadır
func File(path string, content []byte) *file {
	return &file{path: path, content: content, perm: os.ModePerm}
}

// FilePerm, File gibidir ancak izinleri belirtebilirsiniz
func FilePerm(path string, content []byte, perm os.FileMode) *file {
	return &file{path: path, content: content, perm: perm}
}

// Remover, pivot_root edilmiş bir overlay dosya sistemini destekleyen belirli geçici dosyaların
// konumunu tutar. Bu yapının ana amacı, sonrasında temizlemektir.
type Remover struct {
	tmpdir string
	mounts []string
}

// Remove, geçici dizini temizler ve bağlamaları kaldırır
func (m *Remover) Remove() error {
	var errmsgs []string
	for _, mount := range m.mounts {
		err := unix.Unmount(mount, 0)
		if err != nil {
			errmsgs = append(errmsgs, err.Error())
		}
	}
	if len(errmsgs) > 0 {
		return fmt.Errorf("bağlamaları kaldırırken %d hata oluştu: %v", len(errmsgs), strings.Join(errmsgs, ", "))
	}

	return os.RemoveAll(m.tmpdir)
}

func Mount(files ...*file) (*Remover, error) {
	// geçici bir dizin oluştur
	tmpdir, err := os.MkdirTemp("", "overlay-root-*")
	if err != nil {
		return nil, fmt.Errorf("geçerli çalışma dizini alınırken hata oluştu")
	}

	remover := Remover{tmpdir: tmpdir}

	// yeni bir mount namespace'e geç
	err = unix.Unshare(unix.CLONE_NEWNS | unix.CLONE_FS)
	if err != nil {
		return &remover, fmt.Errorf("mount'lar paylaşılırken hata oluştu: %w", err)
	}

	// bu yeni namespace içindeki kök dosya sistemini özel yap, bu da aşağıdaki mount'un
	// üst namespace'e sızmasını engeller
	// man sayfasına göre, aşağıdaki ilk, üçüncü ve beşinci argümanlar yok sayılır
	err = unix.Mount("ignored", "/", "ignored", unix.MS_PRIVATE|unix.MS_REC, "ignored")
	if err != nil {
		return &remover, fmt.Errorf("kök dosya sistemi özel yapılırken hata oluştu")
	}

	// her dosyayı bind-mount yap
	for i, file := range files {
		path := filepath.Join(tmpdir, fmt.Sprintf("%08d_%s", i, filepath.Base(file.path)))
		err := os.WriteFile(path, file.content, file.perm)
		if err != nil {
			return &remover, fmt.Errorf("%v için geçici dosya oluşturulurken hata oluştu: %w", file.path, err)
		}

		// hedefin var olduğundan ve bir dosya olduğundan emin ol
		st, err := os.Stat(file.path)
		if err != nil && !os.IsNotExist(err) {
			return &remover, fmt.Errorf("%v kontrol edilirken hata oluştu: %w", file.path, err)
		}
		if err == nil && !st.Mode().IsRegular() {
			return &remover, fmt.Errorf("%v normal bir dosya değil (bulunan %v)", file.path, st.Mode())
		}

		// bind-mount yap -- aşağıdaki üçüncü ve beşinci parametreler yok sayılır
		err = unix.Mount(path, file.path, "==ignored==", unix.MS_BIND, "==ignored==")
		if err != nil {
			return &remover, fmt.Errorf("%v, %v olarak bind-mount yapılırken hata oluştu: %w", file.path, path, err)
		}

		remover.mounts = append(remover.mounts, file.path)
	}

	return &remover, nil
}
