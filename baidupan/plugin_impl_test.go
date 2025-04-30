package main

import (
	"encoding/json"
	"testing"
)

func TestResponse(t *testing.T) {
	dd := `{
    "avatar_url": "https://dss0.bdstatic.com/7Ls0a8Sm1A5BphGlnYG/sys/portrait/item/netdisk.1.3d20c095.phlucxvny00WCx9W4kLifw.jpg",
    "baidu_name": "百度用户A001",
    "errmsg": "succ",
    "errno": 0,
    "netdisk_name": "netdiskuser",
    "request_id": "674030589892501935",
    "uk": 208281036,
    "vip_type": 2
}`
	type UserInfo struct {
		BaiduName string `json:"baidu_name"`
		Uk        int    `json:"uk"`
	}
	uinfo := UserInfo{}
	resp := Response{}
	t.Log(json.Unmarshal([]byte(dd), &resp))
	t.Logf("%+v", uinfo)
}
