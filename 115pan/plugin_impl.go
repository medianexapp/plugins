package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"plugins/util"
	"strconv"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/medianexapp/plugin_api/httpclient"
	"github.com/medianexapp/plugin_api/plugin"
	"github.com/medianexapp/plugin_api/ratelimit"

	_ "github.com/labulakalia/wazero_net/wasi/http"
)

/*
NOTE: net and http use package
"github.com/labulakalia/wazero_net/wasi/http"
"github.com/labulakalia/wazero_net/wasi/net"
*/

type PluginImpl struct {
	token    *plugin.Token
	userInfo *UserInfo

	client    *httpclient.Client
	ratelimit *ratelimit.RateLimit
}

func NewPluginImpl() *PluginImpl {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	client := httpclient.NewClient()
	limitConfigMap := map[string]ratelimit.LimitConfig{
		"/open/ufile/downurl": ratelimit.LimitConfig{
			Limit:    1,
			Duration: 3 * time.Second,
		},
		"/open/video/subtitle": ratelimit.LimitConfig{
			Limit:    1,
			Duration: 3 * time.Second,
		},
		"/open/video/play": ratelimit.LimitConfig{
			Limit:    1,
			Duration: 3 * time.Second,
		},
		"/open/ufile/files": ratelimit.LimitConfig{
			Limit:    1,
			Duration: 1 * time.Second,
		},
	}
	return &PluginImpl{
		client:    client,
		ratelimit: ratelimit.New(limitConfigMap),
	}
}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "115pan", nil
}

// GetAuthe implements IPlugin.
// Note: not store var in GetAuth
func (p *PluginImpl) GetAuth() (*plugin.Auth, error) {
	qrcodeData, err := util.GetAuthQrcode("115pan")
	if err != nil {
		return nil, err
	}
	authDeviceCodeData := &AuthDeviceCodeData{}
	resp := &QrResponse{
		Data: authDeviceCodeData,
	}
	err = json.Unmarshal(qrcodeData, resp)
	if err != nil {
		return nil, err
	}
	auth := &plugin.Auth{
		AuthMethods: []*plugin.AuthMethod{},
	}
	if authDeviceCodeData.Qrcode != "" {
		qrCode, err := qr.Encode(authDeviceCodeData.Qrcode, qr.M, qr.Auto)
		if err != nil {
			return nil, err
		}
		qrCode, err = barcode.Scale(qrCode, 200, 200)
		if err != nil {
			return nil, err
		}
		buf := &bytes.Buffer{}
		err = png.Encode(buf, qrCode)
		if err != nil {
			return nil, err
		}
		p := map[string]string{
			"uid":  authDeviceCodeData.Uid,
			"time": fmt.Sprint(authDeviceCodeData.Time),
			"sign": authDeviceCodeData.Sign,
		}
		data, err := json.Marshal(p)
		if err != nil {
			return nil, err
		}
		scanQrcode := &plugin.AuthMethod_Scanqrcode{
			Scanqrcode: &plugin.Scanqrcode{
				QrcodeImage:      buf.Bytes(),
				QrcodeImageParam: string(data),
			},
		}
		auth.AuthMethods = append(auth.AuthMethods, &plugin.AuthMethod{
			Method: scanQrcode,
		})
	}
	url := util.GetAuthAddr("115pan")
	authCallbackUrl := &plugin.AuthMethod_Callback{
		Callback: &plugin.Callback{
			CallbackUrl: url,
		},
	}
	auth.AuthMethods = append(auth.AuthMethods, &plugin.AuthMethod{
		Method: authCallbackUrl,
	})
	return auth, nil
}

// CheckAuth implements IPlugin.
func (p *PluginImpl) CheckAuthMethod(authMethod *plugin.AuthMethod) (*plugin.AuthData, error) {
	var (
		token *plugin.Token
		err   error
	)
	switch v := authMethod.Method.(type) {
	case *plugin.AuthMethod_Refresh:
		token = &plugin.Token{}
		err = token.UnmarshalVT(v.Refresh.AuthData.AuthDataBytes)
		if err != nil {
			return nil, err
		}

		token, err = util.GetAuthToken(&util.GetAuthTokenRequest{
			Id:           "115pan",
			RefreshToken: token.RefreshToken,
		})
	case *plugin.AuthMethod_Scanqrcode:
		token, err = util.CheckAuthQrcode("115pan", v.Scanqrcode.QrcodeImageParam)
		if err != nil {
			// if err,need refresh
			// 115pan qrcode if scan,qrcode will expire
			return nil, err
		}
		if token == nil {
			// scan not success
			slog.Warn("qrcode not scan success")
			return nil, nil
		}
	case *plugin.AuthMethod_Callback:
		slog.Info("recv callback data", "callBackData", v.Callback.CallbackUrlData)
		token = &plugin.Token{}
		urlParse, err := url.Parse(v.Callback.CallbackUrlData)
		if err != nil {
			return nil, err
		}
		data, err := base64.URLEncoding.DecodeString(urlParse.Query().Get("token"))
		if err != nil {
			return nil, err
		}
		err = token.UnmarshalVT(data)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported auth method")
	}

	if err != nil {
		slog.Error("authCode to access token failed", "err", err)
		return nil, err
	}
	tokenBytes, err := token.MarshalVT()
	if err != nil {
		slog.Error("marshal token failed", "err", err)
		return nil, err
	}
	expireTime := time.Now().Add(time.Second * time.Duration(token.ExpiresIn-300)).Unix()
	authData := &plugin.AuthData{
		AuthDataBytes:       tokenBytes,
		AuthDataExpiredTime: uint64(expireTime),
	}
	slog.Info("get access token success")
	return authData, nil
}

