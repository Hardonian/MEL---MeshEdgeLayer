package web

import (
	"embed"
	"net/http"
)

//go:embed all:frontend/dist
var staticAssets embed.FS
