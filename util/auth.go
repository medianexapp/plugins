package util

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/medianexapp/plugin_api/httpclient"
	"github.com/medianexapp/plugin_api/plugin"
)

//go:embed env.json
var envData []byte

type Env struct {
	ServerAddr string `json:"server_addr"`
}

func init() {
	env := &Env{
		ServerAddr: "http://127.0.0.1:19971",
	}
	json.Unmarshal(envData, env)
	ServerAddr = env.ServerAddr
}

var (
	ServerAddr         = ""
	getAuthAddrUri     = "/api/get_auth_addr"
	getAuthTokenUri    = "/api/get_auth_token"
	getAuthQrcodeUri   = "/api/get_auth_qrcode_v2"
	checkAuthQrcodeUri = "/api/check_auth_qrcode"
	HttpClient         = httpclient.NewClient()
)

func GetAuthAddr(pluginId string) string {
	return fmt.Sprintf("%s%s?id=%s", ServerAddr, getAuthAddrUri, pluginId)
}

type GetAuthTokenRequest struct {
	Id           string `json:"id"`
	Code         string `json:"code"`
	RefreshToken string `json:"refresh_token"`
	Uid          string `json:"uid"`
}

func GetAuthToken(req *GetAuthTokenRequest) (*plugin.Token, error) {
	slog.Info("start get auth token", "req", req)
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := HttpClient.Post(fmt.Sprintf("%s%s", ServerAddr, getAuthTokenUri), "application/json", bytes.NewBuffer(reqData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		slog.Error("resp err msg", "errMsg", string(errMsg))
		return nil, errors.New(string(errMsg))
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	token := &plugin.Token{}
	err = token.UnmarshalVT(respBytes)
	if err != nil {
		return nil, err
	}
	return token, nil
}

type RequestQrcodeParams struct {
	Method string            `json:"method"`
	URL    string            `json:"url"`
	Data   string            `json:"data"`
	Header map[string]string `json:"header"`
}

func GetAuthQrcode(id string) ([]byte, error) {
	authQrcodeUrl := fmt.Sprintf("%s%s?id=%s", ServerAddr, getAuthQrcodeUri, id)
	resp, err := HttpClient.Get(authQrcodeUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	qrcodeParams := RequestQrcodeParams{}
	err = json.Unmarshal(body, &qrcodeParams)
	if err != nil {
		return nil, err
	}
	qrcodeParams.Data, _ = url.PathUnescape(qrcodeParams.Data)

	req, err := http.NewRequest(qrcodeParams.Method, qrcodeParams.URL, strings.NewReader(qrcodeParams.Data))
	if err != nil {
		return nil, err
	}
	for k, v := range qrcodeParams.Header {
		req.Header.Set(k, v)
	}
	resp, err = HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func CheckAuthQrcode(id, key string) (*plugin.Token, error) {
	url := fmt.Sprintf("%s%s?id=%s&key=%s", ServerAddr, checkAuthQrcodeUri, id, key)
	resp, err := HttpClient.Get(url)
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
