package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/medianexapp/plugin_api/plugin"
)

func TestPluginImpl(t *testing.T) {
	p := NewPluginImpl()
	auth, _ := p.GetAuth()
	method := auth.AuthMethods[0].Method
	t.Log("get method", method)
	formData := method.(*plugin.AuthMethod_Formdata).Formdata

	formData.FormItems[0].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value = ""
	formData.FormItems[1].Value.(*plugin.Formdata_FormItem_StringValue).StringValue.Value = ""

	method.(*plugin.AuthMethod_Formdata).Formdata = formData
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
	req := &plugin.GetDirEntryRequest{
		Path:     "/",
		Page:     1,
		PageSize: 100,
	}
	resp, err := p.GetDirEntry(req)
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Println("resp", resp.FileEntries)
	req.Path += resp.FileEntries[0].Name
	req.FileEntry = resp.FileEntries[0]
	// fmt.Println("FileEntry", req.FileEntry)
	resp, err = p.GetDirEntry(req)
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Println("----resp", resp.FileEntries)
	req.Path += "/" + resp.FileEntries[0].Name
	req.FileEntry = resp.FileEntries[0]
	fmt.Println("FileEntry", req.FileEntry)
	resp, err = p.GetDirEntry(req)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("resp", resp.FileEntries)
	for _, fileEntry := range resp.FileEntries {
		if fileEntry.FileType != plugin.FileEntry_FileTypeFile {
			continue
		}

		// if is movie get file resource
		if strings.HasSuffix(fileEntry.Name, "mp4") || strings.HasSuffix(fileEntry.Name, "mkv") {
			fileResource, err := p.GetFileResource(&plugin.GetFileResourceRequest{
				FilePath:  req.Path + "/" + fileEntry.Name,
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
