// Package controller provides an `embed.FS` containing the assets for the multiscreen web server controller web application.
package controller

import (
	"embed"
)

//go:embed *.html *.css *.js
var FS embed.FS
