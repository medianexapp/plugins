package util

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/medianexapp/plugin_api/httpclient"
	"github.com/medianexapp/plugin_api/plugin"
)

//go:embed env.json
var envData []byte

type Env struct {
	ServerAddr string `json:"server_addr"`
}

func init() {
	env := &Env{}
	json.Unmarshal(envData, env)
	ServerAddr = env.ServerAddr
}

var (
	ServerAddr         = ""
	getAuthAddrUri     = "/api/get_auth_addr"
	getAuthTokenUri    = "/api/get_auth_token"
	getAuthQrcodeUri   = "/api/get_auth_qrcode"
	checkAuthQrcodeUri = "/api/check_auth_qrcode"
	Client             = httpclient.NewClient()
)

func GetAuthAddr(pluginId string) string {
	return fmt.Sprintf("%s%s?id=%s", ServerAddr, getAuthAddrUri, pluginId)
}

type GetAuthTokenRequest struct {
	Id           string `json:"id"`
	State        string `json:"state"`
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RefreshToken string `json:"refresh_token"`
}

func GetAuthToken(req *GetAuthTokenRequest) (*plugin.Token, error) {
	u := url.Values{}
	u.Set("id", req.Id)
	u.Set("state", req.State)
	u.Set("code", req.Code)
	u.Set("code_verifier", req.CodeVerifier)
	u.Set("refresh_token", req.RefreshToken)
	resp, err := Client.Get(fmt.Sprintf("%s%s?%s", ServerAddr, getAuthTokenUri, u.Encode()))
	if err != nil {
		err = errors.Unwrap(err)
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get auth token failed: %s", respBytes)
	}
	token := &plugin.Token{}
	err = token.UnmarshalVT(respBytes)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func GetAuthQrcode(id string) ([]byte, error) {
	url := fmt.Sprintf("%s%s?id=%s", ServerAddr, getAuthQrcodeUri, id)
	resp, err := Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	slog.Info("get qrcode data", "id", id, "resp", string(respBytes))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get auth qrcode failed: %s", respBytes)
	}
	return respBytes, nil
}

func CheckAuthQrcode(id, key string) (*plugin.Token, error) {
	url := fmt.Sprintf("%s%s?id=%s&key=%s", ServerAddr, checkAuthQrcodeUri, id, key)
	resp, err := Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.ContentLength == 0 {
		return nil, nil
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	slog.Info("get qrcode data", "id", id, "resp", string(respBytes))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get auth qrcode failed: %s", respBytes)
	}
	token := &plugin.Token{}
	err = token.UnmarshalVT(respBytes)
	if err != nil {
		return nil, err
	}
	return token, nil
}
