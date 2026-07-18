package main

import (
	"testing"

	"github.com/medianexapp/plugin_api/plugin"
)

func TestImpl(t *testing.T) {
	impl := NewPluginImpl()
	auth, _ := impl.GetAuth()
	formData := auth.AuthMethods[0].Method.(*plugin.AuthMethod_Formdata).Formdata

	formData.FormItems[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value = "https://live.zbds.top/tv/iptv4.m3u"
	authData, err := impl.CheckAuthMethod(&plugin.AuthMethod{
		Method: &plugin.AuthMethod_Formdata{
			Formdata: formData,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = impl.CheckAuthData(authData.AuthDataBytes)
	if err != nil {
		t.Fatal(err)
	}
	menus, err := impl.GetPluginMenus()
	if err != nil {
		t.Fatal(err)
	}
	menu := menus.PluginMenus[0].Menu
	resp, err := impl.ListPluginMediaInfo(&plugin.ListPluginMediaInfoRequest{
		Menu: menu,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ListPluginMediaInfo %+v\n", resp.MediaInfos[0])
	mediaDetailResp, err := impl.GetPluginMediaDetail(&plugin.GetPluginMediaDetailRequest{
		MediaInfoId: resp.MediaInfos[0].PluginMediaId,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("GetPluginMediaDetail resp %+v\n", mediaDetailResp.MediaItems)

	getFileResourceResp, err := impl.GetFileResource(&plugin.GetFileResourceRequest{
		MediaPlayId: mediaDetailResp.MediaItems[0].PluginMediaId,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("GetFileResource resp %+v\n", getFileResourceResp.FileResourceData)

}
