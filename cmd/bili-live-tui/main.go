package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/noahlias/bili-live-tui/internal/config"
	"github.com/noahlias/bili-live-tui/internal/ui"
)

/** 用于修正环境变量 */
func fixCharset() {
	locale := os.Getenv("LANG")
	var asianCharset bool
	var wideCharset = []string{"zh_", "jp_", "ko_", "ja_", "th_", "hi_"}
	for k := range wideCharset {
		if strings.HasPrefix(locale, wideCharset[k]) {
			asianCharset = true
		}
	}
	if asianCharset {
		os.Setenv("LANG", "C.UTF-8")
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		os.Exit(0)
	}
}

func main() {
	fixCharset()
	if !config.Init() {
		return
	}
	ui.Run()
}
