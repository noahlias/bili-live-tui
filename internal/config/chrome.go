package config

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

func tryImportChromeCookie(configFile string) (bool, string) {
	if configFile == "" {
		return false, "config file path empty"
	}
	cookie, err := readChromeCookie()
	if err != nil || cookie == "" {
		if err != nil {
			return false, err.Error()
		}
		return false, "cookie empty"
	}
	return applyImportedCookie(configFile, cookie)
}

func tryImportChromeCookieWithPassword(configFile string, password string) (bool, string) {
	if configFile == "" {
		return false, "config file path empty"
	}
	if strings.TrimSpace(password) == "" {
		return false, "password empty"
	}
	cookie, err := readChromeCookieWithPasswordOverride(password)
	if err != nil || cookie == "" {
		if err != nil {
			return false, err.Error()
		}
		return false, "cookie empty"
	}
	return applyImportedCookie(configFile, cookie)
}

func applyImportedCookie(configFile string, cookie string) (bool, string) {
	Config.Cookie = normalizeCookieHeader(cookie)
	if ok, errMsg := setAuthFromCookieHeader(Config.Cookie); !ok {
		return false, errMsg
	}
	if err := saveConfig(configFile); err != nil {
		return false, err.Error()
	}
	if !configHasCookie(configFile) {
		return false, "cookie not written"
	}
	return true, ""
}

func readChromeCookie() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return readChromeCookieDarwin()
	case "linux":
		return readChromeCookieLinux()
	default:
		return "", fmt.Errorf("chrome cookie import not supported on %s", runtime.GOOS)
	}
}

func readChromeCookieDarwin() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	password, err := chromeKeychainPassword()
	if err != nil || password == "" {
		return "", fmt.Errorf("chrome keychain password not available")
	}
	bases := chromeDarwinBases(home)
	for _, base := range bases {
		paths := chromeCookieDBPaths(base)
		for _, dbPath := range paths {
			cookie, err := readChromeCookieWithPassword(password, dbPath, 1003)
			if err == nil && cookie != "" {
				return cookie, nil
			}
		}
	}
	if cookie, err := readChromeCookiePython(); err == nil && cookie != "" {
		return cookie, nil
	}
	return "", fmt.Errorf("chrome cookies not found")
}

func readChromeCookieLinux() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	passwords := chromePasswordCandidates()
	bases := chromeLinuxBases(home)
	for _, base := range bases {
		paths := chromeCookieDBPaths(base)
		for _, dbPath := range paths {
			for _, password := range passwords {
				cookie, err := readChromeCookieWithPassword(password, dbPath, 1)
				if err == nil && cookie != "" {
					return cookie, nil
				}
			}
		}
	}
	if cookie, err := readChromeCookiePython(); err == nil && cookie != "" {
		return cookie, nil
	}
	return "", fmt.Errorf("chrome cookies not found")
}

func readChromeCookieWithPasswordOverride(password string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		for _, base := range chromeDarwinBases(home) {
			paths := chromeCookieDBPaths(base)
			for _, dbPath := range paths {
				cookie, err := readChromeCookieWithPassword(password, dbPath, 1003)
				if err == nil && cookie != "" {
					return cookie, nil
				}
			}
		}
		if cookie, err := readChromeCookiePython(); err == nil && cookie != "" {
			return cookie, nil
		}
		return "", fmt.Errorf("chrome cookies not found")
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		for _, base := range chromeLinuxBases(home) {
			paths := chromeCookieDBPaths(base)
			for _, dbPath := range paths {
				cookie, err := readChromeCookieWithPassword(password, dbPath, 1)
				if err == nil && cookie != "" {
					return cookie, nil
				}
			}
		}
		if cookie, err := readChromeCookiePython(); err == nil && cookie != "" {
			return cookie, nil
		}
		return "", fmt.Errorf("chrome cookies not found")
	default:
		return "", fmt.Errorf("chrome cookie import not supported on %s", runtime.GOOS)
	}
}

func readChromeCookiePython() (string, error) {
	scriptPath, err := ensureCookieImportScript()
	if err != nil {
		return "", err
	}
	var cmd *exec.Cmd
	if _, err := exec.LookPath("uvx"); err == nil {
		cmd = exec.Command("uvx", "--with", "browser-cookie3", "python3", scriptPath)
	} else {
		cmd = exec.Command("python3", scriptPath)
	}
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return "", errors.New(strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	cookie := strings.TrimSpace(string(out))
	if cookie == "" {
		return "", fmt.Errorf("python cookie empty")
	}
	return normalizeCookieHeader(cookie), nil
}

func ensureCookieImportScript() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "bili")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "cookie_import.py")
	if data, err := os.ReadFile(path); err == nil && bytes.Contains(data, []byte("browser_cookie3")) {
		return path, nil
	}
	if err := os.WriteFile(path, []byte(cookieImportScript), 0644); err != nil {
		return "", err
	}
	return path, nil
}

const cookieImportScript = `import sys
try:
    import browser_cookie3
except Exception as e:
    print(f"IMPORT_ERROR:{e}", file=sys.stderr)
    sys.exit(2)
try:
    cookies = browser_cookie3.chrome(domain_name="bilibili.com")
except Exception as e:
    print(f"CHROME_ERROR:{e}", file=sys.stderr)
    sys.exit(3)
need = {"SESSDATA","bili_jct","DedeUserID","DedeUserID__ckMd5"}
pairs = []
for c in cookies:
    if c.name in need:
        pairs.append(f"{c.name}={c.value}")
if not pairs:
    sys.exit(4)
sys.stdout.write("; ".join(pairs))
`

