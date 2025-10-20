//go:build wasip1

package main

import (
	"crypto/md5"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/labulakalia/wazero_net/util"
	wasi_net "github.com/labulakalia/wazero_net/wasi/net"
	"github.com/medianexapp/ftp"
	"github.com/medianexapp/plugin_api/plugin"
)

/*
NOTE: net and http use package
"github.com/labulakalia/wazero_net/wasi/http"
"github.com/labulakalia/wazero_net/wasi/net"
*/

type PluginImpl struct {
	ftpAuth *ftpAuth
	ftpConn *ftp.ServerConn
}

func NewPluginImpl() *PluginImpl {
	ftpAuth := &ftpAuth{
		Addr:     plugin.String("127.0.0.1:21"),
		User:     plugin.String(""),
		Password: plugin.ObscureString(""),
	}
	return &PluginImpl{
		ftpAuth: ftpAuth,
	}
}

type ftpAuth struct {
	Addr     *plugin.Formdata_FormItem_StringValue
	User     *plugin.Formdata_FormItem_StringValue
	Password *plugin.Formdata_FormItem_ObscureStringValue
}

// Id implements IPlugin.
func (p *PluginImpl) PluginId() (string, error) {
	return "ftp", nil
}

// GetAuthType implements IPlugin.
func (p *PluginImpl) GetAuth() (*plugin.Auth, error) {

	formData := &plugin.AuthMethod_Formdata{
		Formdata: &plugin.Formdata{
			FormItems: []*plugin.Formdata_FormItem{
				{
					Name:  "Addr",
					Value: p.ftpAuth.Addr,
				},
				{
					Name:  "User",
					Value: p.ftpAuth.User,
				},
				{
					Name:  "Password",
					Value: p.ftpAuth.Password,
				},
			},
		},
	}
	auth := &plugin.Auth{
		AuthMethods: []*plugin.AuthMethod{&plugin.AuthMethod{Method: formData}},
	}
	return auth, nil
}

// CheckAuthMethod implements IPlugin.
func (p *PluginImpl) CheckAuthMethod(authMethod *plugin.AuthMethod) (authData *plugin.AuthData, err error) {
	// todo ftp over tls

	formDataBytes, err := authMethod.MarshalVT()
	if err != nil {
		return nil, err
	}
	return &plugin.AuthData{
		AuthDataBytes: formDataBytes,
	}, nil
}

type ConnWrap struct {
	net.Conn
	readTimeout time.Duration
}

func (c *ConnWrap) Read(b []byte) (n int, err error) {
	err = c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	if err != nil {
		return 0, err
	}

	return c.Conn.Read(b)
}

func NewWrapConn(conn net.Conn, readTimeout time.Duration) *ConnWrap {
	return &ConnWrap{
		readTimeout: readTimeout,
		Conn:        conn,
	}
}

