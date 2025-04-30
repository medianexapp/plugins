package main

import (
	"encoding/base64"
	"image/png"
	"os"
	"testing"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/medianexapp/plugin_api/plugin"
)

func TestGenerateQrcode(t *testing.T) {
	// Create the barcode
	qrCode, _ := qr.Encode("Hello World", qr.M, qr.Auto)

	// Scale the barcode to 200x200 pixels
	qrCode, _ = barcode.Scale(qrCode, 200, 200)

	defer os.Remove("qrcode.png")

	// create the output file
	file, _ := os.Create("qrcode.png")
	defer file.Close()
	// encode the barcode as png
	png.Encode(file, qrCode)
}

func TestAuth(t *testing.T) {
	pluginImpl := NewPluginImpl()
	auth, err := pluginImpl.GetAuth()
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range auth.AuthMethods {
		t.Log(method)
	}

}

func TestBase64(t *testing.T) {

	token := &plugin.Token{
		AccessToken:  "abcgsdjdiwjksko",
		RefreshToken: "dwefojeirfpeijrfpwsoerf'pokp'sfoerf",
	}
	tokenData, _ := token.MarshalVT()
	t.Log(tokenData)
	base64Res := base64.URLEncoding.EncodeToString(tokenData)
	t.Log(base64Res)

	t.Log(base64.URLEncoding.DecodeString(base64Res))
	// Eg14eHh4eHh4eHh4eHh4Ggx4eHh4eHh4eHh4eHg=

}
