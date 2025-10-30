package main

import "github.com/medianexapp/plugin_api/plugin"

const PanURl = "https://open-api.123pan.com"

type Response struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	Data     any    `json:"data"`
	XTraceId string `json:"x-traceID"`
}

type AuthToken struct {
	ClientId      *plugin.Formdata_FormItem_StringValue `json:"clientID"`
	ClientSecret  *plugin.Formdata_FormItem_StringValue `json:"clientSecret"`
	AccessToken   string                                `json:"accessToken"`
	ExpiredAt     string                                `json:"expiredAt"`
	ExpiredAtUnix int64                                 `json:"expiredAtUnix"`
}

type UserInfo struct {
	UID      int64  `json:"uid"`
	Nickname string `json:"nickname"`
}

type FileItem struct {
	FileId        uint64 `json:"fileId"`
	FileName      string `json:"fileName"`
	ParentFieldId uint64 `json:"parentFieldId"`
	Type          int    `json:"type"` // 0-文件  1-文件夹
	Size          uint64 `json:"size"`
	Etag          string `json:"etag"`
	Category      int    `json:"category"` // 0-未知 1-音频 2-视频 3-图片
	Status        int    `json:"status"`   // 文件审核状态。 大于 100 为审核驳回文件
	Trashed       int    `json:"trashed"`  // 0 否 1是
	CreateAt      string `json:"createAt"`
	UpdateAt      string `json:"updateAt"`
}

type FileListResponse struct {
	LastFileId int64      `json:"lastFileId"`
	FileList   []FileItem `json:"fileList"`
}

type DownloadInfo struct {
	DownloadUrl string `json:"downloadUrl"`
}

type userTranscodeVideo struct {
	ID         int    `json:"Id"`
	UID        int    `json:"Uid"`
	Resolution string `json:"Resolution"`
	Status     int    `json:"Status"` // 255
	CreateAt   string `json:"CreateAt"`
	UpdateAt   string `json:"UpdateAt"`
	Files      []struct {
		FileName   string `json:"FileName"` // m3u8
		FileSize   string `json:"FileSize"`
		Resolution string `json:"Resolution"`
		CreateAt   string `json:"CreateAt"`
		URL        string `json:"Url"`
	} `json:"Files"`
}

type PlayerVideoResponse struct {
	UserTranscodeVideoList []userTranscodeVideo `json:"UserTranscodeVideoList"`
}