func (p *PluginImpl) unmarshalFormData(formData *plugin.Formdata) {
	p.ftpAuth.Addr.StringValue = formData.FormItems[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue
	p.ftpAuth.User.StringValue = formData.FormItems[1].Value.(*plugin.Formdata_FormItem_StringValue).StringValue
	p.ftpAuth.Password.ObscureStringValue = formData.FormItems[2].Value.(*plugin.Formdata_FormItem_ObscureStringValue).ObscureStringValue

}

func (p *PluginImpl) connectFtp() error {
	if p.ftpConn != nil {
		p.ftpConn.Logout()
	}
	addr := p.ftpAuth.Addr.StringValue.Value
	_, err := netip.ParseAddrPort(addr)
	if err != nil {
		addr = fmt.Sprintf("%s:%d", strings.TrimRight(addr, ":"), 21)
	}

	user := p.ftpAuth.User.StringValue.Value
	password := p.ftpAuth.Password.ObscureStringValue.Value
	if user == "" && password == "" {
		password = "anonymous"
		user = "anonymous"
	}

	ftpConn, err := ftp.Dial(addr, ftp.DialWithDialFunc(func(network, address string) (net.Conn, error) {
		conn, err := wasi_net.Dial(network, address)
		if err != nil {
			slog.Error("dial failed", "err", err)
			return nil, err
		}
		return NewWrapConn(conn, time.Second*5), nil
	}))
	if err != nil {
		return err
	}
	err = ftpConn.Login(user, password)
	if err != nil {
		slog.Error("ftp login failed", "addr", addr)
		return err
	}
	p.ftpConn = ftpConn
	return err
}

// InitAuth implements IPlugin.
func (p *PluginImpl) CheckAuthData(AuthDataBytes []byte) error {
	authMethod := &plugin.AuthMethod{}
	err := authMethod.UnmarshalVT(AuthDataBytes)
	if err != nil {
		return err
	}
	formData := authMethod.Method.(*plugin.AuthMethod_Formdata)
	p.unmarshalFormData(formData.Formdata)
	err = p.connectFtp()
	if err != nil {
		return err
	}
	slog.Info("ftp login success", "addr", p.ftpAuth.Addr.StringValue.Value)
	return nil
}

// AuthId implements IPlugin.
func (p *PluginImpl) PluginAuthId() (string, error) {
	id := fmt.Sprintf("%s%s%s", p.ftpAuth.Addr.StringValue.Value, p.ftpAuth.User.StringValue.Value, p.ftpAuth.Password.ObscureStringValue.Value)
	return fmt.Sprintf("%x", md5.Sum(util.StringToBytes(&id))), nil
}

// GetDirEntry implements IPlugin.
func (p *PluginImpl) GetDirEntry(req *plugin.GetDirEntryRequest) (*plugin.DirEntry, error) {
	dirPath := req.Path
	page := req.Page
	pageSize := req.PageSize

	var (
		entries []*ftp.Entry
		err     error
	)
	for range 3 {
		entries, err = p.ftpConn.List(dirPath)
		if err != nil {
			slog.Error("list failed", "err", err)
			err = p.connectFtp()
			if err != nil {
				slog.Error("reconnect ftp failed", "err", err)
			}
		} else {
			break
		}
	}

	dirEntry := &plugin.DirEntry{
		FileEntries: make([]*plugin.FileEntry, 0, pageSize),
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
		if strings.HasPrefix(entry.Name, ".") {
			continue
		}
		fileEntry := &plugin.FileEntry{
			Name:         entry.Name,
			Size:         entry.Size,
			CreatedTime:  uint64(entry.Time.UnixMilli()),
			ModifiedTime: uint64(entry.Time.UnixMilli()),
			AccessedTime: uint64(entry.Time.UnixMilli()),
		}

		if entry.Type == ftp.EntryTypeFile {
			fileEntry.FileType = plugin.FileEntry_FileTypeFile
		} else if entry.Type == ftp.EntryTypeFolder {
			fileEntry.FileType = plugin.FileEntry_FileTypeDir
		}
		dirEntry.FileEntries = append(dirEntry.FileEntries, fileEntry)
	}
	return dirEntry, nil
}

// GetFileResource implements IPlugin.
func (p *PluginImpl) GetFileResource(req *plugin.GetFileResourceRequest) (*plugin.FileResource, error) {
	// url path ftp://[user[:password]@]server[:port]/path/to/remote/resource.mpeg
	var (
		err error
	)
	for range 3 {
		_, err = p.ftpConn.GetEntry(req.FilePath)
		if err != nil {
			slog.Error("list failed", "err", err)
			err = p.connectFtp()
			if err != nil {
				slog.Error("reconnect ftp failed", "err", err)

			}
		} else {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	userPass := ""
	if p.ftpAuth.User.StringValue.Value != "" || p.ftpAuth.Password.ObscureStringValue.Value != "" {
		userPass = fmt.Sprintf("%s:%s@", p.ftpAuth.User.StringValue.Value, p.ftpAuth.Password.ObscureStringValue.Value)
	}
	fileUrl := fmt.Sprintf("ftp://%s%s%s", userPass, p.ftpAuth.Addr.StringValue.Value, req.FilePath)
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
