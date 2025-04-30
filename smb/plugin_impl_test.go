package main_test

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"testing"

	"github.com/medianexapp/go-smb2"
	// wasi_net "github.com/labulakalia/wazero_net/wasi/net"
)

func TestSmb(t *testing.T) {
	slog.Info("read success")
	smbDialer := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     "labulakalia",
			Password: "109097",
		},
	}
	t.Log("start dial")
	addr := "192.168.123.213:445"
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("dial")
	smbSession, err := smbDialer.DialConn(context.Background(), conn, addr)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(smbSession.ListSharenames())

	share, err := smbSession.Mount("labulakalia")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(share.ReadDir(""))
	slog.Info("read success")
	sp := strings.Split("/labulakalia", "/")[1:]

	shareName := sp[0]
	t.Log(shareName)
	t.Log(strings.Join(sp[1:], "/"))

}
