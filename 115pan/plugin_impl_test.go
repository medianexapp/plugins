package main

import (
	"encoding/json"
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
	authData := `{}`
	token := &plugin.Token{}
	json.Unmarshal([]byte(authData), token)
	// t.Log(token)
	tokenData, _ := token.MarshalVT()
	err := pluginImpl.CheckAuthData(tokenData)
	if err != nil {
		t.Fatal(err)
	}
	dirEntry, err := pluginImpl.GetDirEntry(&plugin.GetDirEntryRequest{
		Path:     "/",
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, fileEntry := range dirEntry.FileEntries {
		if fileEntry.Name == "test.mkv" {
			fileResource, err := pluginImpl.GetFileResource(&plugin.GetFileResourceRequest{
				FilePath:  "/test.mkv",
				FileEntry: fileEntry,
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("fileRe %+v \n", fileResource)
		}
	}
}
