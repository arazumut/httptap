//go:build tools
// +build tools

package harlog

// https://github.com/golang/go/issues/25922#issuecomment-412992431 adresinden alınmıştır

import (
	_ "golang.org/x/lint/golint"         // Kod kalitesini kontrol etmek için kullanılan bir araç
	_ "golang.org/x/tools/cmd/goimports" // Kod formatlama aracı
)
