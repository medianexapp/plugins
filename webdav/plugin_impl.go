package main

import (
	"crypto/md5"
	"fmt"
	"log/slog"

	"github.com/labulakalia/wazero_net/util"
	_ "github.com/labulakalia/wazero_net/wasi/http"
	"github.com/medianexapp/gowebdav"
	"github.com/medianexapp/plugin_api/httpclient"
	"github.com/medianexapp/plugin_api/plugin"
)

type PluginImpl struct {
	webDavAuth *webDavAuth

	client     *gowebdav.Client
	httpclient *httpclient.Client
}

func NewPluginImpl() *PluginImpl {

	return &PluginImpl{
		webDavAuth: &webDavAuth{
			Addr:     plugin.String("http://127.0.0.1"),
			User:     plugin.String(""),
			Password: plugin.ObscureString(""),
		},
		httpclient: httpclient.NewClient(),
	}
}

type webDavAuth struct {
	Addr     *plugin.Formdata_FormItem_StringValue
	User     *plugin.Formdata_FormItem_StringValue
	Password *plugin.Formdata_FormItem_ObscureStringValue
}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "webdav", nil
}

// GetAuthType implements IPlugin.
func (p *PluginImpl) GetAuth() (*plugin.Auth, error) {
	addrValue := p.webDavAuth.Addr
	userValue := p.webDavAuth.User
	passwordValue := p.webDavAuth.Password

	authMethod := &plugin.AuthMethod{
		Method: &plugin.AuthMethod_Formdata{
			Formdata: &plugin.Formdata{
				FormItems: []*plugin.Formdata_FormItem{
					{
						Name:  "Addr",
						Value: addrValue,
					},
					{
						Name:  "User",
						Value: userValue,
					},
					{
						Name:  "Password",
						Value: passwordValue,
					},
				},
			},
		},
	}

	return &plugin.Auth{
		AuthMethods: []*plugin.AuthMethod{authMethod},
	}, nil
}

// CheckAuth implements IPlugin.
func (p *PluginImpl) CheckAuthMethod(authMethod *plugin.AuthMethod) (authData *plugin.AuthData, err error) {
	formData := authMethod.Method.(*plugin.AuthMethod_Formdata).Formdata
	authDataBytes, err := formData.MarshalVT()
	if err != nil {
		return nil, err
	}
	authData = &plugin.AuthData{
		AuthDataBytes: authDataBytes,
	}
	return authData, nil
}

// InitAuth implements IPlugin.
func (p *PluginImpl) CheckAuthData(authData []byte) error {
	formData := &plugin.Formdata{}
	err := formData.UnmarshalVT(authData)
	if err != nil {
		return err
	}
	p.webDavAuth.Addr.StringValue.Value = formData.FormItems[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value
	p.webDavAuth.User.StringValue.Value = formData.FormItems[1].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value
	p.webDavAuth.Password.ObscureStringValue.Value = formData.FormItems[2].Value.(*plugin.Formdata_FormItem_ObscureStringValue).ObscureStringValue.Value

	slog.Debug("webdav connect", "addr", p.webDavAuth.Addr.StringValue.Value, "user", p.webDavAuth.User.StringValue.Value, "passwd", p.webDavAuth.Password.ObscureStringValue.Value)
	p.client = gowebdav.NewClient(p.webDavAuth.Addr.StringValue.Value, p.webDavAuth.User.StringValue.Value, p.webDavAuth.Password.ObscureStringValue.Value)
	p.client.SetClientDo(p.httpclient.Do)
	err = p.client.Connect()
	if err != nil {
		slog.Error("connect failed", "err", err)
		return err
	}
	_, err = p.client.Stat("/")
	return err
}

// AuthId implements IPlugin.
func (p *PluginImpl) PluginAuthId() (string, error) {
	id := fmt.Sprintf("%s%s%s", p.webDavAuth.Addr.StringValue.Value, p.webDavAuth.User.StringValue.Value, p.webDavAuth.Password.ObscureStringValue.Value)
	return fmt.Sprintf("%x", md5.Sum(util.StringToBytes(&id))), nil
}

// GetDirEntry implements IPlugin.
func (p *PluginImpl) GetDirEntry(req *plugin.GetDirEntryRequest) (*plugin.DirEntry, error) {
	dirPath := req.Path
	page := req.Page
	pageSize := req.PageSize
	fileInfos, err := p.client.ReadDir(dirPath)
	if err != nil {
		slog.Error("read dir failed", "err", err, "dir", dirPath, "fileInfos", fileInfos)
		return nil, err
	}
	dirEntry := &plugin.DirEntry{
		FileEntries: make([]*plugin.FileEntry, 0, len(fileInfos)),
	}

	start := int((page - 1) * pageSize)
	end := start + int(pageSize)

	if len(fileInfos) <= start {
		return dirEntry, nil
	} else if len(fileInfos) >= end {
		fileInfos = fileInfos[start:end]
	} else {
		fileInfos = fileInfos[start:]
	}
	for _, fileinfo := range fileInfos {
		fileinfo.Sys()
		fileEntry := &plugin.FileEntry{
			Name:         fileinfo.Name(),
			Size:         uint64(fileinfo.Size()),
			ModifiedTime: uint64(fileinfo.ModTime().Unix()),
			AccessedTime: uint64(fileinfo.ModTime().Unix()),
			CreatedTime:  uint64(fileinfo.ModTime().Unix()),
		}
		if fileinfo.IsDir() {
			fileEntry.FileType = plugin.FileEntry_FileTypeDir
		} else {
			fileEntry.FileType = plugin.FileEntry_FileTypeFile
		}

		dirEntry.FileEntries = append(dirEntry.FileEntries, fileEntry)
	}
	return dirEntry, nil
}

// GetFileResource implements IPlugin.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	_, err := p.client.Stat(req.FilePath)
	if err != nil {
		return nil, err
	}
	pathReq, err := p.client.GetPathRequest(req.FilePath)
	if err != nil {
		return nil, err
	}
	url := pathReq.URL.String()
	header := map[string]string{}

	for k := range pathReq.Header {
		header[k] = pathReq.Header.Get(k)
	}
	return &plugin.FileResource{
		FileResourceData: []*plugin.FileResource_FileResourceData{
			{
				Url:          url,
				Header:       header,
				ResourceType: plugin.FileResource_Video,
				Resolution:   plugin.FileResource_Original,
			},
		},
	}, nil
}
