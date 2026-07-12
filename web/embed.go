// Package web embute o build do frontend (Vite + React, gerado em web/dist
// por `task web-build`) para ser servido pelo binário Go de cmd/play.
package web

import "embed"

//go:embed all:dist
var Dist embed.FS