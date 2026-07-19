package sitegen

import (
	"embed"
	"io/fs"
)

//go:embed templates/*.html assets/*
var embedded embed.FS

// TemplateFS exposes the embedded templates/ subtree.
func TemplateFS() fs.FS {
	sub, err := fs.Sub(embedded, "templates")
	if err != nil {
		panic(err) // compile-time embed; can't fail at runtime
	}
	return sub
}

// AssetFS exposes the embedded assets/ subtree (CSS, favicon, robots.txt).
func AssetFS() fs.FS {
	sub, err := fs.Sub(embedded, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}
