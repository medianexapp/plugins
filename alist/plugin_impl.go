package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/medianexapp/plugin_api/httpclient"

	"github.com/medianexapp/plugin_api/plugin"

	_ "github.com/labulakalia/wazero_net/wasi/http" // if you need http import this
	_ "github.com/labulakalia/wazero_net/wasi/net"  // if you need net.Conn import this
)

type PluginImpl struct {
	authData  *AuthData
	client    *httpclient.Client
	authToken string
}

type AuthData struct {
	Addr         string
	Username     string
	Password     string
	TokenExpired int64
}

func NewPluginImpl() *PluginImpl {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	return &PluginImpl{
		authData: &AuthData{},
		client:   httpclient.NewClient(),
	}
}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "alist", nil
}

// GetAuth return how to auth
// 1.FormData input data
// 2.Callback use url callback auth,like oauth
// 3.Scanqrcode,return qrcode image to auth
func (p *PluginImpl) GetAuth() (*plugin.Auth, error) {
	auth := &plugin.Auth{
		AuthMethods: []*plugin.AuthMethod{
			{
				Method: &plugin.AuthMethod_Formdata{
					Formdata: &plugin.Formdata{
						FormItems: []*plugin.Formdata_FormItem{
							{
								Name:  "Addr",
								Value: plugin.String("http://127.0.0.1:5244"),
							},
							{
								Name:  "Username",
								Value: plugin.String(""),
							},
							{
								Name:  "Password",
								Value: plugin.ObscureString(""),
							},
							{
								Name:  "Default Token Expired(H)",
								Value: plugin.Int64(48),
							},
						},
					},
				},
			},
		},
	}
	return auth, nil
}

func (p *PluginImpl) request(method string, uri string, reqData, respData any) error {
	var body io.Reader
	if reqData != nil {
		data, err := json.Marshal(reqData)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		body = bytes.NewReader(data)
	}
	slog.Debug("alist request", "req", reqData)
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", p.authData.Addr, uri), body)
	if err != nil {
		return err
	}

	if p.authToken != "" {
		req.Header.Set("Authorization", p.authToken)
	}
	req.Header.Add("Content-Type", "application/json")
	httpResp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}
	resp := &Response{
		Data: respData,
	}
	err = json.Unmarshal(data, resp)
	if err != nil {
		return err
	}
	if resp.Code != 200 {
		return fmt.Errorf("%s", resp.Message)
	}
	return nil
}

// CheckAuthMethod check auth is finished and return authDataBytes and authData's expired time
// if authmethod's type is *plugin.AuthMethod_Refresh,you need to refresh token
// assert authMethod.Method's type to check auth is finished,return auth data and expired time if authed
func (p *PluginImpl) CheckAuthMethod(authMethod *plugin.AuthMethod) (*plugin.AuthData, error) {
	slog.Debug("CheckAuthMethod", "authMethod", authMethod)
	switch data := authMethod.Method.(type) {
	case *plugin.AuthMethod_Refresh:
	case *plugin.AuthMethod_Formdata:
		forms := data.Formdata.FormItems
		p.authData.Addr = forms[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value
		p.authData.Username = forms[1].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value
		p.authData.Password = forms[2].Value.(*plugin.Formdata_FormItem_ObscureStringValue).ObscureStringValue.Value
		p.authData.TokenExpired = forms[3].Value.(*plugin.Formdata_FormItem_Int64Value).Int64Value.Value
	}
	authResp := &TokenData{}
	err := p.request(http.MethodPost, "/api/auth/login", &AuthLogin{
		Username: p.authData.Username,
		Password: p.authData.Password,
	}, authResp)
	if err != nil {
		return nil, err
	}
	return &plugin.AuthData{
		AuthDataBytes:       []byte(authResp.Token),
		AuthDataExpiredTime: uint64(time.Hour * time.Duration(p.authData.TokenExpired)),
	}, nil
}

// CheckAuthData use authDataBytes to uath
// you must store auth data to *PluginImpl
func (p *PluginImpl) CheckAuthData(authDataBytes []byte) error {
	slog.Debug("CheckAuthData", "authDataBytes", authDataBytes)
	p.authToken = string(authDataBytes)
	err := p.request(http.MethodGet, "/api/me", nil, nil)
	if err != nil {
		return err
	}
	return err
}

// PluginAuthId implements IPlugin.
// plugin auth id,you can generate id by md5 or sha
func (p *PluginImpl) PluginAuthId() (string, error) {
	data, err := json.Marshal(p.authData)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", md5.Sum(data)), nil
}

// GetDirEntry implements IPlugin.
// return dir file entry
// save your driver file raw data to FileEntry.RawData,you can get it after GetDirEntry and GetFileResource request
// default page_size if 100,if this not for you,change is on DirEntry.PageSize,will use new PageSize for next request
func (p *PluginImpl) GetDirEntry(req *plugin.GetDirEntryRequest) (*plugin.DirEntry, error) {
	slog.Debug("GetDirEntry", "req", req.FileEntry)
	fsResp := &FsListResp{
		Contents: []Content{},
	}
	fsReq := &FsListReq{
		Path:    req.Path,
		Page:    int64(req.Page),
		PerPage: int64(req.PageSize),
		Refresh: false,
	}
	err := p.request(http.MethodPost, "/api/fs/list", fsReq, fsResp)
	if err != nil {
		return nil, err
	}
	dirEntry := plugin.DirEntry{
		FileEntries: []*plugin.FileEntry{},
	}
	for _, content := range fsResp.Contents {
		fileEntry := &plugin.FileEntry{
			Name:         content.Name,
			ModifiedTime: uint64(content.Modified.Unix()),
			AccessedTime: uint64(content.Modified.Unix()),
			CreatedTime:  uint64(content.Created.Unix()),
			Size:         uint64(content.Size),
		}
		if content.IsDir {
			fileEntry.FileType = plugin.FileEntry_FileTypeDir
		} else {
			fileEntry.FileType = plugin.FileEntry_FileTypeFile
		}
		dirEntry.FileEntries = append(dirEntry.FileEntries, fileEntry)
	}
	return &dirEntry, nil
}

// GetFileResource implements IPlugin.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	slog.Debug("GetFileResource", "req", req)
	fsGetReq := &FsGetReq{
		Path: req.FilePath,
	}
	content := &Content{}
	err := p.request(http.MethodPost, "/api/fs/get", fsGetReq, content)
	if err != nil {
		return nil, err
	}
	return &plugin.FileResource{
		FileResourceData: []*plugin.FileResource_FileResourceData{
			{
				Url:          content.RawUrl,
				ResourceType: plugin.FileResource_Video,
				Resolution:   plugin.FileResource_Original,
			},
		},
	}, nil
}
