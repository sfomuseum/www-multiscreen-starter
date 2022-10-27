// Package receiver provides an `embed.FS` containing the assets for the multiscreen web server receiver web application.
package receiver

import (
	"embed"
)

//go:embed *.html *.css *.js
var FS embed.FS
