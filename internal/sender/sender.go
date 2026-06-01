package sender

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/noahlias/bili-live-tui/internal/config"
	"github.com/noahlias/bili-live-tui/internal/getter"
)

const senderUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"

var (
	httpClient        = &http.Client{Timeout: 10 * time.Second}
	liveSendURL       = "https://api.live.bilibili.com/msg/send"
	videoHeartbeatURL = "https://api.bilibili.com/x/click-interface/web/heartbeat"
	heartbeatOnce     sync.Once
)

func heartbeat() {
	if err := sendVideoHeartbeat(0); err != nil {
		fmt.Println("failed to send heartbeat; error:", err)
		os.Exit(0)
	}
	time.AfterFunc(10*time.Second, heartbeat)
}

func SendMsg(roomId int64, msg string, busChan chan getter.DanmuMsg) {
	msgRune := []rune(msg)
	for i := 0; i < len(msgRune); i += 20 {
		end := i + 20
		if end > len(msgRune) {
			end = len(msgRune)
		}
		if err := sendLiveDanmaku(roomId, string(msgRune[i:end])); err != nil {
			busChan <- getter.DanmuMsg{Author: "system", Content: "发送弹幕失败", Type: ""}
		}
		if end < len(msgRune) {
			time.Sleep(time.Second)
		}
	}
}

func Run() {
	heartbeatOnce.Do(func() {
		go heartbeat()
	})
}

func sendLiveDanmaku(roomID int64, msg string) error {
	return postBiliForm(liveSendURL, url.Values{
		"roomid":   {strconv.FormatInt(roomID, 10)},
		"color":    {"16777215"},
		"fontsize": {"25"},
		"mode":     {"1"},
		"msg":      {msg},
		"bubble":   {"0"},
		"rnd":      {strconv.FormatInt(time.Now().Unix(), 10)},
	})
}

func sendVideoHeartbeat(playedTime int64) error {
	mid, err := strconv.ParseInt(config.Auth.DedeUserID, 10, 64)
	if err != nil || mid <= 0 {
		return fmt.Errorf("invalid DedeUserID")
	}
	return postBiliForm(videoHeartbeatURL, url.Values{
		"aid":         {"242531611"},
		"cid":         {"173439442"},
		"mid":         {strconv.FormatInt(mid, 10)},
		"start_ts":    {strconv.FormatInt(time.Now().Unix(), 10)},
		"played_time": {strconv.FormatInt(playedTime, 10)},
	})
}

func postBiliForm(endpoint string, values url.Values) error {
	values.Set("csrf", config.Auth.BiliJCT)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Origin", "https://www.bilibili.com")
	req.Header.Set("Referer", "https://www.bilibili.com")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", senderUserAgent)
	req.Header.Set("Cookie", cookieHeader())

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("(%d) %s", result.Code, result.Message)
	}
	return nil
}

func cookieHeader() string {
	if strings.TrimSpace(config.Config.Cookie) != "" {
		return config.Config.Cookie
	}
	parts := []string{
		"DedeUserID=" + config.Auth.DedeUserID,
		"SESSDATA=" + config.Auth.SESSDATA,
	}
	if config.Auth.DedeUserIDCkMd5 != "" {
		parts = append(parts, "DedeUserID__ckMd5="+config.Auth.DedeUserIDCkMd5)
	}
	if config.Auth.BiliJCT != "" {
		parts = append(parts, "bili_jct="+config.Auth.BiliJCT)
	}
	return strings.Join(parts, ";")
}
