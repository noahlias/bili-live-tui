package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigFromFilePreservesFullCookieHeader(t *testing.T) {
	oldConfig := Config
	t.Cleanup(func() {
		Config = oldConfig
	})

	cookie := "DedeUserID=10001; DedeUserID__ckMd5=abcdef; SESSDATA=session-token; bili_jct=csrf-token"
	configFile := filepath.Join(t.TempDir(), "config.toml")
	data := strings.Join([]string{
		"Cookie = " + quoteTOMLString(cookie),
		"RoomId = 23530682",
	}, "\n")
	if err := os.WriteFile(configFile, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	if err := loadConfigFromFile(configFile); err != nil {
		t.Fatal(err)
	}

	if Config.Cookie != cookie {
		t.Fatalf("Config.Cookie = %q, want %q", Config.Cookie, cookie)
	}
}

func TestSetAuthFromCookieHeaderReportsMissingBiliJCT(t *testing.T) {
	oldAuth := Auth
	t.Cleanup(func() {
		Auth = oldAuth
	})

	ok, errMsg := setAuthFromCookieHeader("DedeUserID=10001; SESSDATA=session-token")
	if ok {
		t.Fatal("setAuthFromCookieHeader ok = true, want false")
	}
	if errMsg != "cookie missing required fields: bili_jct" {
		t.Fatalf("errMsg = %q, want missing bili_jct", errMsg)
	}
}

func quoteTOMLString(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
