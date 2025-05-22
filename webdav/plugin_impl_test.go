package main

import (
	"testing"

	"github.com/medianexapp/gowebdav"
	"github.com/medianexapp/plugin_api/httpclient"
)

func TestWebdav(t *testing.T) {
	client := gowebdav.NewClient("", "", "")
	cc := httpclient.NewClient()
	client.SetClientDo(cc.Do)
	err := client.Connect()
	if err != nil {

		t.Fatal(err)
	}
	dirs, err := client.Stat("/")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(dirs)
	path := `/疯狂动物城 4K原盘REMUX 国英双音 内封字幕 默认国音`

	t.Log(client.ReadDir(path))
}
