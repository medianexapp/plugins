package main

import (
	"testing"

	"github.com/medianexapp/plugin_api/plugin"
)

func TestPluginImpl(t *testing.T) {
	p := NewPluginImpl()

	p.embyAuth.Addr.StringValue.Value = "https://emby.bangumi.ca"
	p.embyAuth.User.StringValue.Value = "labulakalia"
	p.embyAuth.Password.ObscureStringValue.Value = "Ww123456@"

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
	// 2. Test ListPluginMediaItemInfo - pick first menu that has items
	var listResp *plugin.ListPluginMediaInfoResponse
	for _, menu := range menus.GetPluginMenus() {
		m := menu.Menu
		resp, err := p.ListPluginMediaItemInfo(&plugin.ListPluginMediaInfoRequest{
			Menu:     m,
			Page:     1,
			PageSize: 5,
		})
		if err != nil {
			t.Fatalf("ListPluginMediaItemInfo for menu '%s' failed: %v", m.GetName(), err)
		}
		if len(resp.GetMediaInfos()) > 0 {
			listResp = resp
			t.Logf("found %d items in menu '%s'", len(resp.GetMediaInfos()), m.GetName())
			break
		}
	}
	if listResp == nil {
		t.Fatal("no menu returned any items")
	}

	// 3. Test GetPluginMediaItemDetail
	firstMedia := listResp.GetMediaInfos()[0]
	detail, err := p.GetPluginMediaItemDetail(&plugin.GetPluginMediaDetailRequest{
		MediaInfoId: firstMedia.GetMediaId(),
	})
	if err != nil {
		t.Fatalf("GetPluginMediaItemDetail for '%s' failed: %v", firstMedia.GetMediaId(), err)
	}
	t.Logf("detail for '%s' (type=%v): series=%v info=%v items=%d",
		firstMedia.GetName(), firstMedia.GetMediaType(),
		detail.GetMediaSeries() != nil,
		detail.GetMediaInfo() != nil,
		len(detail.GetMediaItems()))

	// 4. Search test
	searchResp, err := p.ListPluginMediaItemInfo(&plugin.ListPluginMediaInfoRequest{
		SearchName: "鱿鱼游戏",
		Page:       1,
		PageSize:   5,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	t.Logf("search results: %d", len(searchResp.GetMediaInfos()))

	// 5. GetFileResource
	fileResource, err := p.GetFileResource(&plugin.GetFileResourceRequest{
		FilePath:    firstMedia.GetMediaId(),
		IsMedia:     true,
		MediaPlayId: firstMedia.GetMediaId(),
	})
	if err != nil {
		t.Fatalf("GetFileResource failed: %v", err)
	}
	t.Logf("file resources: %d", len(fileResource.GetFileResourceData()))

	// 6. GetPluginFilterItems
	filterItems, err := p.GetPluginFilterItems(&plugin.PluginItem{Name: "test", Value: "test"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("filter items: %d filters", len(filterItems.GetFilters()))

	t.Log("ALL TESTS PASSED")
}
