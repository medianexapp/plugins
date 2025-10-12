package util

import (
	"encoding/json"
	"testing"
)

func TestAuth(t *testing.T) {
	type QrcodeResponse struct {
		QrCodeUrl string `json:"qrCodeUrl"`
		Sid       string `json:"sid"`
	}
	res, err := GetAuthQrcode("alipan")
	if err != nil {
		t.Errorf("GetAuthQrcode failed: %v", err)
	}
	qrResp := &QrcodeResponse{}
	err = json.Unmarshal(res, qrResp)
	if err != nil {
		t.Errorf("json.Unmarshal failed: %v", err)
	}
	token, err := CheckAuthQrcode("alipan", qrResp.Sid)
	if err != nil {
		t.Fatal("err is nil")
	}
	if token != nil {
		t.Fatal("token is nil")
	}
}

func TestGetQrcode(t *testing.T) {
	res, err := GetAuthQrcode("alipan")
	if err != nil {
		t.Errorf("GetAuthQrcode failed: %v", err)
	}
	t.Log(string(res))
}
