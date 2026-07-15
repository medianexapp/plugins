package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/medianexapp/plugin_api/plugin"
)

func TestPluginImpl(t *testing.T) {

	p := NewPluginImpl()
	auth, err := p.GetAuth()

	if err != nil {
		t.Fatal(err)
	}

	if len(auth.AuthMethods) != 2 {
		t.Fatal("auth methods count != 2")
	}

	cookie := plugin.String(``)

	auth.AuthMethods[0].Method.(*plugin.AuthMethod_Formdata).Formdata.FormItems[0] = &plugin.Formdata_FormItem{
		Value: cookie,
	}
	d, err := p.CheckAuthMethod(auth.AuthMethods[0])
	if err != nil {
		t.Fatal("err", err)
	}

	err = p.CheckAuthData(d.AuthDataBytes)
	if err != nil {
		t.Fatal(err)
	}
	// t.Log(p.convertCookie(p.cookies))

	dirEntryResp, err := p.GetDirEntry(&plugin.GetDirEntryRequest{
		Path:     "/",
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range dirEntryResp.FileEntries {
		if entry.FileType != plugin.FileEntry_FileTypeFile {
			continue
		}
		fileResResp, err := p.GetFileResource(&plugin.GetFileResourceRequest{
			FilePath:  "/" + entry.Name,
			FileEntry: entry,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("file resourece %+v\n", fileResResp.FileResourceData[0])
	}

}

func TestQuarkQrcode(t *testing.T) {
	p := NewPluginImpl()

	token, err := p.getQrcodeToken()
	if err != nil {
		t.Fatal(err)
	}

	// 直接输出到控制台
	fmt.Println(p.qrcodeData(token))
	for {
		res, err := p.checkQrcode(token)
		if err == nil && res != nil {
			t.Log("cookie", res)
			break
		}
		time.Sleep(time.Second * 2)
	}

	fmt.Println("cookie", p.hb.GetCookies())
}
