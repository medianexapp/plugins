package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/medianexapp/plugin_api/httpclient"
	"github.com/medianexapp/plugin_api/plugin"
	"github.com/medianexapp/plugin_api/ratelimit"

	_ "github.com/labulakalia/wazero_net/wasi/http" // if you need http import this
)

type PluginImpl struct {
	authData  *AuthToken
	client    *httpclient.Client
	userInfo  *UserInfo
	ratelimit *ratelimit.RateLimit
}

// https://123yunpan.yuque.com/org-wiki-123yunpan-muaork/cr6ced/txgcvbfgh0gtuad5

func NewPluginImpl() *PluginImpl {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	return &PluginImpl{
		client: httpclient.NewClient(),
		authData: &AuthToken{
			ClientId:     &plugin.Formdata_FormItem_StringValue{},
			ClientSecret: &plugin.Formdata_FormItem_StringValue{},
		},
		ratelimit: ratelimit.New(map[string]ratelimit.LimitConfig{
			"api/v1/user/info": {Limit: 1, Duration: time.Second},
		}),
	}
}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "123pan", nil
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
								Name:  "Client Id",
								Value: p.authData.ClientId,
							},
							{
								Name:  "Client Secret",
								Value: p.authData.ClientSecret,
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

	switch v := authMethod.Method.(type) {
	case *plugin.AuthMethod_Refresh:
		accessToken := AuthToken{}
		err := json.Unmarshal(v.Refresh.AuthData.AuthDataBytes, &accessToken)
		if err != nil {
			return nil, err
		}
		p.authData.ClientId = accessToken.ClientId
		p.authData.ClientSecret = accessToken.ClientSecret

	case *plugin.AuthMethod_Formdata:
		formItems := v.Formdata.FormItems
		p.authData.ClientId = formItems[0].Value.(*plugin.Formdata_FormItem_StringValue)
		p.authData.ClientSecret = formItems[1].Value.(*plugin.Formdata_FormItem_StringValue)
	}
	reqData := map[string]string{
		"clientID":     p.authData.ClientId.StringValue.Value,
		"clientSecret": p.authData.ClientSecret.StringValue.Value,
	}
	respData := AuthToken{}
	err := p.sendData(http.MethodPost, "/api/v1/access_token", reqData, &respData)
	if err != nil {
		return nil, err
	}
	tt, err := time.Parse("2006-01-02T15:04:05+07:00", respData.ExpiredAt)
	if err != nil {
		return nil, err
	}
	authBytes, err := json.Marshal(respData)
	if err != nil {
		return nil, err
	}
	return &plugin.AuthData{
		AuthDataBytes:       authBytes,
		AuthDataExpiredTime: uint64(tt.Unix()),
	}, nil
}

// CheckAuthData use authDataBytes to uath
// you must store auth data to *PluginImpl
func (p *PluginImpl) CheckAuthData(authDataBytes []byte) error {
	slog.Debug("CheckAuthData", "authDataBytes", authDataBytes)
	authData := AuthToken{}
	err := json.Unmarshal(authDataBytes, &authData)
	if err != nil {
		return err
	}
	p.authData = &authData

	p.userInfo = &UserInfo{}
	err = p.sendData(http.MethodGet, "/api/v1/user/info", nil, p.userInfo)
	if err != nil {
		slog.Error("get user info failed", "err", err)
		return err
	}

	return nil
}

// PluginAuthId implements IPlugin.
// plugin auth id,you can generate id by md5 or sha
func (p *PluginImpl) PluginAuthId() (string, error) {
	return p.userInfo.UID, nil
}

