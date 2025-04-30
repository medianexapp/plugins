package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/hashicorp/go-version"
	"github.com/medianexapp/plugin_api/httpclient"
)

var (
	client = httpclient.NewClient()
)

type pluginConfig struct {
	Id        string   `toml:"id"`
	Name      string   `toml:"name"`
	Desc      string   `toml:"desc"`
	Author    []string `toml:"author"`
	Version   string   `toml:"version"`
	Icon      string   `toml:"icon"`
	Changelog []string `toml:"changelog"`
}

func main() {
	dirs, err := os.ReadDir(".")
	if err != nil {
		slog.Error("read dir failed", "err", err)
		os.Exit(1)
	}
	for _, dir := range dirs {
		if dir.Name() == "smb" || dir.Name() == "util" || !dir.IsDir() {
			continue
		}

		pluginFile, err := os.Open(filepath.Join(dir.Name(), "plugin.toml"))
		if err != nil {
			continue
		}
		pluginConfig := &pluginConfig{}
		_, err = toml.NewDecoder(pluginFile).Decode(pluginConfig)
		if err != nil {
			slog.Error("read plugin.toml failed", "err", err)
			os.Exit(1)
		}
		pluginFile.Close()
		resp, err := client.Get(os.Getenv("SERVER_ADDR") + "/api/get_plugin_version/" + pluginConfig.Id)
		if err != nil {
			slog.Error("get version failed", "err", err)
			os.Exit(1)
		}
		res, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("get version failed", "err", err)
			os.Exit(1)
		}
		needUpload := false
		if len(string(res)) != 0 {
			fmt.Println(string(res), pluginConfig.Version)
			serverVersion, err1 := version.NewVersion(string(res))
			localVersion, err2 := version.NewVersion(pluginConfig.Version)
			if err1 != nil || err2 != nil {
				slog.Error("get version failed", "err1", err1, "err2", err2)
				os.Exit(1)
			}

			if localVersion.GreaterThan(serverVersion) {
				needUpload = true
			}
		} else {
			needUpload = true
		}
		if needUpload {
			file, err := os.Open(filepath.Join(pluginConfig.Id, "dist", fmt.Sprintf("%s_%s.zip", pluginConfig.Id, pluginConfig.Version)))
			if err != nil {
				slog.Error("get version failed", "err", err)
				os.Exit(1)
			}
			req, err := http.NewRequest(http.MethodPost, os.Getenv("SERVER_ADDR")+"/api/upload_plugin", file)
			if err != nil {
				slog.Error("new request failed", "err", err)
				os.Exit(1)
			}
			req.Header.Set("UploadKey", os.Getenv("UPLOAD_KEY"))
			resp, err := client.Do(req)
			if err != nil {
				slog.Error("do request failed", "err", err)
				os.Exit(1)
			}

			if resp.StatusCode != http.StatusOK {
				slog.Info("upload failed", "name", pluginConfig.Name)
			} else {
				slog.Info("upload success", "name", pluginConfig.Name)
			}
			resp.Body.Close()
		} else {
			fmt.Println(pluginConfig.Name, "no upload")
		}

	}
}
