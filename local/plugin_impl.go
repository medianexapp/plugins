//go:build wasip1

package main

import (
	"crypto/md5"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/labulakalia/wazero_net/util"
	"github.com/medianexapp/plugin_api/plugin"
)

/*
NOTE: net and http use package
"github.com/labulakalia/wazero_net/wasi/http"
"github.com/labulakalia/wazero_net/wasi/net"
*/

type PluginImpl struct {
	localpath *plugin.Formdata_FormItem_DirPathValue
	uPath     string
}

func NewPluginImpl() *PluginImpl {
	return &PluginImpl{
		localpath: plugin.DirPath(""),
	}
}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "local", nil
}

// GetAuthType implements IPlugin.
func (p *PluginImpl) GetAuth() (*plugin.Auth, error) {

	formData := &plugin.AuthMethod_Formdata{
		Formdata: &plugin.Formdata{
			FormItems: []*plugin.Formdata_FormItem{
				{
					Name:  "Directory",
					Value: p.localpath,
				},
			},
		},
	}

	return &plugin.Auth{
		AuthMethods: []*plugin.AuthMethod{&plugin.AuthMethod{Method: formData}},
	}, nil
}

// CheckAuth implements IPlugin.
func (p *PluginImpl) CheckAuthMethod(authMethod *plugin.AuthMethod) (authData *plugin.AuthData, err error) {
	authDataBytes, err := authMethod.MarshalVT()
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
	authMethod := &plugin.AuthMethod{}
	err := authMethod.UnmarshalVT(authData)
	if err != nil {
		return err
	}
	fmt.Println("check auth data", authMethod.Method.(*plugin.AuthMethod_Formdata))
	dirPath := authMethod.Method.(*plugin.AuthMethod_Formdata).Formdata.FormItems[0].Value.(*plugin.Formdata_FormItem_DirPathValue)
	p.localpath.DirPathValue = dirPath.DirPathValue
	p.uPath = dirPath.DirPathValue.Value
	if strings.Contains(p.uPath, ":") {
		p.uPath = `/` + strings.ReplaceAll(strings.ReplaceAll(dirPath.DirPathValue.Value, ":", ""), `\`, "/")
	}
	_, err = os.Stat(p.uPath)
	if err != nil {
		return err
	}
	return nil
}

// AuthId implements IPlugin.
func (p *PluginImpl) PluginAuthId() (string, error) {
	return fmt.Sprintf("%x", md5.Sum(util.StringToBytes(&p.localpath.DirPathValue.Value))), nil
}

// GetDirEntry implements IPlugin.
func (p *PluginImpl) GetDirEntry(req *plugin.GetDirEntryRequest) (*plugin.DirEntry, error) {
	dirPath := req.Path
	page := req.Page
	pageSize := req.PageSize

	entries, err := os.ReadDir(filepath.Join(p.uPath, dirPath))
	if err != nil {
		return nil, err
	}
	dirEntry := &plugin.DirEntry{
		FileEntries: []*plugin.FileEntry{},
	}
	start := int((page - 1) * pageSize)
	end := start + int(pageSize)

	if len(entries) <= start {
		return dirEntry, nil
	} else if len(entries) >= end {
		entries = entries[start:end]
	} else {
		entries = entries[start:]
	}
	for _, entry := range entries {
		fileEntry := &plugin.FileEntry{
			Name: entry.Name(),
		}
		if entry.IsDir() {
			fileEntry.FileType = plugin.FileEntry_FileTypeDir
		} else {
			fileEntry.FileType = plugin.FileEntry_FileTypeFile
		}
		fileInfo, err := entry.Info()
		if err != nil {
			slog.Error("Failed to get file info", "error", err)
			continue
		}
		fileEntry.Size = uint64(fileInfo.Size())
		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if ok {
			fileEntry.AccessedTime = uint64(stat.Atim.Sec)
			fileEntry.ModifiedTime = uint64(stat.Mtim.Sec)
			fileEntry.CreatedTime = uint64(stat.Ctim.Sec)
		}
		dirEntry.FileEntries = append(dirEntry.FileEntries, fileEntry)
	}
	return dirEntry, nil
}

// GetFileResource implements IPlugin.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	statPath := filepath.Join(p.uPath, req.FilePath)
	if strings.Contains(statPath, `\`) {
		statPath = strings.ReplaceAll(statPath, `\`, `/`)
	}
	_, err := os.Stat(statPath)
	if err != nil {
		return nil, err
	}
	return &plugin.FileResource{
		FileResourceData: []*plugin.FileResource_FileResourceData{
			{
				Url:          fmt.Sprintf("file://%s", filepath.Join(p.localpath.DirPathValue.Value, req.FilePath)),
				ResourceType: plugin.FileResource_Video,
				Resolution:   plugin.FileResource_Original,
			},
		},
	}, nil
}
