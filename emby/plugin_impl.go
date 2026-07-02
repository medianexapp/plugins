package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/labulakalia/wazero_net/util"
	_ "github.com/labulakalia/wazero_net/wasi/http"
	"github.com/medianexapp/plugin_api/httpclient"
	"github.com/medianexapp/plugin_api/plugin"
	"golang.org/x/sync/errgroup"
)

type PluginImpl struct {
	embyAuth    *embyAuth
	userId      string
	serverId    string
	accessToken string

	hb *httpclient.Builder
}

type embyAuth struct {
	Addr     *plugin.Formdata_FormItem_StringValue
	User     *plugin.Formdata_FormItem_StringValue
	Password *plugin.Formdata_FormItem_ObscureStringValue
}

func NewPluginImpl() *PluginImpl {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	return &PluginImpl{
		embyAuth: &embyAuth{
			Addr:     plugin.String("http://127.0.0.1:8096"),
			User:     plugin.String(""),
			Password: plugin.ObscureString(""),
		},
	}
}

// PluginId implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "emby", nil
}

// PluginType implements IPlugin.
func (p *PluginImpl) PluginType() (plugin.PluginType, error) {
	return plugin.PluginType_PLUGIN_TYPE_MEDIA, nil
}

// GetAuth implements IPlugin.
func (p *PluginImpl) GetAuth() (*plugin.Auth, error) {
	slog.Info("GetAuth")
	authMethod := &plugin.AuthMethod{
		Method: &plugin.AuthMethod_Formdata{
			Formdata: &plugin.Formdata{
				FormItems: []*plugin.Formdata_FormItem{
					{
						Name:  "Addr",
						Value: p.embyAuth.Addr,
					},
					{
						Name:  "User",
						Value: p.embyAuth.User,
					},
					{
						Name:  "Password",
						Value: p.embyAuth.Password,
					},
				},
			},
		},
	}
	return &plugin.Auth{
		AuthMethods: []*plugin.AuthMethod{authMethod},
	}, nil
}

// CheckAuthMethod implements IPlugin.
func (p *PluginImpl) CheckAuthMethod(authMethod *plugin.AuthMethod) (*plugin.AuthData, error) {
	slog.Debug("CheckAuthMethod", "authMethod", authMethod)
	formData := authMethod.Method.(*plugin.AuthMethod_Formdata).Formdata
	authDataBytes, err := formData.MarshalVT()
	if err != nil {
		return nil, err
	}
	return &plugin.AuthData{
		AuthDataBytes: authDataBytes,
	}, nil
}

