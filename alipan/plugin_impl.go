//go:build wasip1

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"plugins/util"
	"strings"
	"sync"
	"time"

	_ "github.com/labulakalia/wazero_net/wasi/http"
	"github.com/medianexapp/plugin_api/plugin"
	"github.com/medianexapp/plugin_api/ratelimit"
)

/*
NOTE: net and http use package
"github.com/labulakalia/wazero_net/wasi/http"
"github.com/labulakalia/wazero_net/wasi/net"
*/

type PluginImpl struct {
	oauthServerURL string

	token                 *plugin.Token
	userInfo              *UserInfoResponse
	getDriverInfoResponse *UserGetDriverInfoResponse

	ratelimit *ratelimit.RateLimit
}

func NewPluginImpl() *PluginImpl {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	limitConfigMap := map[string]ratelimit.LimitConfig{
		"/adrive/v1.0/openFile/list": ratelimit.LimitConfig{
			Limit:    40,
			Duration: 10 * time.Second,
		},
		"/adrive/v1.0/openFile/getDownloadUrl": ratelimit.LimitConfig{
			Limit:    1,
			Duration: time.Second,
		},
	}
	return &PluginImpl{
		ratelimit: ratelimit.New(limitConfigMap),
	}
}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "alipan", nil
}

func (p *PluginImpl) send(method string, uri string, req, resp any) error {
	_ = p.ratelimit.Wait(uri)
	var body io.Reader
	if req != nil {
		data, err := json.Marshal(req)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequest(method, fmt.Sprintf("%s%s", AlipanURL, uri), body)
	if err != nil {
		return err
	}
	httpReq.Header.Add("Content-Type", "application/json")
	if p.token != nil {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.token.AccessToken))
	}
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	bodyData, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}
	slog.Debug("get request resp", "uri", uri, "req", req, "status_code", httpResp.StatusCode, "resp", string(bodyData))
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		errResp := &ErrResponse{}
		err = json.Unmarshal(bodyData, errResp)
		if err != nil {
			return err
		}
		return errResp
	}
	err = json.Unmarshal(bodyData, resp)
	if err != nil {
		return err
	}
	return nil
}

func (p *PluginImpl) getQrcode() (*plugin.AuthMethod_Scanqrcode, error) {
	qrResp := &QrcodeResponse{}
	qrBytes, err := util.GetAuthQrcode("alipan")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(qrBytes, qrResp)
	if err != nil {
		return nil, err
	}

	return &plugin.AuthMethod_Scanqrcode{
		Scanqrcode: &plugin.Scanqrcode{
			QrcodeImageParam: qrResp.Sid,
			QrcodeExpireTime: uint64(time.Now().Add(time.Minute * 3).Unix()),
			QrcodeImageUrl:   qrResp.QrCodeUrl,
		},
	}, nil
}

