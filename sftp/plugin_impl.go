//go:build wasip1

package main

import (
	"crypto/md5"
	"fmt"
	"log/slog"
	"net/netip"
	"strings"
	"time"

	"github.com/medianexapp/plugin_api/plugin"
	"github.com/medianexapp/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/labulakalia/wazero_net/util"
	wasi_net "github.com/labulakalia/wazero_net/wasi/net"
)

/*
NOTE: net and http use package
"github.com/labulakalia/wazero_net/wasi/http"
"github.com/labulakalia/wazero_net/wasi/net"
*/

type PluginImpl struct {
	sftpClient *sftp.Client

	sftpAuth *sftpAuth
}

func NewPluginImpl() *PluginImpl {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	return &PluginImpl{
		sftpAuth: &sftpAuth{
			Addr:     plugin.String("127.0.0.1"),
			User:     plugin.String(""),
			Password: plugin.ObscureString(""),
		},
	}
}

type sftpAuth struct {
	Addr     *plugin.Formdata_FormItem_StringValue
	User     *plugin.Formdata_FormItem_StringValue
	Password *plugin.Formdata_FormItem_ObscureStringValue
}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "sftp", nil
}

// GetAuthType implements IPlugin.
func (p *PluginImpl) GetAuth() (*plugin.Auth, error) {
	formData := &plugin.AuthMethod_Formdata{
		Formdata: &plugin.Formdata{
			FormItems: []*plugin.Formdata_FormItem{
				{
					Name:  "Addr",
					Value: p.sftpAuth.Addr,
				},
				{
					Name:  "User",
					Value: p.sftpAuth.User,
				},
				{
					Name:  "Password",
					Value: p.sftpAuth.Password,
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

func (p *PluginImpl) unmarshalFormData(formData *plugin.Formdata) {
	p.sftpAuth.Addr.StringValue = formData.FormItems[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue
	p.sftpAuth.User.StringValue = formData.FormItems[1].Value.(*plugin.Formdata_FormItem_StringValue).StringValue
	p.sftpAuth.Password.ObscureStringValue = formData.FormItems[2].Value.(*plugin.Formdata_FormItem_ObscureStringValue).ObscureStringValue

}

// InitAuth implements IPlugin.
func (p *PluginImpl) CheckAuthData(AuthDataBytes []byte) error {
	slog.Debug("start check auth data")
	authMethod := &plugin.AuthMethod{}
	err := authMethod.UnmarshalVT(AuthDataBytes)
	if err != nil {
		return err
	}
	formData := authMethod.Method.(*plugin.AuthMethod_Formdata)
	p.unmarshalFormData(formData.Formdata)
	addr := p.sftpAuth.Addr.StringValue.Value
	_, err = netip.ParseAddrPort(addr)
	if err != nil {
		addr = fmt.Sprintf("%s:%d", strings.TrimRight(addr, ":"), 22)
	}

	conn, err := wasi_net.Dial("tcp", addr)
	if err != nil {
		slog.Error("dial failed", "err", err)
		return err
	}
	config := &ssh.ClientConfig{
		User: p.sftpAuth.User.StringValue.Value,
		Auth: []ssh.AuthMethod{
			ssh.Password(p.sftpAuth.Password.ObscureStringValue.Value),
			// ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second,
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		slog.Error("client conn failed", "err", err)
		return err
	}
	sftpClient, err := sftp.NewClient(ssh.NewClient(c, chans, reqs))
	if err != nil {
		slog.Error("sftp client failed", "err", err)
		return err
	}
	p.sftpClient = sftpClient
	slog.Debug("check auth data success")
	return nil
}

// AuthId implements IPlugin.
func (p *PluginImpl) PluginAuthId() (string, error) {
	id := fmt.Sprintf("%s%s%s", p.sftpAuth.Addr.StringValue.Value, p.sftpAuth.User.StringValue.Value, p.sftpAuth.Password.ObscureStringValue.Value)
	return fmt.Sprintf("%x", md5.Sum(util.StringToBytes(&id))), nil
}

// GetDirEntry implements IPlugin.
func (p *PluginImpl) GetDirEntry(req *plugin.GetDirEntryRequest) (*plugin.DirEntry, error) {
	dirPath := req.Path
	page := req.Page
	pageSize := req.PageSize
	entries, err := p.sftpClient.ReadDir(dirPath)
	if err != nil {
		slog.Error("sftp client failed", "err", err)
		return nil, err
	}
	dirEntry := &plugin.DirEntry{
		FileEntries: make([]*plugin.FileEntry, 0, len(entries)),
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
			Size: uint64(entry.Size()),
			Name: entry.Name(),
		}
		if entry.IsDir() {
			fileEntry.FileType = plugin.FileEntry_FileTypeDir
		} else {
			fileEntry.FileType = plugin.FileEntry_FileTypeFile
		}
		stat, ok := entry.Sys().(*sftp.FileStat)

		if ok {
			fileEntry.CreatedTime = uint64(stat.Mtime)
			fileEntry.ModifiedTime = uint64(stat.Mode)
			fileEntry.AccessedTime = uint64(stat.Atime)
		}
		dirEntry.FileEntries = append(dirEntry.FileEntries, fileEntry)
	}
	return dirEntry, nil
}

// GetFileResource implements IPlugin.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	// sftp://[user[:password]@]server[:port]/path/to/remote/resource.mpeg
	_, err := p.sftpClient.Stat(req.FilePath)
	if err != nil {
		return nil, err
	}
	userPass := ""
	if p.sftpAuth.User.StringValue.Value != "" && p.sftpAuth.Password.ObscureStringValue.Value != "" {
		userPass = fmt.Sprintf("%s:%s@", p.sftpAuth.User.StringValue.Value, p.sftpAuth.Password.ObscureStringValue.Value)
	}
	fileUrl := fmt.Sprintf("sftp://%s%s%s", userPass, p.sftpAuth.Addr.StringValue.Value, req.FileEntry)
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
