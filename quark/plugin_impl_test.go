package main

import (
	"net/url"
	"strings"
	"testing"

	"github.com/medianexapp/plugin_api/plugin"
)

func TestPluginImpl(t *testing.T) {
	p := NewPluginImpl()
	auth, _ := p.GetAuth()
	method := auth.AuthMethods[0].Method
	method = method // <= save auth data
	method.(*plugin.AuthMethod_Formdata).Formdata.FormItems[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value = ""
	authData, err := p.CheckAuthMethod(&plugin.AuthMethod{
		Method: auth.AuthMethods[0].Method,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = p.CheckAuthData(authData.AuthDataBytes)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.GetDirEntry(&plugin.GetDirEntryRequest{
		Path:     "/",
		Page:     1,
		PageSize: 50,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, fileEntry := range resp.FileEntries {
		if fileEntry.FileType != plugin.FileEntry_FileTypeFile {
			continue
		}
		t.Log("file entry name", fileEntry.Name)
		// if is movie get file resource
		if strings.HasSuffix(fileEntry.Name, "mp4") || strings.HasSuffix(fileEntry.Name, "mkv") {
			fileResource, err := p.GetFileResource(&plugin.GetFileResourceRequest{
				FilePath:  "/" + fileEntry.Name,
				FileEntry: fileEntry,
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("get file %s fileResource %+v", fileEntry.Name, fileResource.FileResourceData)
		}
	}
	return
}

func TestGetExpire(t *testing.T) {
	u := `https://video-play-c-zb-cf.pds.quark.cn/47R9mNu0/4819440970/c015106770004af7978b0744e3b42144680e36f1/680e36f1e16b6c5d23344effac4468ac54762a4c?Expires=1745837657&OSSAccessKeyId=LTAI5tJJpWQEfrcKHnd1LqsZ&Signature=GfTRz5CRz8u1AficCXdKlsYdQrU%3D&x-oss-traffic-limit=503316480&callback-var=eyJ4OmF1IjoiLSIsIng6dWQiOiIxNi00LTEtMi0xLTUtNy1OLTEtMTYtMi1OIiwieDpzcCI6IjM5NyIsIng6dG9rZW4iOiI0LTJkNTUyNTYzYTE5ZGEzMmNjMWVlZmJkNzJkNDgyYzEzLTgtMi0xMTkxLTcyODFfYzZhNTdkNDE3OGY4MjcxYzFhMThkODBiZDEzOTdmZWEtMC0wLTAtMC1hN2Y0YzU0YjllYTY2MDY5MzQ4NjE1YTA5ZWJiNzU5ZiIsIng6dHRsIjoiMTU4NDMifQ%3D%3D&flag=co&callback=eyJjYWxsYmFja0JvZHlUeXBlIjoiYXBwbGljYXRpb24vanNvbiIsImNhbGxiYWNrU3RhZ2UiOiJiZWZvcmUtZXhlY3V0ZSIsImNhbGxiYWNrRmFpbHVyZUFjdGlvbiI6Imlnbm9yZSIsImNhbGxiYWNrVXJsIjoiaHR0cHM6Ly9kcml2ZS1hdXRoLnF1YXJrLmNuL291dGVyL29zcy9jaGVja3BsYXkiLCJjYWxsYmFja0JvZHkiOiJ7XCJob3N0XCI6JHtodHRwSGVhZGVyLmhvc3R9LFwic2l6ZVwiOiR7c2l6ZX0sXCJyYW5nZVwiOiR7aHR0cEhlYWRlci5yYW5nZX0sXCJyZWZlcmVyXCI6JHtodHRwSGVhZGVyLnJlZmVyZXJ9LFwiY29va2llXCI6JHtodHRwSGVhZGVyLmNvb2tpZX0sXCJtZXRob2RcIjoke2h0dHBIZWFkZXIubWV0aG9kfSxcImlwXCI6JHtjbGllbnRJcH0sXCJwb3J0XCI6JHtjbGllbnRQb3J0fSxcIm9iamVjdFwiOiR7b2JqZWN0fSxcInNwXCI6JHt4OnNwfSxcInVkXCI6JHt4OnVkfSxcInRva2VuXCI6JHt4OnRva2VufSxcImF1XCI6JHt4OmF1fSxcInR0bFwiOiR7eDp0dGx9LFwiZHRfc3BcIjoke3g6ZHRfc3B9LFwiaHNwXCI6JHt4OmhzcH0sXCJjbGllbnRfdG9rZW5cIjoke3F1ZXJ5U3RyaW5nLmNsaWVudF90b2tlbn19In0%3D&cht=_9-co&ud=16-4-1-2-1-5-7-N-1-16-2-N`
	p, _ := url.Parse(u)
	t.Log(p.Query().Get("Expires"))

	u = `https://video-play-h-zb.drive.quark.cn/qv/B61C0D1DB07155A409C0A8FFB94051897EB8F81F_3169275463__sha1_sz3_2b3e8cfa/db4aa08bb007dfaf/media.m3u8?auth_key=1745839379-63295-10800-7cb455c1a9d9b7039391fc5aed4e9d48&sp=55&token=4-2-2-100-55-4c2e_febcc5b9492664c51667b33290df69d5-58bc8e6f68cb6553cc097269e6cd6f6e-1745828579760-56596b2c0415546448e4d15ea82ec3e1&ud=16-0-1-2-1-0-7-N-1-16-0-N&mt=3&ct=2ykXsCxDXp8bZfTrdmpjuAFFPXYWBL%2BUnc3OnjMIELw%3D`
	p, _ = url.Parse(u)
	t.Log(strings.Split(p.Query().Get("auth_key"), "-")[0])
}
