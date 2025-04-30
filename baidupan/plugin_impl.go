//go:build wasip1

package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"plugins/util"
	"time"

	netutil "github.com/labulakalia/wazero_net/util"
	_ "github.com/labulakalia/wazero_net/wasi/http"
	"github.com/medianexapp/plugin_api/httpclient"
	"github.com/medianexapp/plugin_api/plugin"
)

/*
NOTE: net and http use package
"github.com/labulakalia/wazero_net/wasi/http"
"github.com/labulakalia/wazero_net/wasi/net"
*/

type PluginImpl struct {
	client   *httpclient.Client
	token    *plugin.Token
	userInfo *UserInfo
}

func NewPluginImpl() *PluginImpl {
	return &PluginImpl{
		client: httpclient.NewClient(),
	}
}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "baidupan", nil
}

// GetAuthe implements IPlugin.
// Note: not store var in GetAuth
func (p *PluginImpl) GetAuth() (*plugin.Auth, error) {
	auth := &plugin.Auth{
		AuthMethods: []*plugin.AuthMethod{},
	}
	authUrl := util.GetAuthAddr("baidupan")
	callback := &plugin.AuthMethod_Callback{
		Callback: &plugin.Callback{
			CallbackUrl: authUrl,
		},
	}

	auth.AuthMethods = append(auth.AuthMethods, &plugin.AuthMethod{
		Method: callback,
	})

	res, err := util.GetAuthQrcode("baidupan")
	if err != nil {
		slog.Error("get auth qrcode failed", "err", err)
		return nil, err
	}
	qrCode := QrcodeData{}
	err = json.Unmarshal(res, &qrCode)
	if err != nil {
		slog.Error("unmarshal auth qrcode failed", "err", err)
		return nil, err
	}
	qrResp, err := p.client.Get(qrCode.QrcodeURL)
	if err != nil {
		return nil, err
	}
	qrRespBytes, err := io.ReadAll(qrResp.Body)
	if err != nil {
		return nil, err
	}
	if qrResp.StatusCode != http.StatusOK {
		return nil, errors.New(netutil.BytesToString(qrRespBytes))
	}
	authQrcode := &plugin.AuthMethod_Scanqrcode{
		Scanqrcode: &plugin.Scanqrcode{
			QrcodeImage:      qrRespBytes,
			QrcodeImageParam: qrCode.DeviceCode,
			QrcodeExpireTime: uint64(time.Now().Unix()) + uint64(qrCode.ExpiresIn),
		},
	}

	auth.AuthMethods = append(auth.AuthMethods, &plugin.AuthMethod{
		Method: authQrcode,
	})
	return auth, nil
}

// CheckAuth implements IPlugin.
func (p *PluginImpl) CheckAuthMethod(authMethod *plugin.AuthMethod) (authData *plugin.AuthData, err error) {
	var (
		token        *plugin.Token
		authCode     string
		refreshToken string
		state        string
	)
	switch v := authMethod.Method.(type) {
	case *plugin.AuthMethod_Scanqrcode:
		token, err = util.CheckAuthQrcode("baidupan", v.Scanqrcode.QrcodeImageParam)
		if err != nil {
			return nil, err
		}
		if token == nil {
			return nil, nil
		}
	case *plugin.AuthMethod_Refresh:
		token := &plugin.Token{}
		err = token.UnmarshalVT(v.Refresh.AuthData.AuthDataBytes)
		if err != nil {
			return nil, err
		}
		refreshToken = token.RefreshToken
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
		return nil, fmt.Errorf("unsupport %+v", v)
	}
	if token == nil {
		token, err = util.GetAuthToken(&util.GetAuthTokenRequest{
			Id:           "baidupan",
			Code:         authCode,
			State:        state,
			RefreshToken: refreshToken,
		})
		if err != nil {
			slog.Error("authCode to access token failed", "err", err)
			return nil, err
		}
	}

	tokenBytes, err := token.MarshalVT()
	if err != nil {
		slog.Error("marshal token failed", "err", err)
		return nil, err
	}
	expireTime := time.Now().Add(time.Second * time.Duration(token.ExpiresIn-300)).Unix()
	authData = &plugin.AuthData{
		AuthDataBytes:       tokenBytes,
		AuthDataExpiredTime: uint64(expireTime),
	}
	slog.Info("get access token success")
	return authData, nil
}