// CheckAuthData implements IPlugin.
func (p *PluginImpl) CheckAuthData(authDataBytes []byte) error {
	slog.Debug("CheckAuthData", "data", string(authDataBytes))
	formData := &plugin.Formdata{}
	err := formData.UnmarshalVT(authDataBytes)
	if err != nil {
		return err
	}
	p.embyAuth.Addr.StringValue.Value = formData.FormItems[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value
	p.embyAuth.User.StringValue.Value = formData.FormItems[1].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value
	p.embyAuth.Password.ObscureStringValue.Value = formData.FormItems[2].Value.(*plugin.Formdata_FormItem_ObscureStringValue).ObscureStringValue.Value
	// Authenticate with username and password to get access token
	user := p.embyAuth.User.StringValue.Value
	password := p.embyAuth.Password.ObscureStringValue.Value

	authResp := &AuthByNameResponse{}
	authUrl := "/emby/Users/AuthenticateByName"
	bodyData, _ := json.Marshal(map[string]string{
		"Username": user,
		"Pw":       password,
	})

	err = p.sendRequest(http.MethodPost, authUrl, strings.NewReader(string(bodyData)), authResp)
	if err != nil {
		slog.Error("authenticate failed", "err", err)
		return err
	}

	// Store user info and access token for subsequent requests
	p.userId = authResp.User.Id
	p.accessToken = authResp.AccessToken
	slog.Info("authenticated", "userId", p.userId, "userName", authResp.User.Name)

	// Verify connection by getting system info
	sysInfo := &SystemInfoResponse{}
	err = p.sendRequest(http.MethodGet, "/emby/System/Info", nil, sysInfo)
	if err != nil {
		slog.Error("get system info failed", "err", err)
		return err
	}
	slog.Info("connected to emby server", "serverName", sysInfo.ServerName, "version", sysInfo.Version)
	p.serverId = sysInfo.Id

	return nil
}

// PluginAuthId implements IPlugin.
func (p *PluginImpl) PluginAuthId() (string, error) {
	id := fmt.Sprintf("%s%s%s", p.embyAuth.Addr.StringValue.Value, p.embyAuth.User.StringValue.Value, p.embyAuth.Password.ObscureStringValue.Value)
	return fmt.Sprintf("%x", md5.Sum(util.StringToBytes(&id))), nil
}

// GetDirEntry implements IPlugin (not used for media plugins, returns empty).
func (p *PluginImpl) GetDirEntry(req *plugin.GetDirEntryRequest) (*plugin.DirEntry, error) {
	slog.Debug("GetDirEntry (not used for media plugin)")
	return &plugin.DirEntry{FileEntries: []*plugin.FileEntry{}}, nil
}

// GetFileResource implements IPlugin - returns stream URLs for an emby media item.
// Uses MediaPlayId if available, otherwise falls back to FilePath.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	resp := &plugin.FileResource{
		FileResourceData: []*plugin.FileResource_FileResourceData{},
	}
	slog.Debug("GetFileResource", "filePath", req.FilePath, "mediaPlayId", req.GetMediaPlayId())
	addr := strings.TrimRight(p.embyAuth.Addr.StringValue.Value, "/")
	params := url.Values{}
	params.Set("UserId", p.userId)
	params.Set("MaxStreamingBitrate", "200000000")
	params.Set("MediaSourceId", "")
	params.Set("reqformat", "json")
	uri := fmt.Sprintf("/emby/Items/%s/PlaybackInfo", req.MediaPlayId)
	reqData := strings.NewReader(getdeviceProfile(0))
	respData := &MediaSourcesData{}
	err := p.sendRequest(http.MethodPost, fmt.Sprintf("%s?%s", uri, params.Encode()), reqData, respData)
	if err != nil {
		return nil, err
	}
	if len(respData.MediaSources) == 0 {
		return nil, fmt.Errorf("no media sources found")
	}
	mediaResource := respData.MediaSources[0]

	var (
		show4K    bool
		show2K    bool
		show1080p bool
		show720p  bool
	)
	for _, stream := range mediaResource.MediaStreams {
		slog.Info("stream.Type", "type", stream.Type, "stream.Width", stream.Width)
		if stream.Type == "Video" {
			if stream.Width >= 3840 {
				show4K = true
			}
			if stream.Width >= 2560 {
				show2K = true
			}
			if stream.Width >= 1920 {
				show1080p = true
			}
			if stream.Width >= 1280 {
				show720p = true
			}
		}
	}
	params.Set("MediaSourceId", mediaResource.ID)
	params.Set("CurrentPlaySessionId", respData.PlaySessionID)
	if show4K {
		reqData := strings.NewReader(getdeviceProfile(3840))
		respData := &MediaSourcesData{}
		err := p.sendRequest(http.MethodPost, fmt.Sprintf("%s?%s", uri, params.Encode()), reqData, respData)
		if err != nil {
			return nil, err
		}
		if len(respData.MediaSources) > 0 {
			mediaSource := respData.MediaSources[0]
			resp.FileResourceData = append(resp.FileResourceData, &plugin.FileResource_FileResourceData{
				Url:        fmt.Sprintf("%s%s", addr, mediaSource.TranscodingUrl),
				Resolution: plugin.FileResource_UHD,
			})
		}
	}
	if show2K {
		reqData := strings.NewReader(getdeviceProfile(2560))
		respData := &MediaSourcesData{}
		err := p.sendRequest(http.MethodPost, fmt.Sprintf("%s?%s", uri, params.Encode()), reqData, respData)
		if err != nil {
			return nil, err
		}
		if len(respData.MediaSources) > 0 {
			mediaSource := respData.MediaSources[0]
			resp.FileResourceData = append(resp.FileResourceData, &plugin.FileResource_FileResourceData{
				Url:        fmt.Sprintf("%s%s", addr, mediaSource.TranscodingUrl),
				Resolution: plugin.FileResource_QHD,
			})
		}
	}
	if show1080p {
		reqData := strings.NewReader(getdeviceProfile(1920))
		respData := &MediaSourcesData{}
		err := p.sendRequest(http.MethodPost, fmt.Sprintf("%s?%s", uri, params.Encode()), reqData, respData)
		if err != nil {
			return nil, err
		}
		if len(respData.MediaSources) > 0 {
			mediaSource := respData.MediaSources[0]
			resp.FileResourceData = append(resp.FileResourceData, &plugin.FileResource_FileResourceData{
				Url:        fmt.Sprintf("%s%s", addr, mediaSource.TranscodingUrl),
				Resolution: plugin.FileResource_FHD,
			})
		}
	}
	if show720p {
		reqData := strings.NewReader(getdeviceProfile(1280))
		respData := &MediaSourcesData{}
		err := p.sendRequest(http.MethodPost, fmt.Sprintf("%s?%s", uri, params.Encode()), reqData, respData)
		if err != nil {
			return nil, err
		}
		if len(respData.MediaSources) > 0 {
			mediaSource := respData.MediaSources[0]
			resp.FileResourceData = append(resp.FileResourceData, &plugin.FileResource_FileResourceData{
				Url:        fmt.Sprintf("%s%s", addr, mediaSource.TranscodingUrl),
				Resolution: plugin.FileResource_HD,
			})
		}
	}
	return resp, nil
}

