package main

import (
	"fmt"
	"time"
)

var (
	AlipanURL = "https://openapi.alipan.com"
)

type QrcodeResponse struct {
	QrCodeUrl string `json:"qrCodeUrl"`
	Sid       string `json:"sid"`
}

type ErrResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestId string `json:"requestId"`
}

func (e ErrResponse) Error() string {
	return fmt.Sprintf("%s(%s)", e.Message, e.Code)
}

// /oauth/users/info
type UserInfoResponse struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Avator string `json:"avator"`
	Phone  string `json:"phone"`
}

// /adrive/v1.0/user/getDriveInfo
type UserGetDriverInfoResponse struct {
	Name             string `json:"name"`
	DefaultDriverId  string `json:"default_drive_id"`
	ResourceDriverId string `json:"resource_drive_id"`
	BackupDriverId   string `json:"backup_drive_id"`
}

type FileEntry struct {
	DriveId      string    `json:"drive_id"`
	FileId       string    `json:"file_id"`
	ParentFileId string    `json:"parent_file_id"`
	Name         string    `json:"name"`
	Size         uint64    `json:"size"`
	ContentHash  string    `json:"content_hash"`
	Category     string    `json:"category"`
	Type         string    `json:"type"`
	CreatedTime  time.Time `json:"created_at"`
	UpdatedTime  time.Time `json:"updated_at"`
}

// 获取文件列表
// /adrive/v1.0/openFile/list
type OpenFileListRequest struct {
	DriveId      string `json:"drive_id"`
	Limit        int    `json:"limit"`
	Marker       string `json:"marker"`
	OrderBy      string `json:"order_by"` // name_enhanced
	ParentFileId string `json:"parent_file_id"`
	Category     string `json:"category"`
}

type OpenFileListResponse struct {
	NextMarker string       `json:"next_marker"`
	Items      []*FileEntry `json:"items"`
}

// /adrive/v1.0/openFile/get_by_path
type OpenFilegetbypathRequest struct {
	DriveId  string `json:"drive_id"`
	FilePath string `json:"file_path"`
}

type OpenFilegetbypathResponse struct {
	*FileEntry
}

// /adrive/v1.0/openFile/getDownloadUrl
type OpenFilegetDownloadUrlRequest struct {
	DriveId   string `json:"drive_id"`
	FileId    string `json:"file_id"`
	ExpireSec int    `json:"expire_sec"`
}

type OpenFilegetDownloadUrlResponse struct {
	Url        string `json:"url"`
	Expiration string `json:"expiration"`
	Method     string `json:"method"`
}

type CacheFileEntry struct {
	*FileEntry
	ExpireTime uint64
}

type OpenFileGetVideoPreviewPlayInfoRequest struct {
	DriveId         string `json:"drive_id"`
	FileId          string `json:"file_id"`
	Category        string `json:"category"`       // category
	TemplateId      string `json:"template_id"`    // HD|FHD|QHD
	UrlExpireSec    int    `json:"url_expire_sec"` // s
	GetSubtitleInfo bool   `json:"get_subtitle_info"`
}

type OpenFileGetVideoPreviewPlayInfoResponse struct {
	DomainId             string                `json:"domain_id"`
	DriveId              string                `json:"drive_id"`
	FileId               string                `json:"file_id"`
	VideoPreViewPlayInfo *VideoPreViewPlayInfo `json:"video_preview_play_info"`
}

type VideoPreViewPlayInfo struct {
	Category                        string                         `json:"category"` // category
	LiveTranscodingTaskList         []*LiveTranscodingTask         `json:"live_transcoding_task_list"`
	LiveTranscodingSubtitleTaskList []*LiveTranscodingSubtitleTask `json:"live_transcoding_subtitle_task_list"`
}

type LiveTranscodingTask struct {
	TemplateId string `json:"template_id"` // HD|FHD|QHD
	Status     string `json:"status"`      // finished running failed
	Url        string `json:"url"`
}

type LiveTranscodingSubtitleTask struct {
	Language string `json:"language"`
	Status   string `json:"status"` // finished running failed
	Url      string `json:"url"`
}