// InitAuth implements IPlugin.
func (p *PluginImpl) CheckAuthData(authDataBytes []byte) error {
	token := &plugin.Token{}
	err := token.UnmarshalVT(authDataBytes)
	if err != nil {
		slog.Error("unmarshal token failed", "err", err)
		return err
	}
	p.token = token

	resp := &Response{}
	err = p.send(http.MethodGet, "/open/user/info", nil, resp)
	if err != nil {
		slog.Error("get user info failed", "err", err)
		return err
	}
	if resp.State == false {
		slog.Error("get user info failed", "err", err)
		return errors.New(resp.Message)
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		slog.Error("marshal user info failed", "err", err)
		return err
	}
	userInfo := &UserInfo{}
	err = json.Unmarshal(data, &userInfo)
	if err != nil {
		slog.Error("unmarshal user info failed", "err", err)
		return err
	}
	p.userInfo = userInfo
	return nil
}

// AuthId implements IPlugin.
func (p *PluginImpl) PluginAuthId() (string, error) {
	if p.userInfo == nil {
		return "", errors.New("userInfo is nil")
	}
	return fmt.Sprint(p.userInfo.UserId), nil
}

// GetDirEntry implements IPlugin.
func (p *PluginImpl) GetDirEntry(req *plugin.GetDirEntryRequest) (*plugin.DirEntry, error) {
	slog.Debug("get dir entry ", "req", req)

	if req.Page == 0 {
		req.Page = 1
	}

	fileEntries := []*FileEntry{}
	resp := &Response{
		Data: &fileEntries,
	}
	fid := ""
	if req.FileEntry != nil && req.FileEntry.RawData != nil {
		fileEntry := FileEntry{}
		err := json.Unmarshal(req.FileEntry.RawData, &fileEntry)
		if err != nil {
			return nil, err
		}
		fid = fileEntry.Fid
	}
	slog.Debug("get dir entry ", "fid", fid)

	u := url.Values{}
	u.Add("cid", fid)
	u.Add("show_dir", "1")
	u.Add("o", "file_name")
	u.Add("offset", fmt.Sprint((req.Page-1)*req.PageSize))
	u.Add("limit", fmt.Sprint(req.PageSize))
	u.Add("order", "file_name")
	p.ratelimit.Wait("/open/ufile/files")
	err := p.send(http.MethodGet, "/open/ufile/files?"+u.Encode(), nil, resp)
	if err != nil {
		return nil, err
	}
	slog.Debug("get send ")

	if resp.State == false {
		slog.Error("get dir failed", "err", err)
		return nil, errors.New(resp.Message)
	}
	dirEntry := &plugin.DirEntry{
		FileEntries: []*plugin.FileEntry{},
	}
	for _, fileEntry := range fileEntries {
		entry := &plugin.FileEntry{
			Name:         fileEntry.Fn,
			Size:         uint64(fileEntry.Fs),
			CreatedTime:  uint64(fileEntry.Uppt),
			ModifiedTime: uint64(fileEntry.Upt),
			AccessedTime: uint64(fileEntry.Upt),
		}
		entryBytes, err := json.Marshal(fileEntry)
		if err == nil {
			entry.RawData = entryBytes
		}
		if fileEntry.Fc == "0" {
			entry.FileType = plugin.FileEntry_FileTypeDir
		} else {
			entry.FileType = plugin.FileEntry_FileTypeFile
		}
		dirEntry.FileEntries = append(dirEntry.FileEntries, entry)
	}
	slog.Debug("return")

	return dirEntry, nil
}