// ---------------------------------------------------------------------------
// Media Plugin Interface
// ---------------------------------------------------------------------------

// GetPluginMenus returns the emby library views as navigable menus.
// Each top-level menu corresponds to an emby View (library), and some
// may have sub-menus (e.g. genre filters).
func (p *PluginImpl) GetPluginMenus() (*plugin.PluginMenus, error) {
	slog.Info("GetPluginMenus")

	viewsResp := &ItemsResponse{}
	err := p.sendRequest(http.MethodGet, fmt.Sprintf("/emby/Users/%s/Views", p.userId), nil, viewsResp)
	if err != nil {
		slog.Error("get views failed", "err", err)
		return nil, err
	}

	menus := make([]*plugin.PluginMenu, 0, len(viewsResp.Items))
	for _, view := range viewsResp.Items {
		menuPre := ""
		switch view.CollectionType {
		case "tvshows":
			menuPre = "Series"
		case "movies":
			menuPre = "Movie"
		default:
			continue
		}

		menu := &plugin.PluginMenu{
			Menu: &plugin.PluginItem{
				Name:  view.Name,
				Value: fmt.Sprintf("%s_%s", menuPre, view.Id),
			},
		}

		menus = append(menus, menu)
	}

	return &plugin.PluginMenus{PluginMenus: menus}, nil
}

