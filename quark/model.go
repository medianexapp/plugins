package main

import "github.com/medianexapp/plugin_api/plugin"

type Response struct {
	Status  int
	Code    int
	Message string
	Data    any
}

type FileData struct {
	List []File `json:"list"`
}

type File struct {
	Fid           string `json:"fid"`
	FileName      string `json:"file_name"`
	Size          uint64 `json:"size"`
	FileType      uint64 `json:"file_type"`
	CreatedAt     uint64 `json:"created_at"`
	UpdatedAt     uint64 `json:"updated_at"`
	Dir           bool   `json:"dir"`
	File          bool   `json:"file"`
	UpdatedViewAt uint64 `json:"updated_view_at"`

	DownloadUrl string `json:"download_url"`
}

type PlayReq struct {
	Fid         string `json:"fid"`
	Resolutions string `json:"resolutions"`
	Supports    string `json:"supports"`
}

type PlayData struct {
	VideoList []VideoList `json:"video_list"`
}

type VideoList struct {
	Resolution  string `json:"resolution"`
	TransStatus string `json:"trans_status"` // success
	VideoInfo   struct {
		URL string `json:"url"`
	} `json:"video_info"`
}

var resolutionMap = map[string]plugin.FileResource_Resolution{
	"4k":    plugin.FileResource_UHD,
	"super": plugin.FileResource_FHD,
	"high":  plugin.FileResource_HD,
	"low":   plugin.FileResource_LD,
}
