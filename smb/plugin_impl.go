package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strings"

	"github.com/labulakalia/wazero_net/util"
	wasi_net "github.com/labulakalia/wazero_net/wasi/net"
	"github.com/medianexapp/go-smb2"
	"github.com/medianexapp/plugin_api/plugin"
)

/*
NOTE: net and http use package
"github.com/labulakalia/wazero_net/wasi/http"
"github.com/labulakalia/wazero_net/wasi/net"
*/

type PluginImpl struct {
	session *smb2.Session
	shares  map[string]*smb2.Share

	sambaAuth *sambaAuth
}

func NewPluginImpl() *PluginImpl {
	return &PluginImpl{
		sambaAuth: &sambaAuth{
			Addr:     plugin.String("127.0.0.1"),
			User:     plugin.String(""),
			Password: plugin.ObscureString(""),
		},
	}

}

type sambaAuth struct {
	Addr     *plugin.Formdata_FormItem_StringValue
	User     *plugin.Formdata_FormItem_StringValue
	Password *plugin.Formdata_FormItem_ObscureStringValue
}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "smb", nil
}

// GetAuth implements IPlugin.
func (p *PluginImpl) GetAuth() (*plugin.Auth, error) {
	formData := &plugin.AuthMethod_Formdata{
		Formdata: &plugin.Formdata{
			FormItems: []*plugin.Formdata_FormItem{

				{
					Name:  "Addr",
					Value: p.sambaAuth.Addr,
				},
				{
					Name:  "User",
					Value: p.sambaAuth.User,
				},
				{
					Name:  "Password",
					Value: p.sambaAuth.Password,
				},
			},
		},
	}
	return &plugin.Auth{
		AuthMethods: []*plugin.AuthMethod{
			&plugin.AuthMethod{
				Method: formData,
			},
		},
	}, nil
}

func (p *PluginImpl) unmarshalFormData(formData *plugin.Formdata) {
	p.sambaAuth.Addr.StringValue = formData.FormItems[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue
	p.sambaAuth.User.StringValue = formData.FormItems[1].Value.(*plugin.Formdata_FormItem_StringValue).StringValue
	p.sambaAuth.Password.ObscureStringValue = formData.FormItems[2].Value.(*plugin.Formdata_FormItem_ObscureStringValue).ObscureStringValue

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
func (p *PluginImpl) CheckAuthData(authDataBytes []byte) error {
	authMethod := &plugin.AuthMethod{}
	err := authMethod.UnmarshalVT(authDataBytes)
	if err != nil {
		return err
	}
	p.unmarshalFormData(authMethod.Method.(*plugin.AuthMethod_Formdata).Formdata)

	addr := p.sambaAuth.Addr.StringValue.Value
	_, err = netip.ParseAddrPort(addr)
	if err != nil {
		addr = fmt.Sprintf("%s:%d", strings.TrimRight(addr, ":"), 445)
	}
	slog.Info("start dial tcp", "addr", addr)
	conn, err := wasi_net.Dial("tcp", addr)
	if err != nil {
		slog.Error("dial failed", "err", err)
		return err
	}

	user := p.sambaAuth.User.StringValue.Value
	passwd := p.sambaAuth.Password.ObscureStringValue.Value
	if user == "" && passwd == "" {
		user = "guest"
		passwd = "guest"
	}
	smbDialer := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     user,
			Password: passwd,
		},
	}

	smbSession, err := smbDialer.DialConn(context.Background(), conn, addr)
	if err != nil {
		slog.Error("failed to dial", "error", err)
		return err
	}

	p.session = smbSession
	shareNames, err := smbSession.ListSharenames()
	if err != nil {
		return err
	}

	p.shares = make(map[string]*smb2.Share)
	for _, shareName := range shareNames {
		if strings.HasSuffix(shareName, "$") {
			continue
		}
		share, err := smbSession.Mount(shareName)
		if err != nil {
			slog.Error("mount failed", "sharename", shareName, "error", err)
			continue
		}
		p.shares[shareName] = share
		slog.Info("smb session mount", "share name", shareName)
	}

	return nil
}

