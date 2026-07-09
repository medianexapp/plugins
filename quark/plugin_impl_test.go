package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/medianexapp/plugin_api/plugin"
)

func TestPluginImpl(t *testing.T) {

	p := NewPluginImpl()
	auth, err := p.GetAuth()

	if err != nil {
		t.Fatal(err)
	}

	if len(auth.AuthMethods) != 2 {
		t.Fatal("auth methods count != 2")
	}

	cookie := plugin.String(`token`)

	auth.AuthMethods[0].Method.(*plugin.AuthMethod_Formdata).Formdata.FormItems[0] = &plugin.Formdata_FormItem{
		Value: cookie,
	}
	d, err := p.CheckAuthMethod(auth.AuthMethods[0])
	if err != nil {
		t.Fatal("err", err)
	}

	err = p.CheckAuthData(d.AuthDataBytes)
	if err != nil {
		t.Fatal(err)
	}

}

func TestQuarkQrcode(t *testing.T) {
	p := NewPluginImpl()

	token, err := p.getQrcodeToken()
	if err != nil {
		t.Fatal(err)
	}

	// 直接输出到控制台
	fmt.Println(p.qrcodeData(token))
	for {
		res, err := p.checkQrcode(token)
		if err == nil && res != nil {
			t.Log("cookie", res)
			break
		}
		time.Sleep(time.Second * 2)
	}
	p.cookies = p.hb.GetCookies()
	fmt.Println("cookie", p.cookies)
}
