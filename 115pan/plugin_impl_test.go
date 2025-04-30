package main

import (
	"image/png"
	"os"
	"testing"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
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
