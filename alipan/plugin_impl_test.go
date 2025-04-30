package main

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPluginImpl(t *testing.T) {
	u := "mediagate://alipan/?code=bcb52515540f43fba4d791e6860943ad&state=1741361519159140"
	uuu, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", uuu.Query().Get("code"))

	t.Log(strings.CutPrefix("/资源库/Home/xxx/xxx", "/资源库"))

	// "2025-04-11T02:45:56.240Z"
	t.Log(time.Parse("2006-01-02T15:04:05Z", "2025-04-11T02:45:56.240Z"))
}
