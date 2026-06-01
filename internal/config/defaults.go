package config

import (
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
)

func defaultCfgFile() (configFile string, err error) {
	currentUser, err := user.Current()
	if err != nil {
		return
	}
	homeDir := currentUser.HomeDir
	path := homeDir + "/.config/bili"
	if err = os.MkdirAll(path, 0755); err != nil {
		return
	}
	configFile = path + "/config.toml"
	_, err = os.Stat(configFile)
	if os.IsNotExist(err) {
		var f *os.File
		config := ConfigType{
			Cookie:       "从你BILIBILI的请求里抓一个Cookie",
			RoomId:       23333333,
			Theme:        1,
			SingleLine:   1,
			ShowTime:     1,
			TimeColor:    "#FFFFFF",
			NameColor:    "#FFFFFF",
			ContentColor: "#FFFFFF",
			FrameColor:   "#FFFFFF",
			InfoColor:    "#FFFFFF",
			RankColor:    "#FFFFFF",
			Background:   "NONE", // 默认无背景颜色 NONE表示无背景颜色
		}
		f, err = os.Create(configFile)
		if err != nil {
			return
		}
		defer f.Close()
		if err = toml.NewEncoder(f).Encode(config); err != nil {
			return
		}

		fmt.Println("配置文件已生成，请修改配置文件后再次运行，配置文件路径为：" + configFile)
	}

	return
}

func saveConfig(configFile string) error {
	if configFile == "" {
		return fmt.Errorf("config file path empty")
	}
	dir := filepath.Dir(configFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(Config); err != nil {
		return err
	}
	return os.WriteFile(configFile, buf.Bytes(), 0644)
}

func configHasCookie(configFile string) bool {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "Cookie =")
}

func loadConfigFromFile(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	if !utf8.Valid(data) {
		cleaned := bytes.ToValidUTF8(data, []byte{})
		cleaned = scrubCookieLine(cleaned)
		if err := os.WriteFile(configFile, cleaned, 0644); err != nil {
			return err
		}
		data = cleaned
	}
	if _, err := toml.Decode(string(data), &Config); err != nil {
		return err
	}
	if Config.Cookie != "" && !isSaneCookieValue(Config.Cookie) {
		Config.Cookie = ""
		_ = saveConfig(configFile)
	}
	return nil
}

func scrubCookieLine(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Cookie") {
			lines[i] = "Cookie = \"\""
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
