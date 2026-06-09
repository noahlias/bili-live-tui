package config

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tidwall/gjson"
)

var requiredCookieFields = []string{"SESSDATA", "DedeUserID", "bili_jct"}

func printCookieHelp(configFile string) {
	fmt.Println("Cookie missing or invalid.")
	fmt.Println("Open https://live.bilibili.com in your browser, copy the full Cookie header, and paste it into:")
	fmt.Println(configFile)
}

func parseCookies(cookieStr string) map[string]string {
	out := make(map[string]string)
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		out[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return out
}

func authFromCookieHeader(cookie string) (CookieAuth, []string, []string) {
	kvs := parseCookies(cookie)
	auth := CookieAuth{
		SESSDATA:        kvs["SESSDATA"],
		DedeUserID:      kvs["DedeUserID"],
		DedeUserIDCkMd5: kvs["DedeUserID__ckMd5"],
		BiliJCT:         kvs["bili_jct"],
	}
	missing := make([]string, 0, len(requiredCookieFields))
	for _, field := range requiredCookieFields {
		if kvs[field] == "" {
			missing = append(missing, field)
		}
	}
	invalid := make([]string, 0, len(requiredCookieFields)+1)
	for _, field := range requiredCookieFields {
		if kvs[field] != "" && !isSaneCookieValue(kvs[field]) {
			invalid = append(invalid, field)
		}
	}
	if auth.DedeUserIDCkMd5 != "" && !isSaneCookieValue(auth.DedeUserIDCkMd5) {
		invalid = append(invalid, "DedeUserID__ckMd5")
	}
	return auth, missing, invalid
}

func setAuthFromCookieHeader(cookie string) (bool, string) {
	auth, missing, invalid := authFromCookieHeader(cookie)
	Auth = auth
	if len(missing) > 0 {
		return false, "cookie missing required fields: " + strings.Join(missing, ", ")
	}
	if len(invalid) > 0 {
		return false, "cookie has invalid fields: " + strings.Join(invalid, ", ")
	}
	return true, ""
}

func validateCookie(cookie string) (bool, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "https://api.bilibili.com/x/space/myinfo", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	req.Header.Set("Cookie", cookie)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	code := gjson.GetBytes(body, "code").Int()
	if code == 0 {
		return gjson.GetBytes(body, "data.mid").Int() > 0, nil
	}
	if code == -101 {
		return false, nil
	}
	return false, fmt.Errorf("code %d", code)
}

func normalizeCookieHeader(cookie string) string {
	if cookie == "" {
		return cookie
	}
	clean := make([]rune, 0, len(cookie))
	for _, r := range cookie {
		if r == '\r' || r == '\n' || r == 0x7f {
			continue
		}
		if r < 0x20 {
			continue
		}
		clean = append(clean, r)
	}
	return strings.TrimSpace(string(clean))
}

func sanitizeCookieValue(v string) string {
	if v == "" {
		return v
	}
	clean := make([]rune, 0, len(v))
	for _, r := range v {
		if r == ';' || r == '\r' || r == '\n' || r == 0x7f {
			continue
		}
		if r < 0x20 {
			continue
		}
		clean = append(clean, r)
	}
	return strings.TrimSpace(string(clean))
}

func isSaneCookieValue(v string) bool {
	if v == "" || !utf8.ValidString(v) {
		return false
	}
	for _, r := range v {
		if r < 0x20 || r > 0x7e || r == ';' {
			return false
		}
	}
	return true
}
