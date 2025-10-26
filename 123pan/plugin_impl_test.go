package main

import (
	"strings"
	"testing"

	"github.com/medianexapp/plugin_api/plugin"
)

func TestPluginImpl(t *testing.T) {
	p := NewPluginImpl()
	auth, _ := p.GetAuth()
	method := auth.AuthMethods[0].Method
	method = method // &lt;= save auth data
	 
	authData, err := p.CheckAuthMethod(&plugin.AuthMethod{
		Method: auth.AuthMethods[0].Method,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = p.CheckAuthData(authData.AuthDataBytes)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.GetDirEntry(&plugin.GetDirEntryRequest{
		Path:     "/",
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, fileEntry := range resp.FileEntries {
		if fileEntry.FileType != plugin.FileEntry_FileTypeFile {
			continue
		}
		t.Log("file entry name", fileEntry.Name)
		// if is movie get file resource
		if strings.HasSuffix(fileEntry.Name, "mp4") || strings.HasSuffix(fileEntry.Name, "mkv") {
			fileResource, err := p.GetFileResource(&plugin.GetFileResourceRequest{
				FilePath:  "/" + fileEntry.Name,
				FileEntry: fileEntry,
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("get file %s fileResource %+v", fileEntry.Name, fileResource.FileResourceData)
		}
	}
	return
}