// GetAuthe implements IPlugin.
func (p *PluginImpl) GetAuth() (*plugin.Auth, error) {
	auth := &plugin.Auth{
		AuthMethods: []*plugin.AuthMethod{},
	}

	url := util.GetAuthAddr("alipan")
	authCallbackUrl := &plugin.AuthMethod_Callback{
		Callback: &plugin.Callback{
			CallbackUrl: url,
		},
	}
	auth.AuthMethods = append(auth.AuthMethods, &plugin.AuthMethod{
		Method: authCallbackUrl,
	})

	authScanQrcode, err := p.getQrcode()
	if err != nil {
		slog.Error("get qrcode failed", "err", err)
		return nil, err
	}

	auth.AuthMethods = append(auth.AuthMethods, &plugin.AuthMethod{
		Method: authScanQrcode,
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
		token, err = util.CheckAuthQrcode("alipan", v.Scanqrcode.QrcodeImageParam)
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
			Id:           "alipan",
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

// InitAuth implements IPlugin.
func (p *PluginImpl) CheckAuthData(authDataBytes []byte) error {
	token := &plugin.Token{}
	err := token.UnmarshalVT(authDataBytes)
	if err != nil {
		return err
	}
	p.token = token
	resp := &UserInfoResponse{}
	err = p.send(http.MethodGet, "/oauth/users/info", nil, resp)
	if err != nil {
		return err
	}
	p.userInfo = resp
	p.getDriverInfoResponse = &UserGetDriverInfoResponse{}
	err = p.send(http.MethodPost, "/adrive/v1.0/user/getDriveInfo", nil, p.getDriverInfoResponse)
	if err != nil {
		return err
	}

	return nil
}

// AuthId implements IPlugin.
func (p *PluginImpl) PluginAuthId() (string, error) {
	if p.userInfo == nil {
		return "", fmt.Errorf("can get user info")
	}
	return p.userInfo.Id, nil
}

func (d *PluginImpl) getDriverPath(path string) (string, string, error) {
	path = filepath.Clean(path)
	var driverId string
	if strings.HasPrefix(path, "/资源库") {
		path, _ = strings.CutPrefix(path, "/资源库")
		driverId = d.getDriverInfoResponse.ResourceDriverId
	} else if strings.HasPrefix(path, "/备份盘") {
		path, _ = strings.CutPrefix(path, "/备份盘")
		driverId = d.getDriverInfoResponse.BackupDriverId
	} else {
		slog.Error("valid path", "path", path)
		return "", "", os.ErrNotExist
	}
	if path == "" {
		path = "/"
	}
	return driverId, path, nil
}

var (
	pageMarker sync.Map
)

// GetDirEntry implements IPlugin.
func (p *PluginImpl) GetDirEntry(req *plugin.GetDirEntryRequest) (*plugin.DirEntry, error) {
	dirEntry := &plugin.DirEntry{
		FileEntries: []*plugin.FileEntry{},
		PageSize:    100,
	}
	if req.PageSize == 0 {
		req.PageSize = dirEntry.PageSize
	}
	if req.Path == "/" {
		if p.getDriverInfoResponse.ResourceDriverId != "" {
			dirEntry.FileEntries = append(dirEntry.FileEntries, &plugin.FileEntry{
				Name:     "资源库",
				FileType: plugin.FileEntry_FileTypeDir,
			})
		}
		if p.getDriverInfoResponse.BackupDriverId != "" {
			dirEntry.FileEntries = append(dirEntry.FileEntries, &plugin.FileEntry{
				Name:     "备份盘",
				FileType: plugin.FileEntry_FileTypeDir,
			})
		}
		return dirEntry, nil
	}
	driverId, path, err := p.getDriverPath(req.Path)
	if err != nil {
		slog.Error("getDriver failed", "err", err)
		return nil, err
	}
	parentFileId := ""
	if path == "/" {
		parentFileId = "root"
	} else {
		fileEntry := &FileEntry{}
		if req.FileEntry != nil && req.FileEntry.RawData != nil {
			err = json.Unmarshal(req.FileEntry.RawData, fileEntry)
			if err != nil {
				slog.Error("unmarshal failed", "rawData", string(req.FileEntry.RawData), "err", err)
				return nil, err
			}
			parentFileId = fileEntry.FileId
		}
		if parentFileId == "" {
			fileEntry, err = p.getFileEntryInfoByPath(driverId, path)
			if err != nil {
				return nil, err
			}

			parentFileId = fileEntry.FileId
		}
		if parentFileId == "" {
			return nil, errors.New("parent file id is empty")
		}
		parentFileId = fileEntry.FileId
	}
	if req.Page > 1 {
		// page is empty
		return dirEntry, nil
	}

	openFileReq := &OpenFileListRequest{
		DriveId:      driverId,
		Limit:        int(req.PageSize),
		OrderBy:      "name_enhanced",
		ParentFileId: parentFileId,
		Category:     "",
		Marker:       req.DirPageKey,
	}

	openFileRsp := &OpenFileListResponse{}
	err = p.send(http.MethodPost, "/adrive/v1.0/openFile/list", openFileReq, openFileRsp)
	if err != nil {
		return nil, err
	}
	for _, item := range openFileRsp.Items {
		fileType := plugin.FileEntry_FileTypeFile
		if item.Type == "folder" {
			fileType = plugin.FileEntry_FileTypeDir
		}
		fileEntry := &plugin.FileEntry{
			Name:         item.Name,
			FileType:     fileType,
			Size:         item.Size,
			CreatedTime:  uint64(item.CreatedTime.Unix()),
			ModifiedTime: uint64(item.UpdatedTime.Unix()),
			AccessedTime: uint64(item.UpdatedTime.Unix()),
		}
		itemBytes, err := json.Marshal(item)
		if err == nil {
			fileEntry.RawData = itemBytes
		}
		dirEntry.FileEntries = append(dirEntry.FileEntries, fileEntry)
	}
	dirEntry.DirPageKey = openFileRsp.NextMarker
	return dirEntry, nil
}

func (p *PluginImpl) getFileEntryInfoByPath(driverId, path string) (*FileEntry, error) {

	rsp := &OpenFilegetbypathResponse{
		FileEntry: &FileEntry{},
	}
	req := &OpenFilegetbypathRequest{
		DriveId:  driverId,
		FilePath: path,
	}

	err := p.send(http.MethodPost, "/adrive/v1.0/openFile/get_by_path", req, rsp)
	if err != nil {
		return nil, err
	}
	slog.Info("getFileEntryInfoByPath", "resp", rsp)
	return rsp.FileEntry, nil
}

// GetFileResource implements IPlugin.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	driverId, path, err := p.getDriverPath(req.FilePath)
	if err != nil {
		return nil, err
	}
	var fileId string
	if req.FileEntry != nil && req.FileEntry.RawData != nil {
		fileEntry := FileEntry{}
		err = json.Unmarshal(req.FileEntry.RawData, &fileEntry)
		if err != nil {
			return nil, err
		}
		fileId = fileEntry.FileId
	}
	if fileId == "" {
		fileEntry, err := p.getFileEntryInfoByPath(driverId, path)
		if err != nil {
			return nil, err
		}
		fileId = fileEntry.FileId
	}

	fileResource := &plugin.FileResource{
		FileResourceData: []*plugin.FileResource_FileResourceData{},
	}

	getFileDownloadReq := &OpenFilegetDownloadUrlRequest{
		DriveId:   driverId,
		FileId:    fileId,
		ExpireSec: 900,
	}
	getFileDownloadResp := &OpenFilegetDownloadUrlResponse{}
	err = p.send(http.MethodPost, "/adrive/v1.0/openFile/getDownloadUrl", getFileDownloadReq, getFileDownloadResp)
	if err != nil {
		return nil, err
	}
	expireTime, err := time.Parse("2006-01-02T15:04:05Z", getFileDownloadResp.Expiration)
	if err == nil {
		fileResource.FileResourceData = append(fileResource.FileResourceData, &plugin.FileResource_FileResourceData{
			Url:          getFileDownloadResp.Url,
			ExpireTime:   uint64(expireTime.Unix()),
			Resolution:   plugin.FileResource_Original,
			ResourceType: plugin.FileResource_Video,
		})
	} else {
		slog.Error("get download url failed", "err", err)
	}
	if req.IsMedia {
		playReq := &OpenFileGetVideoPreviewPlayInfoRequest{
			DriveId:         driverId,
			FileId:          fileId,
			Category:        "live_transcoding",
			GetSubtitleInfo: true,
			TemplateId:      "LD|SD|HD|FHD|QHD",
			UrlExpireSec:    4 * 60 * 50,
		}
		playRsp := &OpenFileGetVideoPreviewPlayInfoResponse{
			VideoPreViewPlayInfo: &VideoPreViewPlayInfo{
				LiveTranscodingTaskList:         []*LiveTranscodingTask{},
				LiveTranscodingSubtitleTaskList: []*LiveTranscodingSubtitleTask{},
			},
		}
		err = p.send(http.MethodPost, "/adrive/v1.0/openFile/getVideoPreviewPlayInfo", playReq, playRsp)
		if err == nil {
			for _, task := range playRsp.VideoPreViewPlayInfo.LiveTranscodingTaskList {
				if task.Status != "finished" {
					continue
				}
				var resolution plugin.FileResource_Resolution
				switch task.TemplateId {
				case "LD":
					resolution = plugin.FileResource_LD
				case "SD":
					resolution = plugin.FileResource_SD
				case "HD":
					resolution = plugin.FileResource_HD
				case "FHD":
					resolution = plugin.FileResource_FHD
				case "QHD":
					resolution = plugin.FileResource_QHD
				default:
					continue
				}
				if task.Url == "" {
					continue
				}
				fileResource.FileResourceData = append(fileResource.FileResourceData, &plugin.FileResource_FileResourceData{
					Url:          task.Url,
					ExpireTime:   uint64(time.Now().Add(time.Second * 4 * 60 * 50).Unix()),
					ResourceType: plugin.FileResource_Video,
					Resolution:   resolution,
				})
			}
			for _, task := range playRsp.VideoPreViewPlayInfo.LiveTranscodingSubtitleTaskList {
				if task.Status != "finished" {
					continue
				}
				fileResource.FileResourceData = append(fileResource.FileResourceData, &plugin.FileResource_FileResourceData{
					Url:          task.Url,
					Title:        task.Language,
					ResourceType: plugin.FileResource_Subtitle,
				})
			}
		} else {
			slog.Error("get video preview play info failed", "err", err)
		}
	}

	return fileResource, nil
}