func (p *PluginImpl) sendData(_ string, uri string, u url.Values, resp any) error {
	u.Add("access_token", p.token.AccessToken)
	reqUrl := fmt.Sprintf("%s%s?%s", BaiduPanURL, uri, u.Encode())
	reqResp, err := p.client.Get(reqUrl)
	if err != nil {
		slog.Error("request url failed", "req", reqUrl)
		return err
	}
	defer reqResp.Body.Close()
	body, err := io.ReadAll(reqResp.Body)
	if err != nil {
		slog.Error("read response body failed", "err", err)
		return err
	}
	respData := &Response{}
	err = json.Unmarshal(body, respData)
	if err != nil {
		slog.Error("unmarshal response failed", "err", err)
		return err
	}
	if respData.Errno != 0 {
		slog.Error("request failed", "errno", respData.Errno, "errmsg", respData.ErrMsg)
		return errors.New(respData.ErrMsg)
	}
	err = json.Unmarshal(body, resp)
	if err != nil {
		slog.Error("unmarshal response failed", "err", err)
		return err
	}
	return nil
}

// InitAuth implements IPlugin.
func (p *PluginImpl) CheckAuthData(authDataBytes []byte) error {
	token := &plugin.Token{}
	err := token.UnmarshalVT(authDataBytes)
	if err != nil {
		return err
	}
	p.token = token
	userInfo := &UserInfo{}
	u := url.Values{}
	u.Add("method", "uinfo")
	err = p.sendData(http.MethodGet, "/rest/2.0/xpan/nas", u, userInfo)
	if err != nil {
		return err
	}
	p.userInfo = userInfo
	return nil
}

// AuthId implements IPlugin.
func (p *PluginImpl) PluginAuthId() (string, error) {
	return fmt.Sprintf("%x", md5.Sum([]byte(p.userInfo.NetdiskName))), nil
}

// GetDirEntry implements IPlugin.
func (p *PluginImpl) GetDirEntry(req *plugin.GetDirEntryRequest) (*plugin.DirEntry, error) {
	u := url.Values{}
	u.Add("method", "list")
	u.Add("dir", req.Path)
	u.Add("order", "name")
	u.Add("desc", "1")
	u.Add("start", fmt.Sprint((req.Page-1)*req.PageSize))
	resp := &FileListResponse{
		List: []*FileListItem{},
	}
	err := p.sendData(http.MethodGet, "/rest/2.0/xpan/file", u, resp)
	if err != nil {
		slog.Error("list file failed", "err", err)
		return nil, err
	}
	dirEntry := plugin.DirEntry{
		FileEntries: []*plugin.FileEntry{},
	}
	for _, fileItem := range resp.List {
		entry := &plugin.FileEntry{
			Name:         fileItem.ServerFilename,
			Size:         fileItem.Size,
			CreatedTime:  fileItem.ServerCtime,
			ModifiedTime: fileItem.ServerMtime,
			AccessedTime: fileItem.ServerAtime,
			FileType:     plugin.FileEntry_FileTypeFile,
		}
		itemBytes, err := json.Marshal(fileItem)
		if err == nil {
			entry.RawData = itemBytes
		}
		if fileItem.IsDir == 1 {
			entry.FileType = plugin.FileEntry_FileTypeDir
		}
		dirEntry.FileEntries = append(dirEntry.FileEntries, entry)
	}
	return &dirEntry, nil
}

// GetFileResource implements IPlugin.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	fileItem := &FileListItem{}
	err := json.Unmarshal(req.FileEntry.RawData, fileItem)
	if err != nil {
		return nil, err
	}

	fileMetaResp := &FileMetasResponse{
		List: []*FileMetaItem{},
	}
	u := url.Values{}
	u.Add("method", "filemetas")
	u.Add("fsids", fmt.Sprintf("[%d]", fileItem.FsId))
	u.Add("dlink", "1")
	err = p.sendData(http.MethodGet, "/rest/2.0/xpan/multimedia", u, fileMetaResp)
	if err != nil {
		return nil, err
	}
	if len(fileMetaResp.List) == 0 {
		return nil, fmt.Errorf("file %s not found", req.FilePath)
	}
	fileResource := &plugin.FileResource{
		FileResourceData: []*plugin.FileResource_FileResourceData{
			{
				Url:        fmt.Sprintf("%s&access_token=%s", fileMetaResp.List[0].Dlink, p.token.AccessToken),
				ExpireTime: uint64(time.Now().Add(time.Hour * 8).Unix()),
				Resolution: plugin.FileResource_Original,
				Header: map[string]string{
					"Host":       "d.pcs.baidu.com",
					"User-Agent": "pan.baidu.com",
				},
			},
		},
	}
	return fileResource, nil
}