// GetDirEntry implements IPlugin.
// return dir file entry
// save your driver file raw data to FileEntry.RawData,you can get it after GetDirEntry and GetFileResource request
// default page_size if 100,if this not for you,change is on DirEntry.PageSize,will use new PageSize for next request
func (p *PluginImpl) GetDirEntry(req *plugin.GetDirEntryRequest) (*plugin.DirEntry, error) {
	slog.Debug("GetDirEntry", "req", req.FileEntry)
	if req.PageSize == 0 {
		req.PageSize = 100
	}
	params := map[string]string{
		"limit":      fmt.Sprintf("%d", req.PageSize),
		"lastFileId": req.DirPageKey,
	}
	var parentFileId string
	if req.Path == "/" {
		parentFileId = "0"
	} else {
		fileItem := FileItem{}
		if req.FileEntry == nil {
			return nil, fmt.Errorf("can get file info")
		}
		err := json.Unmarshal(req.FileEntry.RawData, &fileItem)
		if err != nil {
			return nil, err
		}
		parentFileId = fmt.Sprint(fileItem.FileId)
	}
	params["parentFileId"] = parentFileId
	resp := FileListResponse{
		FileList: []FileItem{},
	}
	err := p.sendData(http.MethodGet, "/api/v2/file/list", params, &resp)
	if err != nil {
		return nil, err
	}
	getDirEntryResp := &plugin.DirEntry{
		FileEntries: []*plugin.FileEntry{},
		DirPageKey:  fmt.Sprint(resp.FileList),
	}
	for _, fileItem := range resp.FileList {
		fileEntry := &plugin.FileEntry{
			Name: fileItem.FileName,
			Size: fileItem.Size,
		}
		if fileItem.Type == 0 {
			fileEntry.FileType = plugin.FileEntry_FileTypeFile
		} else {
			fileEntry.FileType = plugin.FileEntry_FileTypeDir
		}
		rawData, err := json.Marshal(fileItem)
		if err != nil {
			return nil, err
		}
		fileEntry.RawData = rawData
		t, err := time.Parse("2006-01-02 15:04:05", fileItem.CreateAt)
		if err == nil {
			fileEntry.CreatedTime = uint64(t.Unix())
		}
		t, err = time.Parse("2006-01-02 15:04:05", fileItem.UpdateAt)
		if err == nil {
			fileEntry.ModifiedTime = uint64(t.Unix())
			fileEntry.AccessedTime = uint64(t.Unix())
		}
		getDirEntryResp.FileEntries = append(getDirEntryResp.FileEntries, fileEntry)
	}

	return getDirEntryResp, nil
}

// GetFileResource implements IPlugin.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	slog.Debug("GetFileResource", "req", req)
	if req.FileEntry == nil {
		return nil, fmt.Errorf("can get file info")
	}
	fileResource := &plugin.FileResource{
		FileResourceData: []*plugin.FileResource_FileResourceData{},
	}
	fileItem := FileItem{}
	err := json.Unmarshal(req.FileEntry.RawData, &fileItem)
	if err != nil {
		return nil, err
	}
	downInfo := &DownloadInfo{}
	err = p.sendData(http.MethodGet, "/api/v1/file/download_info", map[string]string{
		"fileId": fmt.Sprint(fileItem.FileId),
	}, downInfo)
	if err != nil {
		return nil, err
	}
	fileResource.FileResourceData = append(fileResource.FileResourceData,
		&plugin.FileResource_FileResourceData{
			Url:          downInfo.DownloadUrl,
			Resolution:   plugin.FileResource_Original,
			ResourceType: plugin.FileResource_Video,
		},
	)

	playerVideoResp := &PlayerVideoResponse{
		UserTranscodeVideoList: []userTranscodeVideo{},
	}

	type ReqFileId struct {
		FileId uint64 `json:"fileId"`
	}

	err = p.sendData(http.MethodPost, "/api/v1/transcode/video/result", map[string]uint64{
		"fileId": fileItem.FileId,
	}, playerVideoResp)
	if err != nil {
		return nil, err
	}
	for _, video := range playerVideoResp.UserTranscodeVideoList {
		if video.Status != 255 || len(video.Files) == 0 {
			continue
		}
		var res plugin.FileResource_Resolution
		switch video.Resolution {
		case "720P":
			res = plugin.FileResource_HD
		case "1080P":
			res = plugin.FileResource_FHD
		case "2160P":
			res = plugin.FileResource_QHD

		}
		fileResource.FileResourceData = append(fileResource.FileResourceData,
			&plugin.FileResource_FileResourceData{
				Url:          video.Files[0].URL,
				Resolution:   res,
				ResourceType: plugin.FileResource_Video,
			},
		)
	}
	return fileResource, nil

}

func (p *PluginImpl) sendData(method string, uri string, reqData any, respData any) error {
	b := httpclient.NewBuilder().
		Request(fmt.Sprintf("%s%s", PanURl, uri)).
		SetMethod(method)
	if reqData != nil {
		if method == http.MethodGet {
			for k, v := range reqData.(map[string]string) {
				b = b.SetQueryParam(k, v)
			}
		} else {
			b = b.SetBody(reqData)
		}
	}

	b = b.SetHeader("Platform", "open_platform")

	if p.authData != nil && p.authData.AccessToken != "" {
		b = b.SetHeader("Authorization", fmt.Sprintf("Bearer %s", p.authData.AccessToken))
	}

	rsp := &Response{
		Data: respData,
	}
	err := b.JSONResponse(&rsp)
	if err != nil {
		return err
	}
	if rsp.Code != 0 {
		slog.Error("Request Failed", "code", rsp.Code, "message", rsp.Message)
		return errors.New(rsp.Message)
	}
	return nil
}
