package main

import (
	"testing"

	"github.com/medianexapp/gowebdav"
	"github.com/medianexapp/plugin_api/httpclient"
)

func TestWebdav(t *testing.T) {
	client := gowebdav.NewClient("http://127.0.0.1:5244/dav/", "admin", "password")
	cc := httpclient.NewClient()
	client.SetClientDo(cc.Client.Do)
	err := client.Connect()
	if err != nil {

		t.Fatal(err)
	}
	dirs, err := client.Stat("/")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(dirs)
	path := "/tianyi/%E6%88%91%E7%9A%84%E8%A7%86%E9%A2%91/%E7%BB%9D%E5%AF%B9%E6%9D%83%E5%8A%9B%5B%E7%AE%80%E7%B9%81%E8%8B%B1%E5%AD%97%E5%B9%95%5D.Absolute.Power.1997.EUR.1080p.BluRay.x265.10bit.DTS-SONYHD/Absolute.Power.1997.EUR.1080p.BluRay.x265.10bit.DTS-SONYHD.mkv"

	t.Log(client.GetPathRequest(path))
}
