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
	"github.com/medianexapp/plugin_api/plugin"
	"golang.org/x/sync/errgroup"
)

type PluginImpl struct {
	embyAuth    *embyAuth
	userId      string
	serverId    string
	accessToken string
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
	slog.Debug("CheckAuthData")
	formData := &plugin.Formdata{}
	err := formData.UnmarshalVT(authDataBytes)
	if err != nil {
		return err
	}
	p.embyAuth.Addr.StringValue.Value = formData.FormItems[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value
	p.embyAuth.User.StringValue.Value = formData.FormItems[1].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value
	p.embyAuth.Password.ObscureStringValue.Value = formData.FormItems[2].Value.(*plugin.Formdata_FormItem_ObscureStringValue).ObscureStringValue.Value

	addr := strings.TrimRight(p.embyAuth.Addr.StringValue.Value, "/")
	slog.Debug("emby connect", "addr", addr)

	// Authenticate with username and password to get access token
	user := p.embyAuth.User.StringValue.Value
	password := p.embyAuth.Password.ObscureStringValue.Value

	authResp := &AuthByNameResponse{}
	authUrl := "/emby/Users/AuthenticateByName"
	bodyData, _ := json.Marshal(map[string]string{
		"Username": user,
		"Pw":       password,
	})
	err = p.sendAuthPost(http.MethodPost, authUrl, bodyData, authResp)
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
	err = p.sendGet("/emby/System/Info", nil, sysInfo)
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
	slog.Debug("GetFileResource", "filePath", req.FilePath, "mediaPlayId", req.GetMediaPlayId())

	addr := strings.TrimRight(p.embyAuth.Addr.StringValue.Value, "/")

	mediaPlayId := req.GetMediaPlayId()
	if mediaPlayId == "" {
		mediaPlayId = req.FilePath
	}

	fileResource := &plugin.FileResource{
		FileResourceData: []*plugin.FileResource_FileResourceData{},
	}

	// Direct stream URL (auth handled by user context via userId)
	streamUrl := fmt.Sprintf("%s/emby/Videos/%s/stream?UserId=%s&Static=true",
		addr, mediaPlayId, p.userId)
	fileResource.FileResourceData = append(fileResource.FileResourceData, &plugin.FileResource_FileResourceData{
		Url:          streamUrl,
		Resolution:   plugin.FileResource_Original,
		ResourceType: plugin.FileResource_Video,
	})

	// HLS stream URL (adaptive streaming)
	hlsUrl := fmt.Sprintf("%s/emby/Videos/%s/master.m3u8?UserId=%s",
		addr, mediaPlayId, p.userId)
	fileResource.FileResourceData = append(fileResource.FileResourceData, &plugin.FileResource_FileResourceData{
		Url:          hlsUrl,
		Resolution:   plugin.FileResource_Original,
		ResourceType: plugin.FileResource_Video,
	})

	return fileResource, nil
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
	err := p.sendGet(fmt.Sprintf("/emby/Users/%s/Views", p.userId), nil, viewsResp)
	if err != nil {
		slog.Error("get views failed", "err", err)
		return nil, err
	}

	menus := make([]*plugin.PluginMenu, 0, len(viewsResp.Items))
	for _, view := range viewsResp.Items {
		if !view.IsFolder {
			continue
		}

		menu := &plugin.PluginMenu{
			Menu: &plugin.PluginItem{
				Name:  view.Name,
				Value: view.Id,
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
	filterItems := &plugin.PluginFilterItems{
		Filters: []*plugin.PluginFilterItems_Filter{
			{
				Name:  "分类",
				Items: make([]*plugin.PluginItem, 0, 20),
			},
			{
				Name:  "年份",
				Items: make([]*plugin.PluginItem, 0, 20),
			},
			{
				Name:  "分级",
				Items: make([]*plugin.PluginItem, 0, 20),
			},
			{
				Name:  "标签",
				Items: make([]*plugin.PluginItem, 0, 20),
			},
			{
				Name:  "工作室",
				Items: make([]*plugin.PluginItem, 0, 20),
			},
		},
	}
	g := errgroup.Group{}
	g.Go(func() error {
		genres, err := p.getGenres(subItem.Value)
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
		years, err := p.getYears(subItem.Value)
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
		officialRatings, err := p.getOfficialRatings(subItem.Value)
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
		tags, err := p.getTags(subItem.Value)
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
		studios, err := p.getStudios(subItem.Value)
		if err != nil {
			return err
		}
		for _, item := range studios {
			filterItems.Filters[3].Items = append(filterItems.Filters[4].Items, &plugin.PluginItem{
				Name:  item.Name,
				Value: item.Id,
			})
		}
		return nil
	})
	g.Wait()
	return filterItems, nil
}

// ListPluginMediaItemInfo lists media items (movies, series, episodes)
// from the emby server. Supports browsing by menu, filtering by genre,
// and text search.
func (p *PluginImpl) ListPluginMediaItemInfo(req *plugin.ListPluginMediaInfoRequest) (*plugin.ListPluginMediaInfoResponse, error) {
	slog.Info("ListPluginMediaItemInfo",
		"req", req,
	)

	pageSize := min(req.GetPageSize(), 50)

	params := url.Values{}
	params.Set("StartIndex", fmt.Sprint((req.Page-1)*pageSize))
	params.Set("Limit", fmt.Sprint(pageSize))
	params.Set("ParentId", req.Menu.Value)
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
	params.Set("SortBy", "SortName")
	params.Set("SortOrder", "Ascending")
	// params.Set("Fields", "Overview,Genres,MediaSources,People,PrimaryImageAspectRatio")
	apiUrl := fmt.Sprintf("/emby/Users/%s/Items?%s", p.userId, params.Encode())

	itemsResp := &ItemsResponse{
		Items: []*EmbyItem{},
	}
	err := p.sendGet(apiUrl, nil, itemsResp)
	if err != nil {
		slog.Error("get items failed", "err", err)
		return nil, err
	}

	mediaInfos := make([]*plugin.PluginMedia, 0, len(itemsResp.Items))
	for _, item := range itemsResp.Items {
		// TODO convert plugin media
		mediaInfo := p.embyItemToPluginMedia(item)
		mediaInfos = append(mediaInfos, mediaInfo)
	}
	resp := &plugin.ListPluginMediaInfoResponse{
		MediaInfos: mediaInfos,
	}
	return resp, nil
}

// GetPluginMediaItemDetail returns detailed information about a specific
// media item, including its seasons (for series), episodes (for seasons),
// and parent series information.
func (p *PluginImpl) GetPluginMediaItemDetail(req *plugin.GetPluginMediaDetailRequest) (*plugin.GetPluginMediaDetailResponse, error) {
	mediaInfoId := req.GetMediaInfoId()
	slog.Info("GetPluginMediaItemDetail", "mediaInfoId", mediaInfoId)

	// Fetch the item details
	item := &EmbyItem{}
	itemUrl := fmt.Sprintf("/emby/Users/%s/Items/%s?Fields=Overview,Genres,MediaSources,People,PrimaryImageAspectRatio",
		p.userId, mediaInfoId)
	err := p.sendGet(itemUrl, nil, item)
	if err != nil {
		slog.Error("get item failed", "err", err)
		return nil, err
	}

	mediaInfo := p.embyItemToPluginMedia(item)
	resp := &plugin.GetPluginMediaDetailResponse{}

	switch item.Type {
	case "Series":
		resp.MediaSeries = mediaInfo
		// Get seasons as media items
		seasonsResp := &ItemsResponse{}
		seasonsUrl := fmt.Sprintf("/emby/Users/%s/Items?ParentId=%s&IncludeItemTypes=Season&Fields=Overview",
			p.userId, mediaInfoId)
		err = p.sendGet(seasonsUrl, nil, seasonsResp)
		if err == nil {
			resp.MediaItems = make([]*plugin.PluginMedia, 0, len(seasonsResp.Items))
			for _, season := range seasonsResp.Items {
				resp.MediaItems = append(resp.MediaItems, p.embyItemToPluginMedia(season))
			}
		}

	case "Season":
		resp.MediaInfo = mediaInfo
		// Get episodes
		episodesResp := &ItemsResponse{}
		episodesUrl := fmt.Sprintf("/emby/Users/%s/Items?ParentId=%s&IncludeItemTypes=Episode&Fields=Overview,MediaSources",
			p.userId, mediaInfoId)
		err = p.sendGet(episodesUrl, nil, episodesResp)
		if err == nil {
			resp.MediaItems = make([]*plugin.PluginMedia, 0, len(episodesResp.Items))
			for _, episode := range episodesResp.Items {
				resp.MediaItems = append(resp.MediaItems, p.embyItemToPluginMedia(episode))
			}
		}
		// Fetch the parent series info
		if item.ParentId != "" {
			seriesItem := &EmbyItem{}
			seriesUrl := fmt.Sprintf("/emby/Users/%s/Items/%s?Fields=Overview,Genres,PrimaryImageAspectRatio",
				p.userId, item.ParentId)
			err = p.sendGet(seriesUrl, nil, seriesItem)
			if err == nil {
				resp.MediaSeries = p.embyItemToPluginMedia(seriesItem)
			}
		}

	default:
		// Movie, Episode, or other - treat as media info
		resp.MediaInfo = mediaInfo
	}

	return resp, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// sendGet sends a GET request to the emby server.
func (p *PluginImpl) sendGet(uri string, reqBody, respBody interface{}) error {
	addr := strings.TrimRight(p.embyAuth.Addr.StringValue.Value, "/")

	fullUrl := fmt.Sprintf("%s%s", addr, uri)
	slog.Debug("sending GET request", "url", fullUrl)

	httpReq, err := http.NewRequest(http.MethodGet, fullUrl, nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("X-Emby-Token", p.accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	return p.doRequest(httpReq, uri, respBody)
}

// sendAuthPost sends a POST request with Emby auth headers (for /Users/AuthenticateByName).
func (p *PluginImpl) sendAuthPost(method string, uri string, body []byte, respBody interface{}) error {
	addr := strings.TrimRight(p.embyAuth.Addr.StringValue.Value, "/")

	fullUrl := fmt.Sprintf("%s%s", addr, uri)
	slog.Debug("sending auth POST request", "url", fullUrl)

	httpReq, err := http.NewRequest(method, fullUrl, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// Emby auth requires this header format
	httpReq.Header.Set("X-Emby-Authorization",
		`MediaBrowser Client="Plugin", Device="Plugin", DeviceId="plugin", Version="1.0.0"`)

	return p.doRequest(httpReq, uri, respBody)
}

// doRequest executes an HTTP request and unmarshals the response.
func (p *PluginImpl) doRequest(httpReq *http.Request, uri string, respBody interface{}) error {
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	bodyData, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}
	slog.Info("doRequest finished", "uri", uri, "statusCode", httpResp.StatusCode, "body", string(bodyData))
	if httpResp.StatusCode >= 400 {
		slog.Error("emby api error", "uri", uri, "statusCode", httpResp.StatusCode, "body", string(bodyData))
		return fmt.Errorf("emby api error: status=%d, body=%s", httpResp.StatusCode, string(bodyData))
	}

	if respBody != nil {
		err = json.Unmarshal(bodyData, respBody)
		if err != nil {
			return fmt.Errorf("unmarshal failed for %s: %w, body=%s", uri, err, string(bodyData))
		}
	}

	return nil
}

// getGenres fetches available genres for a parent view.
func (p *PluginImpl) getGenres(parentId string) ([]*FilterItem, error) {
	genresUrl := fmt.Sprintf("/emby/Genres?ParentId=%s&UserId=%s", parentId, p.userId)
	resp := &FilterItemsResponse{}
	err := p.sendGet(genresUrl, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (p *PluginImpl) getYears(parentId string) ([]*FilterItem, error) {
	genresUrl := fmt.Sprintf("/emby/Years?ParentId=%s&UserId=%s", parentId, p.userId)
	resp := &FilterItemsResponse{}
	err := p.sendGet(genresUrl, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (p *PluginImpl) getOfficialRatings(parentId string) ([]*FilterItem, error) {
	genresUrl := fmt.Sprintf("/emby/OfficialRatings?ParentId=%s&UserId=%s", parentId, p.userId)
	resp := &FilterItemsResponse{}
	err := p.sendGet(genresUrl, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (p *PluginImpl) getStudios(parentId string) ([]*FilterItem, error) {
	genresUrl := fmt.Sprintf("/emby/Studios?ParentId=%s&UserId=%s", parentId, p.userId)
	resp := &FilterItemsResponse{}
	err := p.sendGet(genresUrl, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (p *PluginImpl) getTags(parentId string) ([]*FilterItem, error) {
	genresUrl := fmt.Sprintf("/emby/Tags?ParentId=%s&UserId=%s", parentId, p.userId)
	resp := &FilterItemsResponse{}
	err := p.sendGet(genresUrl, nil, resp)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// buildMenuItems converts a list of PluginItems into PluginMenu entries.
func buildMenuItems(label string, items []*plugin.PluginItem) []*plugin.PluginMenu {
	result := make([]*plugin.PluginMenu, 0, len(items))
	for _, item := range items {
		result = append(result, &plugin.PluginMenu{
			Menu: item,
		})
	}
	return result
}

// embyItemToPluginMedia converts an Emby item to a PluginMedia struct.
func (p *PluginImpl) embyItemToPluginMedia(item *EmbyItem) *plugin.PluginMedia {
	addr := strings.TrimRight(p.embyAuth.Addr.StringValue.Value, "/")

	media := &plugin.PluginMedia{
		MediaId:       item.Id,
		Name:          item.Name,
		ParentMediaId: item.ParentId,
	}

	// Set media type based on emby item type
	switch item.Type {
	case "Series":
		media.MediaType = plugin.PluginMedia_MEDIA_SERIES
	case "Season":
		media.MediaType = plugin.PluginMedia_MEDIA_INFO
		media.Name = fmt.Sprintf("%s - %s", item.SeriesName, item.Name)
	case "Episode":
		media.MediaType = plugin.PluginMedia_MEDIA_PLAY_ITEM
		media.Name = fmt.Sprintf("E%d - %s", item.IndexNumber, item.Name)
		media.PlayIndex = uint64(item.IndexNumber)
		if item.RunTimeTicks > 0 {
			media.Duration = uint64(item.RunTimeTicks / 10000000)
		}
	default:
		if item.MediaType == "Video" || item.Type == "Movie" {
			media.MediaType = plugin.PluginMedia_MEDIA_PLAY_ITEM
			if item.RunTimeTicks > 0 {
				media.Duration = uint64(item.RunTimeTicks / 10000000)
			}
		} else if item.IsFolder {
			media.MediaType = plugin.PluginMedia_MEDIA_INFO
		} else {
			media.MediaType = plugin.PluginMedia_MEDIA_PLAY_ITEM
		}
	}

	// Overview / description
	if item.Overview != "" {
		media.Desc = item.Overview
	}

	// Release date and year
	if item.PremiereDate != "" {
		media.ReleaseDate = item.PremiereDate
	}
	if item.ProductionYear > 0 {
		media.Year = uint64(item.ProductionYear)
	}

	// Genres
	if len(item.GenreItems) > 0 {
		media.Genres = make([]string, len(item.GenreItems))
		for i, g := range item.GenreItems {
			media.Genres[i] = g.Name
		}
	}

	// Image URLs using emby's image API
	if item.ImageTags != nil {
		if _, ok := item.ImageTags["Primary"]; ok {
			media.PosterUrl = fmt.Sprintf("%s/emby/Items/%s/Images/Primary?UserId=%s", addr, item.Id, p.userId)
		}
		if _, ok := item.ImageTags["Backdrop"]; ok {
			media.BackdropUrl = fmt.Sprintf("%s/emby/Items/%s/Images/Backdrop?UserId=%s", addr, item.Id, p.userId)
		}
	}

	return media
}
