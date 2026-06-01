package sender

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/noahlias/bili-live-tui/internal/config"
)

func TestSendLiveDanmakuPostsExpectedForm(t *testing.T) {
	restore := setupSenderTest(t)
	defer restore()

	var gotForm url.Values
	var gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		gotCookie = r.Header.Get("Cookie")
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		gotForm = r.Form
		w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()
	httpClient = srv.Client()
	liveSendURL = srv.URL

	if err := sendLiveDanmaku(12345, "hello"); err != nil {
		t.Fatalf("sendLiveDanmaku: %v", err)
	}

	if gotCookie != config.Config.Cookie {
		t.Fatalf("Cookie = %q, want %q", gotCookie, config.Config.Cookie)
	}
	assertFormValue(t, gotForm, "roomid", "12345")
	assertFormValue(t, gotForm, "msg", "hello")
	assertFormValue(t, gotForm, "color", "16777215")
	assertFormValue(t, gotForm, "fontsize", "25")
	assertFormValue(t, gotForm, "mode", "1")
	assertFormValue(t, gotForm, "bubble", "0")
	assertFormValue(t, gotForm, "csrf", "csrf-token")
	if gotForm.Get("rnd") == "" {
		t.Fatal("rnd is empty")
	}
}

func TestSendVideoHeartbeatUsesDedeUserIDAsMid(t *testing.T) {
	restore := setupSenderTest(t)
	defer restore()

	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		gotForm = r.Form
		w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()
	httpClient = srv.Client()
	videoHeartbeatURL = srv.URL

	if err := sendVideoHeartbeat(12); err != nil {
		t.Fatalf("sendVideoHeartbeat: %v", err)
	}

	assertFormValue(t, gotForm, "mid", "10001")
	assertFormValue(t, gotForm, "played_time", "12")
	assertFormValue(t, gotForm, "csrf", "csrf-token")
	if gotForm.Get("start_ts") == "" {
		t.Fatal("start_ts is empty")
	}
}

func TestPostBiliFormReturnsAPIError(t *testing.T) {
	restore := setupSenderTest(t)
	defer restore()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":-400,"message":"bad request"}`))
	}))
	defer srv.Close()
	httpClient = srv.Client()

	if err := postBiliForm(srv.URL, url.Values{}); err == nil {
		t.Fatal("postBiliForm error = nil, want API error")
	}
}

func setupSenderTest(t *testing.T) func() {
	t.Helper()
	oldClient := httpClient
	oldLiveURL := liveSendURL
	oldHeartbeatURL := videoHeartbeatURL
	oldConfig := config.Config
	oldAuth := config.Auth

	httpClient = &http.Client{Timeout: time.Second}
	config.Config.Cookie = "DedeUserID=10001; SESSDATA=session; bili_jct=csrf-token"
	config.Auth = config.CookieAuth{
		DedeUserID:      "10001",
		DedeUserIDCkMd5: "md5",
		SESSDATA:        "session",
		BiliJCT:         "csrf-token",
	}

	return func() {
		httpClient = oldClient
		liveSendURL = oldLiveURL
		videoHeartbeatURL = oldHeartbeatURL
		config.Config = oldConfig
		config.Auth = oldAuth
	}
}

func assertFormValue(t *testing.T, form url.Values, key string, want string) {
	t.Helper()
	if got := form.Get(key); got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}
