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

	"github.com/hashicorp/go-uuid"

	_ "github.com/labulakalia/wazero_net/wasi/http" // if you need http import this
	"github.com/medianexapp/plugin_api"
	"github.com/medianexapp/plugin_api/httpclient"
	"github.com/medianexapp/plugin_api/plugin"
	"github.com/medianexapp/plugin_api/ratelimit"
)

const (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) quark-cloud-drive/3.24.0 Chrome/112.0.5615.165 Electron/24.1.3.8 Safari/537.36 Channel/pckk_other_ch"
	referer   = "https://pan.quark.cn"
	api       = "https://drive-pc.quark.cn/1/clouddrive"
	pr        = "ucpro"
	clientid  = "386"
)

type PluginImpl struct {
	plugin_api.IPlugin
	cookies   []*http.Cookie
	client    *httpclient.Client
	hb        *httpclient.Builder
	ratelimit *ratelimit.RateLimit // move to httpclient
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
		hb: httpclient.NewBuilder().SetUserAgent(userAgent).
			SetHeader("Referer", "https://pan.quark.cn/").
			SetHeader("Origin", "https://pan.quark.cn").
			SetQueryParam("pr", "ucpro").
			SetQueryParam("fr", "pc").
			SetQueryParam("platform", "pc"),
		client:    httpclient.NewClient(httpclient.WithUserAgent(userAgent)),
		ratelimit: ratelimit.New(limitConfigMap),
	}
}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "quark", nil
}

func (p *PluginImpl) checkQrcode(param string) ([]*http.Cookie, error) {
	checkQrurl := "https://uop.quark.cn/cas/ajax/getServiceTicketByQrcodeToken"

	rqId, err := uuid.GenerateUUID()
	if err != nil {
		return nil, err
	}
	u := url.Values{}
	u.Add("client_id", clientid)
	u.Add("v", "1.2")
	u.Add("request_id", rqId)
	u.Add("token", param)
	u.Add("uc_param_str", "utprfr")
	u.Add("ut", "NekR9SeMxs0xvu0ZMJEKfpU4YeEziTZ%2FBQ3rb11LvaF1VQ%3D%3D")
	u.Add("pr", "ucpro")
	u.Add("fr", "pc")
	ticket := &ServiceTocket{}
	resp := &Response{
		Data: ticket,
	}
	respHeader := http.Header{}
	err = p.hb.SetQueryParams(u).Get(checkQrurl).GetRespHeader(&respHeader).JSONResponse(&resp)
	if err != nil {
		return nil, err
	}
	p.hb.ParseCookie(respHeader)

	if resp.Status == 50004001 {
		slog.Warn("not get res", "msg", resp)
		return nil, nil
	}
	if resp.Status == 50004002 {
		slog.Warn("token is valid", "msg", resp.Message)
		return nil, errors.New(resp.Message)
	}

	respHeader2 := http.Header{}
	// https://pan.quark.cn/account/info?st=sta2c633397o8vz9iio35n2ki6ys5z84&lw=scan
	_, err = p.hb.SetQueryParam("st", ticket.Members.Ticket).
		SetQueryParam("lw", "scan").
		Get("https://pan.quark.cn/account/info").
		GetRespHeader(&respHeader2).BytesResponse()
	if err != nil {
		slog.Error("get account info failed", "err", err)
		return nil, err
	}
	p.hb.ParseCookie(respHeader2)
	return p.hb.GetCookies(), nil

}

func (p *PluginImpl) getQrcodeToken() (string, error) {
	qrcodeUrl := "https://uop.quark.cn/cas/ajax/getTokenForQrcodeLogin"

	rqId, err := uuid.GenerateUUID()
	if err != nil {
		return "", err
	}

	u := url.Values{}
	u.Add("client_id", clientid)
	u.Add("v", "1.2")
	u.Add("request_id", rqId)
	u.Add("uc_param_str", "utprfr")
	u.Add("ut", "NekR9SeMxs0xvu0ZMJEKfpU4YeEziTZ%2FBQ3rb11LvaF1VQ%3D%3D")

	token := &Token{}
	resp := &Response{
		Data: token,
	}
	respHeader := http.Header{}
	err = p.hb.SetQueryParams(u).Get(qrcodeUrl).GetRespHeader(&respHeader).JSONResponse(&resp)
	if err != nil {
		return "", err
	}
	if resp.Status != 2000000 {
		return "", errors.New(resp.Message)
	}
	p.hb.ParseCookie(respHeader)
	return token.Members.Token, nil
}