// GetPluginFilterItems returns filter items for a given sub-item.
// For Emby, it returns genre filters available under a parent view.
func (p *PluginImpl) GetPluginFilterItems(subItem *plugin.PluginItem) (*plugin.PluginFilterItems, error) {
	slog.Info("GetPluginFilterItems", "name", subItem.GetName(), "value", subItem.GetValue())
	id := strings.Split(subItem.GetValue(), "_")[1]
	filterItems := &plugin.PluginFilterItems{
		Filters: []*plugin.PluginFilterItems_Filter{
			{
				Name:  "分类",
				Items: make([]*plugin.PluginItem, 0, 10),
			},
			{
				Name:  "年份",
				Items: make([]*plugin.PluginItem, 0, 10),
			},
			{
				Name:  "分级",
				Items: make([]*plugin.PluginItem, 0, 10),
			},
			{
				Name:  "标签",
				Items: make([]*plugin.PluginItem, 0, 10),
			},
			{
				Name:  "工作室",
				Items: make([]*plugin.PluginItem, 0, 10),
			},
		},
	}
	g := errgroup.Group{}
	g.Go(func() error {
		genres, err := p.getGenres(id)
		if err != nil {
			return err
		}
		for _, item := range genres {
			filterItems.Filters[0].Items = append(filterItems.Filters[0].Items, &plugin.PluginItem{
				Name:  item.Name,
				Value: item.Id,
			})
		}
		return nil
	})
	g.Go(func() error {
		years, err := p.getYears(id)
		if err != nil {
			return err
		}
		for _, item := range years {
			if item.Id == "" {
				item.Id = item.Name
			}
			filterItems.Filters[1].Items = append(filterItems.Filters[1].Items, &plugin.PluginItem{
				Name:  item.Name,
				Value: item.Id,
			})
		}
		return nil
	})
	g.Go(func() error {
		officialRatings, err := p.getOfficialRatings(id)
		if err != nil {
			return err
		}
		for _, item := range officialRatings {
			if item.Id == "" {
				item.Id = item.Name
			}
			filterItems.Filters[2].Items = append(filterItems.Filters[2].Items, &plugin.PluginItem{
				Name:  item.Name,
				Value: item.Id,
			})
		}
		return nil
	})
	g.Go(func() error {
		tags, err := p.getTags(id)
		if err != nil {
			return err
		}
		for _, item := range tags {
			filterItems.Filters[3].Items = append(filterItems.Filters[3].Items, &plugin.PluginItem{
				Name:  item.Name,
				Value: item.Id,
			})
		}
		return nil
	})
	g.Go(func() error {
		studios, err := p.getStudios(id)
		if err != nil {
			return err
		}
		for _, item := range studios {
			filterItems.Filters[4].Items = append(filterItems.Filters[4].Items, &plugin.PluginItem{
				Name:  item.Name,
				Value: item.Id,
			})
		}
		return nil
	})
	g.Wait()
	return filterItems, nil
}

// ListPluginMedianfo lists media items (movies, series, episodes)
// from the emby server. Supports browsing by menu, filtering by genre,
// and text search.
func (p *PluginImpl) ListPluginMediaInfo(req *plugin.ListPluginMediaInfoRequest) (*plugin.ListPluginMediaInfoResponse, error) {
	slog.Debug("ListPluginMediaItemInfo",
		"req", req,
	)
	pageSize := min(req.GetPageSize(), 50)

	params := url.Values{}
	params.Set("StartIndex", fmt.Sprint((req.Page-1)*pageSize))
	params.Set("Limit", fmt.Sprint(pageSize))
	sp := strings.Split(req.Menu.Value, "_")
	if len(sp) != 2 {
		return nil, fmt.Errorf("invalid menu value: %s", req.Menu.Value)
	}
	includeType := sp[0]
	parentId := sp[1]

	params.Set("ParentId", parentId)
	params.Set("IncludeItemTypes", includeType)
	params.Set("Recursive", "true")
	if req.Filters != nil {
		for _, filter := range req.Filters.Filters {
			if len(filter.Items) == 0 {
				continue
			}
			values := []string{}
			for _, item := range filter.Items {
				values = append(values, item.Value)
			}
			if filter.Name == "分类" {
				params.Set("GenreIds", strings.Join(values, ","))
			}
			if filter.Name == "分级" {
				params.Set("OfficialRatings", strings.Join(values, ","))
			}
			if filter.Name == "年份" {
				params.Set("Years", strings.Join(values, ","))
			}
			if filter.Name == "标签" {
				params.Set("TagIds", strings.Join(values, ","))
			}
			if filter.Name == "工作室" {
				params.Set("StudioIds", strings.Join(values, ","))
			}
		}
	}

	params.Set("SortBy", "SortName")
	params.Set("SortOrder", "Ascending")
	params.Set("Fields", "BasicSyncInfo,PrimaryImageAspectRatio,ProductionYear,Status,EndDate")
	if req.SearchName != "" {
		params.Set("SearchTerm", req.SearchName)
	}
	apiUrl := fmt.Sprintf("/emby/Users/%s/Items?%s", p.userId, params.Encode())

	itemsResp := &ItemsResponse{
		Items: []*EmbyItem{},
	}
	err := p.sendRequest(http.MethodGet, apiUrl, nil, itemsResp)
	if err != nil {
		slog.Error("get items failed", "err", err)
		return nil, err
	}
	mediaInfos := make([]*plugin.PluginMedia, 0, len(itemsResp.Items))
	for _, item := range itemsResp.Items {
		// TODO convert plugin media
		mediaInfo := p.embyItemToPluginMedia(item, plugin.PluginMedia_MEDIA_INFO)
		mediaInfos = append(mediaInfos, mediaInfo)
	}
	resp := &plugin.ListPluginMediaInfoResponse{
		MediaInfos:        mediaInfos,
		SupportSearchName: true,
	}
	return resp, nil
}

