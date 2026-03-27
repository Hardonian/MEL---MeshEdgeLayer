package web

import (
	"embed"
)

//go:embed ../../frontend/dist
var staticAssets embed.FS
