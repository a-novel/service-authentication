// Package assets embeds the static files the email templates need and exposes them
// in the encoding those templates consume.
package assets

import (
	_ "embed"
	"encoding/base64"
)

//go:embed banner.png
var banner []byte

// BannerBase64 is the banner image as a base64 data URI, ready to drop into an HTML
// image source so the email renders the banner inline without fetching a remote asset.
var BannerBase64 = "data:image/png;base64," + base64.StdEncoding.EncodeToString(banner)
