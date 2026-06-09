package config

import (
	"flag"
	"fmt"
	"strings"
)

type ConfigType struct {
	Cookie       string // 登录cookie
	RoomId       int64  // 直播间id
	Theme        int64  // 主题
	SingleLine   int64  // 是否开启单行
	ShowTime     int64  // 是否显示时间
	TimeColor    string // 时间颜色
	NameColor    string // 名字颜色
	ContentColor string // 内容颜色
	FrameColor   string // 边框颜色
	InfoColor    string // 房间信息颜色
	RankColor    string // 排行榜颜色
	Background   string // 背景颜色
}

var Auth CookieAuth

var Config ConfigType

func Init() bool {
	var err error
	configFile := ""
	roomId := int64(-1)
	theme := int64(-1)
	single_line := int64(-1)
	show_time := int64(-1)
	flag.StringVar(&configFile, "c", "", "usage for config")
	flag.Int64Var(&roomId, "r", -1, "usage for room id")
	flag.Int64Var(&theme, "t", -1, "usage for theme")
	flag.Int64Var(&single_line, "l", -1, "usage for single_line")
	flag.Int64Var(&show_time, "s", -1, "usage for show_time")
	flag.Parse()

	if configFile == "" {
		configFile, err = defaultCfgFile()
		if err != nil {
			fmt.Println("Failed to load config:", err)
			return false
		}
	}

	if err := loadConfigFromFile(configFile); err != nil {
		fmt.Printf("Error decoding config.toml: %s\n", err)
		return false
	}
	if strings.TrimSpace(Config.Cookie) == "" || Config.Cookie == "从你BILIBILI的请求里抓一个Cookie" {
		if ok, errMsg := tryImportChromeCookie(configFile); ok {
			// Config.Cookie already updated and saved
		} else if errMsg != "" {
			fmt.Println("Chrome import failed:", errMsg)
		}
		if strings.TrimSpace(Config.Cookie) == "" || Config.Cookie == "从你BILIBILI的请求里抓一个Cookie" {
			return handleCookieFlow(configFile)
		}
	}

	if roomId != -1 {
		Config.RoomId = roomId
	}
	if theme != -1 {
		Config.Theme = theme
	}
	if single_line != -1 {
		Config.SingleLine = single_line
	}
	if show_time != -1 {
		Config.ShowTime = show_time
	}
	if Config.TimeColor == "" {
		Config.TimeColor = "#bbbbbb"
	}
	if Config.NameColor == "" {
		Config.NameColor = "#bbbbbb"
	}
	if Config.ContentColor == "" {
		Config.ContentColor = "#bbbbbb"
	}
	if Config.TimeColor == "" {
		Config.TimeColor = "#bbbbbb"
	}
	if Config.NameColor == "" {
		Config.NameColor = "#bbbbbb"
	}
	if Config.ContentColor == "" {
		Config.ContentColor = "#bbbbbb"
	}
	if Config.InfoColor == "" {
		Config.InfoColor = "#bbbbbb"
	}
	if Config.RankColor == "" {
		Config.RankColor = "#bbbbbb"
	}
	if Config.FrameColor == "" {
		Config.FrameColor = "#bbbbbb"
	}
	if Config.Background == "" {
		Config.Background = "NONE"
	}

	if ok, _ := setAuthFromCookieHeader(Config.Cookie); !ok {
		if ok, errMsg := tryImportChromeCookie(configFile); !ok && errMsg != "" {
			fmt.Println("Chrome import failed:", errMsg)
		}
		if ok, _ := setAuthFromCookieHeader(Config.Cookie); !ok {
			return handleCookieFlow(configFile)
		}
	}
	return true
}

func ValidateCookie(cookie string) (bool, error) {
	return validateCookie(cookie)
}
