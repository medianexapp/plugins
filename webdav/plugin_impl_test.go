package main_test

import (
	"log/slog"
	"testing"

	"github.com/medianexapp/gowebdav"
	"github.com/medianexapp/plugin_api/httpclient"
)

func TestWebdav(t *testing.T) {
	httpCC := httpclient.NewClient()
	client := gowebdav.NewClient("http://192.168.123.213:8080", "user", "passwd")
	client.SetClientDo(httpCC.Do)
	err := client.Connect()
	if err != nil {
		slog.Error("connect failed", "err", err)

	}
}
