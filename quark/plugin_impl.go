package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	_ "github.com/labulakalia/wazero_net/wasi/http" // if you need http import this
	"github.com/medianexapp/plugin_api/httpclient"
	"github.com/medianexapp/plugin_api/plugin"
	"github.com/medianexapp/plugin_api/ratelimit"
)

const (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) quark-cloud-drive/3.20.0 Chrome/112.0.5615.165 Electron/24.1.3.8 Safari/537.36 Channel/pckk_other_ch"
	referer   = "https://pan.quark.cn"
	api       = "https://drive-pc.quark.cn/1/clouddrive"
	pr        = "ucpro"
)

type PluginImpl struct {
	cookie    string
	client    *httpclient.Client
	ratelimit *ratelimit.RateLimit
}

func NewPluginImpl() *PluginImpl {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	limitConfigMap := map[string]ratelimit.LimitConfig{
		"": ratelimit.LimitConfig{
			Limit:    1,
			Duration: time.Second,
		},
	}
	return &PluginImpl{
		client:    httpclient.NewClient(httpclient.WithUserAgent(userAgent)),
		ratelimit: ratelimit.New(limitConfigMap),
	}
}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "quark", nil
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
								Name:  "Cookie",
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
	slog.Debug("CheckAuthMethod", "authMethod", authMethod)
	authDataBytes, err := authMethod.Method.(*plugin.AuthMethod_Formdata).Formdata.MarshalVT()
	if err != nil {
		return nil, err
	}
	return &plugin.AuthData{AuthDataBytes: authDataBytes}, nil
}

// CheckAuthData use authDataBytes to uath
// you must store auth data to *PluginImpl
func (p *PluginImpl) CheckAuthData(authDataBytes []byte) error {
	slog.Debug("CheckAuthData", "authDataBytes", authDataBytes)
	formdata := &plugin.Formdata{}
	if err := formdata.UnmarshalVT(authDataBytes); err != nil {
		return err
	}
	p.cookie = formdata.FormItems[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value
	err := p.request("/config", http.MethodGet, nil, nil, nil)
	if err != nil {
		return err
	}
	// fmt.Println("check auth data", string(res))
	// https://pan.quark.cn/account/info?fr=pc&platform=pc
	return nil
}

// PluginAuthId implements IPlugin.
// plugin auth id,you can generate id by md5 or sha
func (p *PluginImpl) PluginAuthId() (string, error) {
	return "quark", nil
}

// GetDirEntry implements IPlugin.
// return dir file entry
// save your driver file raw data to FileEntry.RawData,you can get it after GetDirEntry and GetFileResource request
// default page_size if 100,if this not for you,change is on DirEntry.PageSize,will use new PageSize for next request
func (p *PluginImpl) GetDirEntry(req *plugin.GetDirEntryRequest) (*plugin.DirEntry, error) {
	slog.Debug("GetDirEntry", "req", req)
	var pdirFid string
	if req.Path == "/" {
		pdirFid = "0"
	} else {
		file := File{}
		if req.FileEntry == nil || req.FileEntry.RawData == nil {
			return nil, errors.New("file entry is nil")
		}
		err := json.Unmarshal(req.FileEntry.RawData, &file)
		if err != nil {
			return nil, err
		}
		pdirFid = file.Fid
	}
	if req.PageSize > 50 {
		req.PageSize = 50
	}
	u := url.Values{}
	u.Add("pdir_fid", pdirFid)
	u.Add("_page", fmt.Sprint(req.Page))
	u.Add("_size", fmt.Sprint(req.PageSize))
	u.Add("_fetch_total", "1")
	fileData := &FileData{
		List: []File{},
	}
	err := p.request("/file/sort", http.MethodGet, u, nil, fileData)
	if err != nil {
		return nil, err
	}
	dirEntry := &plugin.DirEntry{
		PageSize:    50,
		FileEntries: []*plugin.FileEntry{},
	}
	for _, file := range fileData.List {
		fileEntry := &plugin.FileEntry{
			Name:         file.FileName,
			Size:         file.Size,
			CreatedTime:  file.CreatedAt / 1000,
			ModifiedTime: file.UpdatedAt / 1000,
			AccessedTime: file.UpdatedViewAt,
		}
		if file.File {
			fileEntry.FileType = plugin.FileEntry_FileTypeFile
		} else {
			fileEntry.FileType = plugin.FileEntry_FileTypeDir
		}
		fileRawData, err := json.Marshal(file)
		if err == nil {
			fileEntry.RawData = fileRawData
		}
		dirEntry.FileEntries = append(dirEntry.FileEntries, fileEntry)
	}
	return dirEntry, nil
}

// GetFileResource implements IPlugin.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	slog.Debug("GetFileResource", "req", req)
	file := File{}
	if req.FileEntry == nil || req.FileEntry.RawData == nil {
		return nil, errors.New("file entry is nil")
	}
	err := json.Unmarshal(req.FileEntry.RawData, &file)
	if err != nil {
		return nil, err
	}
	fileResource := &plugin.FileResource{
		FileResourceData: []*plugin.FileResource_FileResourceData{},
	}
	data := map[string][]string{
		"fids": {file.Fid},
	}
	respData := []File{}
	err = p.request("/file/download", http.MethodPost, nil, data, &respData)
	if err != nil {
		return nil, err
	}
	if len(respData) == 1 {
		expireTime, err := getExpires(respData[0].DownloadUrl)
		if err != nil {
			slog.Error("get expires failed", "url", respData[0].DownloadUrl, "err", err)
		} else {
			fileResource.FileResourceData = append(fileResource.FileResourceData, &plugin.FileResource_FileResourceData{
				Url:          respData[0].DownloadUrl,
				Resolution:   plugin.FileResource_Original,
				ResourceType: plugin.FileResource_Video,
				Header: map[string]string{
					"Cookie":     p.cookie,
					"Referer":    referer,
					"User-Agent": userAgent,
				},
				ExpireTime:         expireTime,
				Size:               req.FileEntry.Size,
				Proxy:              true,
				ProxyChunkParallel: 3,
				ProxyChunkSize:     1024 * 1024 * 5,
			})
		}
	}
	if req.IsMedia {
		// 获取播放链接
		uri := "/file/v2/play"
		reqData := PlayReq{
			Fid:         file.Fid,
			Resolutions: "normal,low,high,super,2k,4k",
			Supports:    "fmp4,m3u8",
		}
		u := url.Values{}
		u.Add("uc_param_str", "")
		playData := PlayData{
			VideoList: []VideoList{},
		}
		err = p.request(uri, http.MethodPost, u, reqData, &playData)
		if err != nil {
			return nil, err
		}
		// 4k
		// super 2k
		// 1080p
		// 720p
		for _, item := range playData.VideoList {
			if item.VideoInfo.URL == "" {
				continue
			}
			expireTime, err := getExpires(item.VideoInfo.URL)
			if err != nil {
				slog.Error("get expires failed", "url", item.VideoInfo.URL, "err", err)

			}
			fileResource.FileResourceData = append(fileResource.FileResourceData, &plugin.FileResource_FileResourceData{
				Url:          item.VideoInfo.URL,
				Resolution:   resolutionMap[item.Resolution],
				ResourceType: plugin.FileResource_Video,
				Header: map[string]string{
					"Cookie":     p.cookie,
					"Referer":    referer,
					"User-Agent": userAgent,
				},
				ExpireTime: expireTime,
			})
		}
	}

	return fileResource, nil
}

