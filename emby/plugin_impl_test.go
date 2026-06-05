package main

import (
	"testing"

	"github.com/medianexapp/plugin_api/plugin"
)

// emby
// 电影没有自动集合  Type : "Movie" 刚好对应play info
// 电视剧默认展示 Type: "Series" 对应 play seriers ，转换为play info，在relation里展示emby的所有季
//

func TestPluginImpl(t *testing.T) {
	p := NewPluginImpl()

	p.embyAuth.Addr.StringValue.Value = "https://emby.bangumi.ca"
	p.embyAuth.User.StringValue.Value = "labulakalia"
	p.embyAuth.Password.ObscureStringValue.Value = "Ww123456@@@"

	auth, err := p.GetAuth()
	if err != nil {
		t.Fatal(err)
	}

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

	// 1. Test GetPluginMenus
	menus, err := p.GetPluginMenus()
	if err != nil {
		t.Fatal(err)
	}
	if len(menus.GetPluginMenus()) == 0 {
		t.Fatal("no menus returned")
	}
	t.Logf("menus %+v\n", menus.GetPluginMenus())

	// 2. Test GetPluginFilterItems
	resp, err := p.GetPluginFilterItems(menus.GetPluginMenus()[1].Menu)
	if err != nil {
		t.Fatalf("GetPluginFilterItems for menu '%s' failed: %v", menus.GetPluginMenus()[0].Menu, err)
	}
	for _, filters := range resp.Filters {
		t.Logf("filter %+v\n", filters)
	}

	mediaInfosResp, err := p.ListPluginMediaItemInfo(&plugin.ListPluginMediaInfoRequest{
		Menu:     menus.GetPluginMenus()[1].Menu,
		PageSize: 50,
		Page:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("mediaInfosResp %+v\n", mediaInfosResp)

}
