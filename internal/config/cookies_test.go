package config

import (
	"strings"
	"testing"
)

func TestParseCookieValidationResponseReturnsBilibiliReason(t *testing.T) {
	ok, err := parseCookieValidationResponse([]byte(`{"code":-101,"message":"账号未登录"}`))
	if ok {
		t.Fatal("ok = true, want false")
	}
	if err == nil {
		t.Fatal("err = nil, want Bilibili rejection reason")
	}
	if !strings.Contains(err.Error(), "Bilibili code -101") || !strings.Contains(err.Error(), "账号未登录") {
		t.Fatalf("err = %q, want code and message", err.Error())
	}
}

func TestParseCookieValidationResponseAcceptsLoggedInUser(t *testing.T) {
	ok, err := parseCookieValidationResponse([]byte(`{"code":0,"data":{"mid":10001}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("ok = false, want true")
	}
}
