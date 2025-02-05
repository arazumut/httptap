// overlay paketi, mevcut bir dizinin üzerine bir overlay dosya sistemi monte eder. pivot_root'a gerek yoktur ve herhangi bir yerde olabilir.

package overlay

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// Bakeable, kendisini geçici bir dizine damgalar
type Bakeable interface {
	Bake(dir string) error
}

// file, orijinal dosya sistemindeki bir dosyanın içeriğini sabit içerikle değiştirir
type file struct {
	path    string
	content []byte
	perm    os.FileMode
}

// Bake, bireysel dosyalar için Bakeable.Bake'i uygular
func (f *file) Bake(dir string) error {
	path := filepath.Join(dir, f.path)

	err := os.MkdirAll(filepath.Dir(path), f.perm)
	if err != nil {
		return err
	}

	return os.WriteFile(path, f.content, f.perm)
}

// File, içeriği bir dize ile belirtilen bir dosyadır
func File(path string, content []byte) *file {
	return &file{path: path, content: content, perm: os.ModePerm}
}

// FilePerm, File gibidir ancak izinleri belirtebilirsiniz
func FilePerm(path string, content []byte, perm os.FileMode) *file {
	return &file{path: path, content: content, perm: perm}
}

// Remover, pivot_root edilmiş overlay dosya sistemini destekleyen belirli geçici dosyaların konumunu tutar. Bu yapının ana amacı, sonrasında temizlemektir.
type Remover struct {
	tmpdir string
}

// Remove, Pivot tarafından oluşturulan geçici dizini temizler
func (m *Remover) Remove() error {
	return os.RemoveAll(m.tmpdir)
}

func Mount(path string, nodes ...Bakeable) (*Remover, error) {
	// geçici bir dizin oluştur
	tmpdir, err := os.MkdirTemp("", "overlay-*")
	if err != nil {
		return nil, fmt.Errorf("geçerli çalışma dizini alınırken hata oluştu")
	}

	// ana mount syscall için bazı yolları hazırla
	workdir := filepath.Join(tmpdir, "work")   // overlayfs sürücüsü bunu çalışma dizini olarak kullanacak
	layerdir := filepath.Join(tmpdir, "layer") // bu, köke uygulanacak "farkı" tutan dizindir

	// aşağıdaki syscalls için bu dizinlerin tümü zaten var olmalıdır
	for _, dir := range []string{layerdir, workdir} {
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return nil, fmt.Errorf("%v dizini oluşturulurken hata oluştu: %w", dir, err)
		}
	}

	// katmanı damgala
	for _, node := range nodes {
		if err := node.Bake(layerdir); err != nil {
			return nil, fmt.Errorf("%T%#v damgalanırken hata oluştu: %w", node, node, err)
		}
	}

	// yeni bir mount namespace'e geç
	err = unix.Unshare(unix.CLONE_NEWNS | unix.CLONE_FS)
	if err != nil {
		return nil, fmt.Errorf("mount'lar ayrılırken hata oluştu: %w", err)
	}

	// bu yeni namespace'teki kök dosya sistemini özel yap, bu da aşağıdaki mount'un üst namespace'e sızmasını önler
	// man sayfasına göre, aşağıdaki ilk, üçüncü ve beşinci argümanlar göz ardı edilir
	err = unix.Mount("ignored", "/", "ignored", unix.MS_PRIVATE|unix.MS_REC, "ignored")
	if err != nil {
		return nil, fmt.Errorf("kök dosya sistemi özel yapılırken hata oluştu")
	}

	// bir overlay dosya sistemi monte et; eşdeğer:
	//
	//   sudo mount -t overlay overlay -olowerdir=<path>,upperdir=<layer>,workdir=<work> <path>
	mountopts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", path, layerdir, workdir)
	err = unix.Mount("overlay", path, "overlay", 0, mountopts)
	if err != nil {
		return nil, fmt.Errorf("%v (%q) üzerine overlay dosya sistemi monte edilirken hata oluştu: %w", path, mountopts, err)
	}

	return &Remover{tmpdir: tmpdir}, nil
}