func (p *PluginImpl) getItemById(itemId string) (*EmbyItem, error) {
	item := &EmbyItem{}
	params := url.Values{}
	itemUrl := fmt.Sprintf("/emby/Users/%s/Items/%s?%s", p.userId, itemId, params.Encode())
	err := p.sendRequest(http.MethodGet, itemUrl, nil, item)
	if err != nil {
		slog.Error("get item failed", "err", err)
		return nil, err
	}
	return item, nil
}

// GetPluginMediaDetail returns detailed information about a specific
// media item, including its seasons (for series), episodes (for seasons),
// and parent series information.
func (p *PluginImpl) GetPluginMediaDetail(req *plugin.GetPluginMediaDetailRequest) (*plugin.GetPluginMediaDetailResponse, error) {
	resp := &plugin.GetPluginMediaDetailResponse{
		MediaItems:         []*plugin.PluginMedia{},
		MediaInfoRelations: []*plugin.PluginMedia{},
	}
	// Fetch the item details
	mediaId := req.GetMediaInfoId()
	sp := strings.Split(mediaId, "_")
	var (
		itemId   string
		seasonId string
	)
	if len(sp) > 0 {
		itemId = sp[0]
	}
	if len(sp) > 1 {
		seasonId = sp[1]
	}
	item, err := p.getItemById(itemId)
	if err != nil {
		slog.Error("get item failed", "err", err)
		return nil, err
	}

	switch item.Type {
	case "Series":
		resp.MediaSeries = p.embyItemToPluginMedia(item, plugin.PluginMedia_MEDIA_SERIES)
		seasonParams := url.Values{}
		seasonParams.Set("UserId", p.userId)
		seasonParams.Set("Fields", "BasicSyncInfo,CanDelete,PrimaryImageAspectRatio,Overview")
		seaUri := fmt.Sprintf("/emby/Shows/%s/Seasons?%s", item.Id, seasonParams.Encode())
		seasonsResp := &ItemsResponse{
			Items: []*EmbyItem{},
		}
		err = p.sendRequest(http.MethodGet, seaUri, nil, seasonsResp)
		if err != nil {
			slog.Error("get season episodes failed", "err", err)
			return nil, err
		}

		if seasonId == "" {
			seasonId = seasonsResp.Items[0].Id
		}
		for _, seasonItem := range seasonsResp.Items {
			if seasonItem.Id == seasonId {
				resp.MediaInfo = p.embyItemToPluginMedia(seasonItem, plugin.PluginMedia_MEDIA_INFO)
			} else {
				resp.MediaInfoRelations = append(resp.MediaInfoRelations, p.embyItemToPluginMedia(seasonItem, plugin.PluginMedia_MEDIA_INFO))
			}
		}
		resp.MediaInfo.LogoUrl = resp.MediaSeries.LogoUrl
		episodesparams := url.Values{}
		episodesparams.Set("SeasonId", seasonId)
		episodesparams.Set("UserId", p.userId)
		episodesparams.Set("Fields", "Overview,PrimaryImageAspectRatio,PremiereDate,ProductionYear,SyncStatus")
		uri := fmt.Sprintf("/emby/Shows/%s/Episodes?%s", item.ParentId, episodesparams.Encode())
		episodesResp := &ItemsResponse{
			Items: []*EmbyItem{},
		}
		err = p.sendRequest(http.MethodGet, uri, nil, episodesResp)
		if err != nil {
			slog.Error("get season episodes failed", "err", err)
			return nil, err
		}
		for _, item := range episodesResp.Items {
			resp.MediaItems = append(resp.MediaItems, p.embyItemToPluginMedia(item, plugin.PluginMedia_MEDIA_PLAY_ITEM))
		}
	case "Movie":
		// todo get item
		resp.MediaInfo = p.embyItemToPluginMedia(item, plugin.PluginMedia_MEDIA_INFO)
		mediaPlayItem := p.embyItemToPluginMedia(item, plugin.PluginMedia_MEDIA_PLAY_ITEM)
		mediaPlayItem.Desc = ""
		mediaPlayItem.Name = ""
		resp.MediaItems = []*plugin.PluginMedia{mediaPlayItem}
	default:
	}
	return resp, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (p *PluginImpl) sendRequest(method string, uri string, reqBody any, respBody any) error {
	if p.hb == nil {
		addr := strings.TrimRight(p.embyAuth.Addr.StringValue.Value, "/")
		p.hb = httpclient.NewBuilder().SetBaseURL(addr).
			SetHeader("Content-Type", "application/json").
			SetHeader("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36").
			SetHeader("X-Emby-Authorization",
				`MediaBrowser Client="Medianex/Plugin", Device="Plugin", DeviceId="plugin", Version="1.0.0"`)
	}

	sendReq := p.hb.SetURI(uri).SetMethod(method).Debug()
	if reqBody != nil {
		sendReq = sendReq.SetBody(reqBody)
	}
	if p.accessToken != "" {
		sendReq = sendReq.SetHeader("X-Emby-Token", p.accessToken)
	}

	resp, err := sendReq.RawResponse()
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	json.Unmarshal(data, respBody)
	return nil
}

// getGenres fetches available genres for a parent view.
func (p *PluginImpl) getGenres(parentId string) ([]*FilterItem, error) {
	genresUrl := fmt.Sprintf("/emby/Genres?ParentId=%s&UserId=%s", parentId, p.userId)
	resp := &FilterItemsResponse{}
	err := p.sendRequest(http.MethodGet, genresUrl, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (p *PluginImpl) getYears(parentId string) ([]*FilterItem, error) {
	genresUrl := fmt.Sprintf("/emby/Years?ParentId=%s&UserId=%s", parentId, p.userId)
	resp := &FilterItemsResponse{}
	err := p.sendRequest(http.MethodGet, genresUrl, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (p *PluginImpl) getOfficialRatings(parentId string) ([]*FilterItem, error) {
	genresUrl := fmt.Sprintf("/emby/OfficialRatings?ParentId=%s&UserId=%s", parentId, p.userId)
	resp := &FilterItemsResponse{}
	err := p.sendRequest(http.MethodGet, genresUrl, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (p *PluginImpl) getStudios(parentId string) ([]*FilterItem, error) {
	genresUrl := fmt.Sprintf("/emby/Studios?ParentId=%s&UserId=%s", parentId, p.userId)
	resp := &FilterItemsResponse{}
	err := p.sendRequest(http.MethodGet, genresUrl, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (p *PluginImpl) getTags(parentId string) ([]*FilterItem, error) {
	genresUrl := fmt.Sprintf("/emby/Tags?ParentId=%s&UserId=%s", parentId, p.userId)
	resp := &FilterItemsResponse{}
	err := p.sendRequest(http.MethodGet, genresUrl, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// embyItemToPluginMedia converts an Emby item to a PluginMedia struct.
func (p *PluginImpl) embyItemToPluginMedia(item *EmbyItem, pluginMediaType plugin.PluginMedia_MediaType) *plugin.PluginMedia {
	addr := strings.TrimRight(p.embyAuth.Addr.StringValue.Value, "/")
	media := &plugin.PluginMedia{
		PluginMediaId: item.Id,
		Name:          item.Name,
		Year:          uint64(item.ProductionYear),
		Genres:        []string{},
		Desc:          item.Overview,
		OriginalName:  item.OriginalTitle,
		MediaType:     pluginMediaType,
		Index:         uint64(item.IndexNumber),
	}

	if len(item.GenreItems) > 0 {
		media.Genres = []string{}
		for _, genre := range item.GenreItems {
			media.Genres = append(media.Genres, genre.Name)
		}
	}

	if len(item.People) > 0 {
		media.Credit = []*plugin.PluginMedia_Credit{}
		for _, person := range item.People {
			credit := &plugin.PluginMedia_Credit{
				Name:       person.Name,
				Character:  person.Role,
				ProfileUrl: fmt.Sprintf("%s/emby/Items/%s/Images/Primary?maxHeight=300&maxWidth=200&tag=%s&quality=90", addr, item.Id, item.ImageTags["PrimaryImageTag"]),
			}
			switch person.Role {
			case "Actor":
				credit.CreditType = plugin.PluginMedia_CreditActor
			case "Director":
				credit.CreditType = plugin.PluginMedia_CreditCastDirecting
			}
			media.Credit = append(media.Credit, credit)
		}
	}

	if item.RunTimeTicks > 0 {
		duration := item.RunTimeTicks / 10000000
		media.Duration = uint64(duration)
	}

	switch item.Type {
	case "Series":
		if len(item.BackdropImageTags) > 0 {
			media.BackdropUrl = fmt.Sprintf("%s/emby/Items/%s/Images/Backdrop/0?tag=%s&maxWidth=1920&quality=70", addr, item.Id, item.BackdropImageTags[0])
		}
		if item.ImageTags["Primary"] != "" {
			media.PosterUrl = fmt.Sprintf("%s/emby/Items/%s/Images/Primary?maxHeight=400&tag=%s&maxWidth=300&quality=90", addr, item.Id, item.ImageTags["Primary"])
		}
		if item.ImageTags["Logo"] != "" {
			media.LogoUrl = fmt.Sprintf("%s/emby/Items/%s/Images/Logo?tag=%s&quality=90", addr, item.Id, item.ImageTags["Logo"])
		}

	case "Season":
		if item.ImageTags["Primary"] != "" {
			media.PosterUrl = fmt.Sprintf("%s/emby/Items/%s/Images/Primary?maxHeight=400&tag=%s&maxWidth=300&quality=90", addr, item.Id, item.ImageTags["Primary"])
		} else if item.SeriesPrimaryImageTag != "" {
			media.PosterUrl = fmt.Sprintf("%s/emby/Items/%s/Images/Primary?maxHeight=400&tag=%s&maxWidth=300&quality=90", addr, item.SeriesId, item.SeriesPrimaryImageTag)
		}
		// logo url
		media.ParentPluginMediaId = item.SeriesId
		media.Name = fmt.Sprintf("%s %s", item.SeriesName, item.Name)
		media.PluginMediaId = fmt.Sprintf("%s_%s", item.SeriesId, item.Id)
	case "Episode":
		if item.ImageTags["Primary"] != "" {
			media.StillUrl = fmt.Sprintf("%s/emby/Items/%s/Images/Primary?maxHeight=400&tag=%s&maxWidth=300&quality=90", addr, item.Id, item.ImageTags["Primary"])
		}
		item.ParentId = item.SeriesId

	case "Movie":
		if item.ImageTags["Primary"] != "" {
			media.PosterUrl = fmt.Sprintf("%s/emby/Items/%s/Images/Primary?maxHeight=400&tag=%s&maxWidth=300&quality=90", addr, item.Id, item.ImageTags["Primary"])
		}
		if item.ImageTags["Logo"] != "" {
			media.LogoUrl = fmt.Sprintf("%s/emby/Items/%s/Images/Logo?maxHeight=400&tag=%s&quality=90", addr, item.Id, item.ImageTags["Logo"])
		}
		if len(item.BackdropImageTags) > 0 {
			media.BackdropUrl = fmt.Sprintf("%s/emby/Items/%s/Images/Backdrop/0?maxHeight=400&tag=%s&maxWidth=1920&quality=70", addr, item.Id, item.BackdropImageTags[0])
		}

	default:
		return nil
	}
	return media
}