func readChromeCookieWithPassword(password string, cookiesPath string, iterations int) (string, error) {
	if cookiesPath == "" {
		return "", fmt.Errorf("cookies path empty")
	}
	if _, err := os.Stat(cookiesPath); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp("", "bili_chrome_cookies_*.db")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := copyFile(cookiesPath, tmp.Name()); err != nil {
		return "", err
	}
	if iterations <= 0 {
		iterations = 1
	}
	key := pbkdf2.Key([]byte(password), []byte("saltysalt"), iterations, 16, sha1.New)
	rows, err := queryChromeCookies(tmp.Name())
	if err != nil {
		return "", err
	}
	needed := map[string]bool{
		"SESSDATA":          false,
		"bili_jct":          false,
		"DedeUserID":        false,
		"DedeUserID__ckMd5": false,
	}
	values := make(map[string]string)
	for _, row := range rows {
		if _, ok := needed[row.name]; !ok {
			continue
		}
		if row.value != "" {
			values[row.name] = sanitizeCookieValue(row.value)
		} else if row.encHex != "" {
			v, err := decryptChromeValue(row.encHex, key)
			if err == nil && v != "" {
				values[row.name] = sanitizeCookieValue(v)
			}
		}
	}
	for k := range needed {
		if values[k] == "" {
			return "", fmt.Errorf("cookie %s missing", k)
		}
		if !isSaneCookieValue(values[k]) {
			return "", fmt.Errorf("cookie %s invalid", k)
		}
	}
	parts := make([]string, 0, len(values))
	for k, v := range values {
		parts = append(parts, k+"="+v)
	}
	return normalizeCookieHeader(strings.Join(parts, "; ")), nil
}

func chromeCookieDBPaths(base string) []string {
	if base == "" {
		return nil
	}
	profileDirs := chromeProfileDirs(base)
	paths := make([]string, 0, len(profileDirs)*2)
	seen := make(map[string]bool)
	for _, dir := range profileDirs {
		if dir == "" {
			continue
		}
		for _, rel := range []string{"Network/Cookies", "Cookies"} {
			p := filepath.Join(dir, rel)
			if _, err := os.Stat(p); err == nil {
				if !seen[p] {
					paths = append(paths, p)
					seen[p] = true
				}
			}
		}
	}
	return paths
}

func chromePasswordCandidates() []string {
	candidates := []string{}
	for _, env := range []string{"CHROME_SAFE_STORAGE_PASSWORD", "BILI_CHROME_PASSWORD", "CHROME_PASSWORD"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			candidates = append(candidates, v)
		}
	}
	candidates = append(candidates, "peanuts", "")
	seen := make(map[string]bool, len(candidates))
	out := []string{}
	for _, v := range candidates {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func chromeLinuxBases(home string) []string {
	if home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, ".config", "google-chrome"),
		filepath.Join(home, ".config", "google-chrome-beta"),
		filepath.Join(home, ".config", "google-chrome-unstable"),
		filepath.Join(home, ".config", "chromium"),
		filepath.Join(home, ".config", "chromium-browser"),
	}
}

func chromeDarwinBases(home string) []string {
	if home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, "Library", "Application Support", "Google", "Chrome"),
		filepath.Join(home, "Library", "Application Support", "Google", "Chrome Beta"),
		filepath.Join(home, "Library", "Application Support", "Google", "Chrome Canary"),
		filepath.Join(home, "Library", "Application Support", "Chromium"),
	}
}

func chromeProfileDirs(base string) []string {
	if base == "" {
		return nil
	}
	paths := []string{}
	addDir := func(p string) {
		if p == "" {
			return
		}
		if _, err := os.Stat(p); err == nil {
			paths = append(paths, p)
		}
	}
	addDir(filepath.Join(base, "Default"))
	addDir(filepath.Join(base, "Guest Profile"))
	addDir(filepath.Join(base, "System Profile"))
	if matches, _ := filepath.Glob(filepath.Join(base, "Profile *")); len(matches) > 0 {
		for _, p := range matches {
			addDir(p)
		}
	}
	return paths
}

type chromeCookieRow struct {
	name   string
	value  string
	encHex string
}

func queryChromeCookies(dbPath string) ([]chromeCookieRow, error) {
	query := "select name, value, hex(encrypted_value) from cookies where host_key like '%bilibili.com' and name in ('SESSDATA','bili_jct','DedeUserID','DedeUserID__ckMd5');"
	cmd := exec.Command("sqlite3", "-readonly", dbPath, query)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	rows := make([]chromeCookieRow, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		rows = append(rows, chromeCookieRow{
			name:   parts[0],
			value:  parts[1],
			encHex: parts[2],
		})
	}
	return rows, nil
}

func chromeKeychainPassword() (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-w", "-s", "Chrome Safe Storage")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func decryptChromeValue(hexStr string, key []byte) (string, error) {
	blob, err := hex.DecodeString(hexStr)
	if err != nil || len(blob) == 0 {
		return "", fmt.Errorf("invalid encrypted value")
	}
	if bytes.HasPrefix(blob, []byte("v10")) || bytes.HasPrefix(blob, []byte("v11")) {
		blob = blob[3:]
	}
	iv := bytes.Repeat([]byte(" "), aes.BlockSize)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	if len(blob)%aes.BlockSize != 0 {
		return "", fmt.Errorf("invalid block size")
	}
	dst := make([]byte, len(blob))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(dst, blob)
	unpadded, err := pkcs7Unpad(dst)
	if err != nil {
		return "", err
	}
	return string(unpadded), nil
}

func pkcs7Unpad(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("empty")
	}
	pad := int(b[len(b)-1])
	if pad == 0 || pad > len(b) {
		return nil, fmt.Errorf("invalid padding")
	}
	for i := len(b) - pad; i < len(b); i++ {
		if int(b[i]) != pad {
			return nil, fmt.Errorf("invalid padding")
		}
	}
	return b[:len(b)-pad], nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