func (p *PluginImpl) qrcodeData(token string) string {
	return fmt.Sprintf("https://su.quark.cn/4_eMHBJ?token=%s&client_id=386&sch=pckk@other_ch&sve=3.20.0&ssb=weblogin&uc_param_str=&uc_biz_str=S:custom|OPT:SAREA@0|OPT:IMMERSIVE@1|OPT:BACK_BTN_STYLE@0", token)
}

// GetAuth return how to auth
// 1.FormData input data
// 2.Callback use url callback auth,like oauth
// 3.Scanqrcode,return qrcode image to auth
func (p *PluginImpl) GetAuth() (*plugin.Auth, error) {
	slog.Info("GetAuth")
	token, err := p.getQrcodeToken()
	if err != nil {
		return nil, err
	}
	qrdataUrl := p.qrcodeData(token)
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
				HelpDocUrl: "",
			},
			{
				Method: &plugin.AuthMethod_Scanqrcode{
					Scanqrcode: &plugin.Scanqrcode{
						QrcodeExpireTime:   uint64(time.Now().Add(time.Minute * 2).Unix()),
						QrcodeImageParam:   token,
						QrcodeImageContent: qrdataUrl,
					},
				},
				HelpDocUrl: "",
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

	var err error
	var cookies []*http.Cookie
	switch authMethod.Method.(type) {
	case *plugin.AuthMethod_Formdata:
		authDataBytes := authMethod.Method.(*plugin.AuthMethod_Formdata).Formdata.FormItems[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value
		cookies, err = http.ParseCookie(authDataBytes)
		if err != nil {
			return nil, err
		}
	case *plugin.AuthMethod_Scanqrcode:
		scanCode := authMethod.Method.(*plugin.AuthMethod_Scanqrcode).Scanqrcode
		cookies, err = p.checkQrcode(scanCode.QrcodeImageParam)
		if err != nil {
			return nil, err
		}
		if cookies == nil {
			return nil, nil
		}
	}
	cookiesBytes := p.convertCookie(cookies)
	return &plugin.AuthData{AuthDataBytes: []byte(cookiesBytes)}, err
}

// CheckAuthData use authDataBytes to uath
// you must store auth data to *PluginImpl
func (p *PluginImpl) CheckAuthData(authDataBytes []byte) error {
	slog.Debug("CheckAuthData", "authDataBytes", string(authDataBytes))
	cookies, err := http.ParseCookie(string(authDataBytes))
	if err != nil {
		return err
	}

	p.cookies = cookies
	err = p.request("/config", http.MethodGet, nil, nil, nil)
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
	cookie := p.convertCookie(p.cookies)
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
					"Cookie":     cookie,
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
					"Cookie":     cookie,
					"Referer":    referer,
					"User-Agent": userAgent,
				},
				ExpireTime: expireTime,
			})
		}
	}

	return fileResource, nil
}

func (p *PluginImpl) convertCookie(cookies []*http.Cookie) string {
	cc := []string{}
	for _, c := range cookies {
		cc = append(cc, c.String())
	}
	cookie := strings.Join(cc, "; ")
	return cookie
}

func (p *PluginImpl) request(uri string, method string, u url.Values, reqData, respData any) error {
	if u == nil {
		u = url.Values{}
	}
	p.ratelimit.Wait("")

	cb := p.hb.SetHeader("Cookie", p.convertCookie(p.cookies)).SetQueryParams(u).SetMethod(method).SetURL(fmt.Sprintf("%s%s", api, uri))
	var body io.Reader
	if reqData != nil {
		data, err := json.Marshal(reqData)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(data)
		cb = cb.SetBody(body)
	}

	response := Response{
		Data: respData,
	}
	respHeader := http.Header{}
	respBytes, err := cb.GetRespHeader(&respHeader).BytesResponse()
	if err != nil {
		return err
	}
	slog.Info("resp bytes", "data", string(respBytes))
	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return err
	}
	if response.Code != 0 {
		slog.Error("resp code failed", "response", response)
		return fmt.Errorf("%s", response.Message)
	}

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
