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
	UID      string `json:"uid"`
	Nickname string `json:"nickname"`
}

type FileListRequest struct {
}

type FileListResponse struct {
}
