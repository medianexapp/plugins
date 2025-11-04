package main

import (
	"testing"

	"github.com/medianexapp/plugin_api/plugin"
)

func TestPluginImpl(t *testing.T) {
	p := NewPluginImpl()
	auth, _ := p.GetAuth()
	method := auth.AuthMethods[0].Method
	authFormData := method.(*plugin.AuthMethod_Formdata)
	authFormData.Formdata.FormItems[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value = "http://127.0.0.1:5244"
	authFormData.Formdata.FormItems[1].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value = "admin"
	authFormData.Formdata.FormItems[2].Value.(*plugin.Formdata_FormItem_ObscureStringValue).ObscureStringValue.Value = "password"
	authFormData.Formdata.FormItems[3].Value.(*plugin.Formdata_FormItem_Int64Value).Int64Value.Value = 48

	authData, err := p.CheckAuthMethod(&plugin.AuthMethod{
		Method: authFormData,
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
		PageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp.FileEntries)

	fileResource, err := p.GetFileResource(&plugin.GetFileResourceRequest{
		FilePath: "/tianyi/我的视频/绝对权力[简繁英字幕].Absolute.Power.1997.EUR.1080p.BluRay.x265.10bit.DTS-SONYHD/Absolute.Power.1997.EUR.1080p.BluRay.x265.10bit.DTS-SONYHD.mkv",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("get file  fileResource %+v", fileResource.FileResourceData[0])
	return
}