func (p *PluginImpl) request(uri string, method string, u url.Values, reqData, respData any) error {
	if u == nil {
		u = url.Values{}
	}
	p.ratelimit.Wait("")
	u.Add("pr", pr)
	u.Add("fr", "pc")
	var body io.Reader
	if reqData != nil {
		data, err := json.Marshal(reqData)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(data)
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s?%s", api, uri, u.Encode()), body)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", p.cookie)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", userAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "__puus" {
			h := http.Header{}
			h.Add("Cookie", p.cookie)
			cookieStrs := []string{}
			r := http.Request{Header: h}
			for _, oldCookie := range r.Cookies() {
				oldCookieStr := oldCookie.String()
				if oldCookie.Name == "__puus" {
					oldCookieStr = cookie.String()
				}
				cookieStrs = append(cookieStrs, oldCookieStr)
			}
			p.cookie = strings.Join(cookieStrs, ";")
		}
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	response := Response{
		Data: respData,
	}
	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return err
	}
	if response.Code != 0 {
		slog.Error("resp code failed", "response", response)
		return fmt.Errorf("%s", response.Message)
	}

	defer resp.Body.Close()
	return nil
}

func getExpires(u string) (uint64, error) {
	p, err := url.Parse(u)
	if err != nil {
		return 0, err
	}
	expires := p.Query().Get("Expires")
	epInt, err := strconv.Atoi(expires)
	if err != nil {
		p, _ = url.Parse(u)
		sp := strings.Split(p.Query().Get("auth_key"), "-")
		if len(sp) > 0 {
			epInt, err = strconv.Atoi(sp[0])
			if err != nil {
				return 0, err
			}
		}
	}
	return uint64(epInt), nil
}