// AuthId implements IPlugin.
func (p *PluginImpl) PluginAuthId() (string, error) {
	id := fmt.Sprintf("%s%s%s", p.sambaAuth.Addr.StringValue.Value, p.sambaAuth.User.StringValue.Value, p.sambaAuth.Password.ObscureStringValue.Value)
	return fmt.Sprintf("%x", md5.Sum(util.StringToBytes(&id))), nil
}

func (p *PluginImpl) checkShare(dir_path string) (*smb2.Share, string, error) {
	sp := strings.Split(dir_path, "/")[1:]
	shareName := sp[0]
	share, ok := p.shares[shareName]
	if !ok {
		return nil, "", fmt.Errorf("%s not exist", dir_path)
	}
	return share, strings.Join(sp[1:], "/"), nil
}

// GetDirEntry implements IPlugin.
func (p *PluginImpl) GetDirEntry(req *plugin.GetDirEntryRequest) (*plugin.DirEntry, error) {
	dirPath := req.Path
	page := req.Page
	pageSize := req.PageSize
	dirEntry := &plugin.DirEntry{
		FileEntries: []*plugin.FileEntry{},
	}
	if dirPath == "/" {
		for name := range p.shares {
			dirEntry.FileEntries = append(dirEntry.FileEntries, &plugin.FileEntry{
				Name:     name,
				FileType: plugin.FileEntry_FileTypeDir,
			})
		}
		return dirEntry, nil
	}

	share, smbPath, err := p.checkShare(dirPath)
	if err != nil {
		return nil, err
	}

	fileInfos, err := share.ReadDir(smbPath)
	if err != nil {
		return nil, err
	}
	newFileInfos := []os.FileInfo{}
	for _, fileinfo := range fileInfos {
		if strings.HasPrefix(fileinfo.Name(), ".") {
			continue
		}
		newFileInfos = append(newFileInfos, fileinfo)
	}
	fileInfos = newFileInfos
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
		fileEntry := &plugin.FileEntry{
			Name: fileinfo.Name(),
			Size: uint64(fileinfo.Size()),
		}
		if fileinfo.IsDir() {
			fileEntry.FileType = plugin.FileEntry_FileTypeDir
		} else {
			fileEntry.FileType = plugin.FileEntry_FileTypeFile
		}
		fileStat, ok := fileinfo.Sys().(*smb2.FileStat)
		if ok {
			fileEntry.CreatedTime = uint64(fileStat.ChangeTime.Unix())
			fileEntry.ModifiedTime = uint64(fileStat.LastWriteTime.Unix())
			fileEntry.AccessedTime = uint64(fileStat.LastAccessTime.Unix())
		}
		dirEntry.FileEntries = append(dirEntry.FileEntries, fileEntry)
	}
	return dirEntry, nil
}

// GetFileResource implements IPlugin.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	// smb://[[domain:]user[:password@]]server[/share[/path[/file]]]
	share, smbPath, err := p.checkShare(req.FilePath)
	if err != nil {
		return nil, err
	}
	_, err = share.Stat(smbPath)
	if err != nil {
		return nil, err
	}
	userPass := p.sambaAuth.User.StringValue.Value
	if p.sambaAuth.Password.ObscureStringValue.Value != "" {
		userPass = fmt.Sprintf("%s:%s@", p.sambaAuth.User.StringValue.Value, p.sambaAuth.Password.ObscureStringValue.Value)
	}
	fileUrl := fmt.Sprintf("smb://%s%s%s", userPass, p.sambaAuth.Addr.StringValue.Value, req.FilePath)
	return &plugin.FileResource{
		FileResourceData: []*plugin.FileResource_FileResourceData{
			{
				Url:          fileUrl,
				Resolution:   plugin.FileResource_Original,
				ResourceType: plugin.FileResource_Video,
			},
		},
	}, nil
}
