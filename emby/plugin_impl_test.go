package main

import (
	"log/slog"
	"os"
	"testing"

	"github.com/medianexapp/plugin_api/plugin"
)

// emby
// 电影没有自动集合  Type : "Movie" 刚好对应play info
// 电视剧默认展示 Type: "Series" 对应 play seriers ，转换为play info，在relation里展示emby的所有季
//

var (
	p *PluginImpl
)

func TestMain(m *testing.M) {
	p = NewPluginImpl()

	p.embyAuth.Addr.StringValue.Value = "https://emby.bangumi.ca"
	p.embyAuth.User.StringValue.Value = "labulakalia"
	p.embyAuth.Password.ObscureStringValue.Value = "Ww123456@@@"

	auth, err := p.GetAuth()
	if err != nil {
		slog.Error("get auth failed", "err", err)
		os.Exit(1)
	}

	authData, err := p.CheckAuthMethod(&plugin.AuthMethod{
		Method: auth.AuthMethods[0].Method,
	})
	if err != nil {
		slog.Error("check auth failed", "err", err)
		os.Exit(1)
	}
	err = p.CheckAuthData(authData.AuthDataBytes)
	if err != nil {
		slog.Error("check auth data failed", "err", err)
		os.Exit(1)
	}
	m.Run()

}

func TestPluginImpl(t *testing.T) {

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
	resp, err := p.GetPluginFilterItems(menus.GetPluginMenus()[0].Menu)
	if err != nil {
		t.Fatalf("GetPluginFilterItems for menu '%s' failed: %v", menus.GetPluginMenus()[0].Menu, err)
	}
	for _, filters := range resp.Filters {
		t.Logf("filter %+v\n", filters)
	}

	t.Log("Menu", menus.GetPluginMenus()[0].Menu)
	mediaInfosResp2, err := p.ListPluginMediaItemInfo(&plugin.ListPluginMediaInfoRequest{
		Menu:     menus.GetPluginMenus()[0].Menu,
		PageSize: 10,
		Page:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range mediaInfosResp2.MediaInfos {
		if info.Name == "" || info.Year == 0 || info.PosterUrl == "" {
			t.Fatalf("media info is empty %+v\n", info)
		}
		t.Logf("media info %+v\n", info)
	}

	mediaInfoDetailResp1, err := p.GetPluginMediaItemDetail(&plugin.GetPluginMediaDetailRequest{
		MediaInfoId: mediaInfosResp2.MediaInfos[1].PluginMediaId,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("mediaInfoDetailResp MediaSeries %+v\n", mediaInfoDetailResp1.MediaSeries)
	t.Logf("mediaInfoDetailResp MediaInfo %+v\n", mediaInfoDetailResp1.MediaInfo)
	t.Logf("mediaInfoDetailResp MediaInfoRelations %+v\n", mediaInfoDetailResp1.MediaInfoRelations)
	t.Logf("mediaInfoDetailResp mediaInfoDetailResp %+v\n", len(mediaInfoDetailResp1.MediaItems))
	if len(mediaInfoDetailResp1.MediaInfoRelations) > 0 {
		mediaInfoDetailResp2, err := p.GetPluginMediaItemDetail(&plugin.GetPluginMediaDetailRequest{
			MediaInfoId: mediaInfoDetailResp1.MediaInfoRelations[0].PluginMediaId,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("mediaInfoDetailResp MediaSeries %+v\n", mediaInfoDetailResp2.MediaSeries)
		t.Logf("mediaInfoDetailResp MediaInfo %+v\n", mediaInfoDetailResp2.MediaInfo)
		t.Logf("mediaInfoDetailResp MediaInfoRelations %+v\n", len(mediaInfoDetailResp2.MediaInfoRelations))
		t.Logf("mediaInfoDetailResp mediaInfoDetailResp %+v\n", len(mediaInfoDetailResp2.MediaItems))

	}

	// fileResource, err := p.GetFileResource(&plugin.GetFileResourceRequest{
	// 	MediaPlayId: mediaInfoDetailResp2.MediaItems[0].MediaId,
	// })
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// // t.Logf("fileResource %+v\n", )
	// for _, item := range fileResource.FileResourceData {
	// 	t.Log("url", item.Url)
	// }
	//movie
	mediaMovieResp2, err := p.ListPluginMediaItemInfo(&plugin.ListPluginMediaInfoRequest{
		Menu:     menus.GetPluginMenus()[5].Menu,
		PageSize: 10,
		Page:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	mediaMovieDetailResp1, err := p.GetPluginMediaItemDetail(&plugin.GetPluginMediaDetailRequest{
		MediaInfoId: mediaMovieResp2.MediaInfos[0].PluginMediaId,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("mediaInfoDetailResp MediaSeries %+v\n", mediaMovieDetailResp1.MediaSeries)
	t.Logf("mediaInfoDetailResp MediaInfo %+v\n", mediaMovieDetailResp1.MediaInfo)
	t.Logf("mediaInfoDetailResp MediaInfoRelations %+v\n", mediaMovieDetailResp1.MediaInfoRelations)
	t.Logf("mediaInfoDetailResp mediaInfoDetailResp %+v\n", len(mediaMovieDetailResp1.MediaItems))

}

func TestGetMediaDetail(t *testing.T) {
	resp, err := p.GetPluginMediaItemDetail(&plugin.GetPluginMediaDetailRequest{
		MediaInfoId: "95195",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("media series %+v\n", resp.MediaSeries)
	t.Logf("media info %+v\n", resp.MediaInfo)
	t.Logf("media info %+v\n", resp.MediaInfoRelations)
}
