package assets

import (
	_ "embed"
	"encoding/base64"
)

//go:embed banner.png
var banner []byte

var BannerBase64 = "data:image/png;base64," + base64.StdEncoding.EncodeToString(banner)
