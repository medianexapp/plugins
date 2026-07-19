package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"log/slog"
	"plugins/util"
	"strings"

	"github.com/medianexapp/plugin_api"
	"github.com/medianexapp/plugin_api/httpclient"

	"github.com/medianexapp/plugin_api/plugin"

	_ "github.com/labulakalia/wazero_net/wasi/http" // if you need http import this
	// _ "github.com/labulakalia/wazero_net/wasi/net"  // if you need net.Conn import this
	"github.com/medianexapp/m3u"
)

type PluginImpl struct {
	plugin_api.PluginExport
	url      string
	playList m3u.Playlist
}

func NewPluginImpl() *PluginImpl {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	return &PluginImpl{}
}

// PluginType implements IPlugin.
func (p *PluginImpl) PluginType() (plugin.PluginType, error) {

	return plugin.PluginType_PLUGIN_TYPE_MEDIA, nil

}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "iptv", nil
}

// GetAuth return how to auth
// 1.FormData input data
// 2.Callback use url callback auth,like oauth
// 3.Scanqrcode,return qrcode image to auth
func (p *PluginImpl) GetAuth() (*plugin.Auth, error) {
	slog.Info("GetAuth")
	auth := &plugin.Auth{
		AuthMethods: []*plugin.AuthMethod{
			{
				Method: &plugin.AuthMethod_Formdata{
					Formdata: &plugin.Formdata{
						FormItems: []*plugin.Formdata_FormItem{
							{
								Name:  "URL",
								Value: plugin.String(""),
							},
						},
					},
				},
			},
		},
	}
	return auth, nil
}

// CheckAuthMethod check auth is finished and return authDataBytes and authData's expired time
// if authmethod's type is *plugin.AuthMethod_Refresh,you need to refresh token
// assert authMethod.Method's type to check auth is finished,return auth data and expired time if authed
func (p *PluginImpl) CheckAuthMethod(authMethod *plugin.AuthMethod) (*plugin.AuthData, error) {
	value := authMethod.Method.(*plugin.AuthMethod_Formdata).Formdata.FormItems[0].Value
	url := value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value
	p.url = url

	return &plugin.AuthData{
		AuthDataBytes: []byte(url),
	}, nil
}

// CheckAuthData use authDataBytes to uath
// you must store auth data to *PluginImpl
func (p *PluginImpl) CheckAuthData(authDataBytes []byte) error {
	url := p.url
	resp, err := httpclient.NewBuilder().SetUserAgent(util.GetUserAgent()).Get(url).RawResponse()
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s", data)
	}
	playList, err := m3u.ParseFromReader(bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	p.playList = playList
	return nil
}

// PluginAuthId implements IPlugin.
// plugin auth id,you can generate id by md5 or sha
func (p *PluginImpl) PluginAuthId() (string, error) {
	return fmt.Sprintf("%x", md5.Sum([]byte(p.url))), nil
}

// plugin type PLUGIN_TYPE_FILE_SYSTEM
// GetFileResource implements IPlugin.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	slog.Debug("GetFileResource", "req", req)
	for _, track := range p.playList.Tracks {
		if req.MediaPlayId == track.TagData.TvgId || req.MediaPlayId == track.Name {
			pluginMediaId := track.TagData.TvgId
			if pluginMediaId == "" {
				pluginMediaId = track.Name
			}
			resp := &plugin.FileResource{
				FileResourceData: []*plugin.FileResource_FileResourceData{},
			}
			fileResource := &plugin.FileResource_FileResourceData{
				Url:          track.URI,
				Title:        track.Name,
				Resolution:   plugin.FileResource_Original,
				ResourceType: plugin.FileResource_Video,
			}
			resp.FileResourceData = append(resp.FileResourceData, fileResource)

			return resp, nil
		}
	}
	return nil, fmt.Errorf("not found %s", req.MediaPlayId)
}

// plugin type PLUGIN_TYPE_MEDIA
// support filter by multi column
func (p *PluginImpl) GetPluginMenus() (*plugin.PluginMenus, error) {
	slog.Debug("GetPluginMenus")
	groupTitles := []string{}
	groupTitleMap := map[string]bool{}
	for _, track := range p.playList.Tracks {
		if !groupTitleMap[track.TagData.GroupTitle] {
			groupTitles = append(groupTitles, track.TagData.GroupTitle)
			groupTitleMap[track.TagData.GroupTitle] = true
		} else {
			continue
		}
	}
	pluginMeus := &plugin.PluginMenus{
		PluginMenus: []*plugin.PluginMenu{},
	}
	for _, groupTitle := range groupTitles {
		if groupTitle == "" {
			groupTitle = "M3U8"
		}
		pluginMeus.PluginMenus = append(pluginMeus.PluginMenus, &plugin.PluginMenu{
			Menu: &plugin.PluginItem{
				Name:  groupTitle,
				Value: groupTitle,
			},
		})
	}
	return pluginMeus, nil
}

// get page filter items
func (p *PluginImpl) GetPluginFilterItems(req *plugin.PluginItem) (*plugin.PluginFilterItems, error) {
	return &plugin.PluginFilterItems{Filters: []*plugin.PluginFilterItems_Filter{}}, nil
}

// list media info
func (p *PluginImpl) ListPluginMediaInfo(req *plugin.ListPluginMediaInfoRequest) (*plugin.ListPluginMediaInfoResponse, error) {
	slog.Debug("ListPluginMediaInfo", "req", req)
	groupTitleId := req.Menu.Value
	resp := &plugin.ListPluginMediaInfoResponse{
		MediaInfos: []*plugin.PluginMedia{},
	}
	for _, track := range p.playList.Tracks {
		if track.TagData.GroupTitle == groupTitleId {
			if req.SearchName != "" && !strings.Contains(track.TagData.GroupTitle, req.SearchName) {
				continue
			}
			pluginMediaId := track.TagData.TvgId
			if pluginMediaId == "" {
				pluginMediaId = track.Name
			}
			resp.MediaInfos = append(resp.MediaInfos, &plugin.PluginMedia{
				Name:          track.Name,
				PluginMediaId: pluginMediaId,
				PosterUrl:     track.TagData.TvgLogo,
				MediaType:     plugin.PluginMedia_MEDIA_INFO,
			})
		}
	}
	return resp, nil
}

// get media info by id
func (p *PluginImpl) GetPluginMediaDetail(req *plugin.GetPluginMediaDetailRequest) (*plugin.GetPluginMediaDetailResponse, error) {
	slog.Debug("GetPluginMediaDetail", "req", req)
	for _, track := range p.playList.Tracks {
		if req.MediaInfoId == track.TagData.TvgId || req.MediaInfoId == track.Name {
			pluginMediaId := track.TagData.TvgId
			if pluginMediaId == "" {
				pluginMediaId = track.Name
			}
			resp := &plugin.GetPluginMediaDetailResponse{}
			pluginMedia := &plugin.PluginMedia{
				Name:          track.Name,
				PluginMediaId: pluginMediaId,
				PosterUrl:     track.TagData.TvgLogo,
				MediaType:     plugin.PluginMedia_MEDIA_INFO,
			}
			resp.MediaInfo = pluginMedia
			resp.MediaItems = []*plugin.PluginMedia{&plugin.PluginMedia{
				Name:          track.Name,
				PluginMediaId: pluginMediaId,
				PosterUrl:     track.TagData.TvgLogo,
				MediaType:     plugin.PluginMedia_MEDIA_PLAY_ITEM,
			}}
			return resp, nil
		}
	}
	return nil, fmt.Errorf("not found %s", req.MediaInfoId)
}