// GetFileResource implements IPlugin.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	if req.FileEntry == nil || req.FileEntry.RawData == nil {
		return nil, fmt.Errorf("invalid path %s", req.FilePath)
	}

	fileResource := &plugin.FileResource{
		FileResourceData: []*plugin.FileResource_FileResourceData{}}
	fileEntry := &FileEntry{}
	err := json.Unmarshal(req.FileEntry.RawData, fileEntry)
	if err != nil {
		return nil, err
	}
	reqURL := url.Values{}
	reqURL.Add("pick_code", fileEntry.Pc)

	respData := map[string]FileURL{}
	resp := Response{
		Data: &respData,
	}
	p.ratelimit.Wait("/open/ufile/downurl")
	err = p.send(http.MethodPost, "/open/ufile/downurl", reqURL, &resp)
	if err != nil {
		return nil, err
	}
	if resp.State == true {
		fileURL, ok := respData[fileEntry.Fid]
		if ok {
			uu, err := url.Parse(fileURL.Url.Url)
			if err != nil {
				return nil, err
			}
			expireTime, err := strconv.ParseUint(uu.Query().Get("t"), 10, 0)
			if err != nil {
				return nil, err
			}
			fileResource.FileResourceData = append(fileResource.FileResourceData, &plugin.FileResource_FileResourceData{
				Url:          fileURL.Url.Url,
				Resolution:   plugin.FileResource_Original,
				ResourceType: plugin.FileResource_Video,
				ExpireTime:   expireTime,
				Header: map[string]string{
					"User-Agent": httpclient.GetDefaultUserAgent(),
				},
			})
		}
	} else {
		slog.Error("get down file failed", "msg", resp.Message)
	}

	if req.IsMedia {

		subtitleData := &SubtitleData{
			List: []Subtitle{},
		}
		resp = Response{
			Data: &subtitleData,
		}
		p.ratelimit.Wait("/open/video/subtitle")
		err = p.send(http.MethodGet, "/open/video/subtitle?"+reqURL.Encode(), nil, &resp)
		if err != nil {
			return nil, err
		}
		for _, subtitle := range subtitleData.List {
			fileResource.FileResourceData = append(fileResource.FileResourceData, &plugin.FileResource_FileResourceData{
				Url:          subtitle.URL,
				ResourceType: plugin.FileResource_Subtitle,
				Title:        subtitle.Title,
				Header: map[string]string{
					"User-Agent": httpclient.GetDefaultUserAgent(),
				},
			})
		}

		playVideoInfo := PlayVideoInfo{}
		resp = Response{
			Data: &playVideoInfo,
		}
		// get video play address
		p.ratelimit.Wait("/open/video/play")
		err = p.send(http.MethodGet, "/open/video/play?"+reqURL.Encode(), nil, &resp)
		if err != nil {
			return nil, err
		}
		if resp.State == true {
			for _, playVideoInfo := range playVideoInfo.VideoURL {
				data := &plugin.FileResource_FileResourceData{
					Url:          playVideoInfo.URL,
					ResourceType: plugin.FileResource_Video,
					Title:        playVideoInfo.Title,
					Header: map[string]string{
						"User-Agent": httpclient.GetDefaultUserAgent(),
					},
				}
				if playVideoInfo.DefinitionN == 1 {
					data.Resolution = plugin.FileResource_SD
				} else if playVideoInfo.DefinitionN == 2 {
					data.Resolution = plugin.FileResource_LD
				} else if playVideoInfo.DefinitionN == 3 {
					data.Resolution = plugin.FileResource_HD
				} else if playVideoInfo.DefinitionN == 4 {
					data.Resolution = plugin.FileResource_FHD
				} else if playVideoInfo.DefinitionN == 4 {
					data.Resolution = plugin.FileResource_UHD
				} else if playVideoInfo.DefinitionN == 5 {
					data.Resolution = plugin.FileResource_Original
				}
				fileResource.FileResourceData = append(fileResource.FileResourceData, data)
			}
		} else {
			slog.Error("get play video info failed", "msg", resp.Message)
		}
	}

	return fileResource, nil
}

func (p *PluginImpl) send(method string, uri string, req, resp any) error {
	var body io.Reader
	if req != nil {
		urlValue, ok := req.(url.Values)
		if !ok {
			return errors.New("req not is urlValues")
		}
		body = strings.NewReader(urlValue.Encode())
	}

	httpReq, err := http.NewRequest(method, fmt.Sprintf("%s%s", Api115PanAddr, uri), body)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if p.token == nil {
		return errors.New("token is nil")
	}
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.token.AccessToken))
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return err
	}

	bodyData, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}
	slog.Debug("get request resp", "uri", uri, "req", req, "status_code", httpResp.StatusCode, "header", httpResp.Header, "resp", string(bodyData))
	defer httpResp.Body.Close()

	err = json.Unmarshal(bodyData, resp)
	if err != nil {
		slog.Error("unmarshal body data failed", "err", err)
		return err
	}
	return nil
}
