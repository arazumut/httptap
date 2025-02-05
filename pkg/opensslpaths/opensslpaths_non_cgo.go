//go:build !cgo
// +build !cgo

package opensslpaths

import (
	"sync"
)

// libcrypto değişkeni, libcryptoFuncs türünde bir değeri saklar ve bu değer yalnızca bir kez hesaplanır.
var libcrypto = sync.OnceValue[*libcryptoFuncs](func() *libcryptoFuncs {
	return nil
})
