// package overlayroot kök dosya sisteminin mevcut işleme erişilebilir görünümünü değiştirir

package overlayroot

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// Bakeable geçici bir dizine kendini damgalar
type Bakeable interface {
	Bake(overlayroot string) error
}

// file orijinal dosya sistemindeki bir dosyanın içeriğini sabit içerikle değiştirir
type file struct {
	path    string
	content []byte
	perm    os.FileMode
}

// Bake bireysel dosyalar için Bakeable.Bake'i uygular
func (f *file) Bake(overlayroot string) error {
	path := filepath.Join(overlayroot, f.path)

	err := os.MkdirAll(filepath.Dir(path), f.perm)
	if err != nil {
		return err
	}

	return os.WriteFile(path, f.content, f.perm)
}

// File içeriği bir string ile belirtilen bir dosyadır
func File(path string, content []byte) *file {
	return &file{path: path, content: content, perm: os.ModePerm}
}

// FilePerm, File gibidir ancak izinleri belirtebilirsiniz
func FilePerm(path string, content []byte, perm os.FileMode) *file {
	return &file{path: path, content: content, perm: perm}
}

// Remover pivot_root edilmiş bir overlay dosya sistemini destekleyen belirli geçici dosyaların konumunu tutar.
// Bu yapının ana amacı, sonrasında temizlemektir.
type Remover struct {
	tmpdir string
}

// Remove, Pivot tarafından oluşturulan geçici dizini temizler
func (m *Remover) Remove() error {
	return os.RemoveAll(m.tmpdir)
}

func Pivot(nodes ...Bakeable) (*Remover, error) {
	// geçici bir dizin oluştur
	tmpdir, err := os.MkdirTemp("", "overlay-root-*")
	if err != nil {
		return nil, fmt.Errorf("geçerli çalışma dizini alınırken hata oluştu")
	}

	// ana mount syscall için bazı yolları hazırla
	newroot := filepath.Join(tmpdir, "merged") // bu bir overlayfs olarak monte edilecek
	oldroot := filepath.Join(newroot, "old")   // pivot_root tarafından eski kökün konulacağı yer
	workdir := filepath.Join(tmpdir, "work")   // overlayfs sürücüsü bunu çalışma dizini olarak kullanacak
	layerdir := filepath.Join(tmpdir, "layer") // bu, köke uygulanacak "diff"i tutan dizindir

	// aşağıdaki syscalls için bu dizinlerin hepsinin zaten var olması gerekiyor
	for _, dir := range []string{newroot, oldroot, layerdir, workdir} {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("dizin oluşturulurken hata oluştu %v: %w", dir, err)
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

	// bu yeni namespace'teki kök dosya sistemini özel yap, bu da aşağıdaki mount'un
	// üst namespace'e sızmasını önler
	// man sayfasına göre, aşağıdaki ilk, üçüncü ve beşinci argümanlar göz ardı edilir
	err = unix.Mount("ignored", "/", "ignored", unix.MS_PRIVATE|unix.MS_REC, "ignored")
	if err != nil {
		return nil, fmt.Errorf("kök dosya sistemi özel yapılırken hata oluştu")
	}

	// bir overlay dosya sistemi monte et
	// sudo mount -t overlay overlay -olowerdir=$(pwd)/lower,upperdir=$(pwd)/upper,workdir=$(pwd)/work $(pwd)/merged
	mountopts := fmt.Sprintf("lowerdir=/,upperdir=%s,workdir=%s", layerdir, workdir)
	err = unix.Mount("overlay", newroot, "overlay", 0, mountopts)
	if err != nil {
		return nil, fmt.Errorf("overlay dosya sistemi monte edilirken hata oluştu: %w", err)
	}

	// dosya sisteminin kökünü overlay olarak ayarla
	err = unix.PivotRoot(newroot, oldroot)
	if err != nil {
		return nil, fmt.Errorf("dosya sisteminin kökü %v olarak değiştirilirken hata oluştu: %w", newroot, err)
	}

	return &Remover{tmpdir: tmpdir}, nil
}

func Mount(nodes ...Bakeable) (*Remover, error) {
	// geçici bir dizin oluştur
	tmpdir, err := os.MkdirTemp("", "overlay-root-*")
	if err != nil {
		return nil, fmt.Errorf("geçerli çalışma dizini alınırken hata oluştu")
	}

	// ana mount syscall için bazı yolları hazırla
	newroot := filepath.Join(tmpdir, "merged") // bu bir overlayfs olarak monte edilecek
	oldroot := filepath.Join(newroot, "old")   // pivot_root tarafından eski kökün konulacağı yer
	workdir := filepath.Join(tmpdir, "work")   // overlayfs sürücüsü bunu çalışma dizini olarak kullanacak
	layerdir := filepath.Join(tmpdir, "layer") // bu, köke uygulanacak "diff"i tutan dizindir

	// aşağıdaki syscalls için bu dizinlerin hepsinin zaten var olması gerekiyor
	for _, dir := range []string{newroot, oldroot, layerdir, workdir} {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("dizin oluşturulurken hata oluştu %v: %w", dir, err)
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

	// bu yeni namespace'teki kök dosya sistemini özel yap, bu da aşağıdaki mount'un
	// üst namespace'e sızmasını önler
	// man sayfasına göre, aşağıdaki ilk, üçüncü ve beşinci argümanlar göz ardı edilir
	err = unix.Mount("ignored", "/", "ignored", unix.MS_PRIVATE|unix.MS_REC, "ignored")
	if err != nil {
		return nil, fmt.Errorf("kök dosya sistemi özel yapılırken hata oluştu")
	}

	// bir overlay dosya sistemi monte et
	// sudo mount -t overlay overlay -olowerdir=$(pwd)/lower,upperdir=$(pwd)/upper,workdir=$(pwd)/work $(pwd)/merged
	mountopts := fmt.Sprintf("lowerdir=/,upperdir=%s,workdir=%s", layerdir, workdir)
	err = unix.Mount("overlay", newroot, "overlay", 0, mountopts)
	if err != nil {
		return nil, fmt.Errorf("overlay dosya sistemi monte edilirken hata oluştu: %w", err)
	}

	log.Printf("overlay %v'ye monte edildi", newroot)

	return &Remover{tmpdir: tmpdir}, nil
}
