package main

import "time"

type AuthLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type TokenData struct {
	Token string `json:"token"`
}

type FsListReq struct {
	Page     int64  `json:"page"`
	Password string `json:"password"`
	Path     string `json:"path"`
	PerPage  int64  `json:"per_page"`
	Refresh  bool   `json:"refresh"`
}
type FsListResp struct {
	Total    int       `json:"total"`
	Contents []Content `json:"content"`
}

type Content struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	IsDir    bool      `json:"is_dir"`
	Modified time.Time `json:"modified"`
	Created  time.Time `json:"created"`

	RawUrl string `json:"raw_url"`
}

type FsGetReq struct {
	Path string
}
type FsGetResp struct {
	Path string
}
