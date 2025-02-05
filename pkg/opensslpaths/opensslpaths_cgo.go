//go:build cgo
// +build cgo

package opensslpaths

import (
	"reflect"
	"sync"

	"github.com/ebitengine/purego"
)

// Purego, cgo olmadan harici kütüphaneleri yüklemenin bir yolunu sağlar,
// ancak bu, bu yürütülebilir dosyaların libc'ye dinamik olarak bağlanmasına neden olur,
// bu da amacını biraz bozar. Burada, yalnızca CGO etkinleştirildiğinde kullanıyoruz çünkü
// kullandığımız şey, dinamik kütüphaneleri yüklemek ve bulunamadıklarında varsayılan davranışa geri dönmektir.

var libcrypto = sync.OnceValue[*libcryptoFuncs](func() *libcryptoFuncs {
	defer recover()

	libcrypto, err := purego.Dlopen("libcrypto.so", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return nil
	}

	var funcs libcryptoFuncs
	v := reflect.ValueOf(&funcs).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		purego.RegisterLibFunc(v.Field(i).Addr().Interface(), libcrypto, t.Field(i).Name)
	}

	return &funcs
})
