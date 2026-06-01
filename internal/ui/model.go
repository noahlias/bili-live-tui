package ui

import (
	"fmt"
	_ "image/jpeg"
	"time"

	"github.com/noahlias/bili-live-tui/internal/config"
	"github.com/noahlias/bili-live-tui/internal/getter"
	"github.com/noahlias/bili-live-tui/internal/sender"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxLines                = 1200
	maxMessages             = 1200
	footerHeight            = 1
	sidebarMinW             = 24
	sidebarWideW            = 32
	panelGap                = 2
	bodyRightMargin         = 1
	panelBorderSize         = 2
	sidebarCoverRows        = 6
	maxRecentRooms          = 10
	compactMinWidth         = 100
	compactMinHeight        = 20
	maxSideLines            = 200
	defaultAudienceRows     = 9
	sidebarEventTail        = 2
	sidebarScrollW          = 1
	paneScrollStep          = 2
	maxInlineAvatarMessages = 48
	themeCount              = 6
	scrollbarWidth          = 1
	smoothScrollDelay       = 12 * time.Millisecond
	iconStatus              = "\uf111"
	iconViewers             = "\uf06e"
	iconFPS                 = "\uf0e4"
	iconClock               = "\uf017"
	entryToastTTL           = 4 * time.Second
)

type danmuMsg getter.DanmuMsg

type roomInfoMsg getter.RoomInfo

type userInfoMsg struct {
	uid       int64
	face      string
	topPhoto  string
	liveCover string
	name      string
	err       error
}

type roomSummaryMsg struct {
	roomID int64
	name   string
	face   string
	live   int64
	err    error
}

type roomStatusMsg struct {
	roomID int64
	live   int64
	err    error
}

type roomsRefreshMsg struct{}

type imageMsg struct {
	key  string
	data []byte
	path string
}

type editorFinishedMsg struct {
	err  error
	path string
}

type toastExpiredMsg struct {
	seq int
}

type smoothScrollMsg struct {
	target string
	step   int
}

type cookieValidatedMsg struct {
	ok  bool
	err error
}

type imageEntry struct {
	payload string
	path    string
}

type recentRoom struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	LiveStatus int64  `json:"live_status"`
}

type model struct {
	width             int
	height            int
	roomID            int64
	busChan           chan getter.DanmuMsg
	roomInfoChan      chan getter.RoomInfo
	viewport          viewport.Model
	input             textinput.Model
	lines             []string
	msgs              []getter.DanmuMsg
	msgLineIdx        []int
	accessLines       []string
	giftLines         []string
	roomInfo          getter.RoomInfo
	styles            styles
	kitty             bool
	pendingUID        map[int64]bool
	avatarKeyBy       map[int64]string
	imageCache        map[string]imageEntry
	streamerUID       int64
	headerAvatar      string
	nameToUID         map[string]int64
	sidebarCover      string
	stopGetter        func()
	recentRooms       []recentRoom
	roomAvatarKey     map[int64]string
	activeAudience    map[string]getter.AudienceUser
	pickerIdx         int
	focusPicker       bool
	focusAudience     bool
	selectMode        bool
	selectIdx         int
	selectRange       bool
	selectRangeStart  int
	selectRangeEnd    int
	yankBuffer        string
	commandMode       bool
	searchMode        bool
	searchQuery       string
	searchMatches     []int
	searchIdx         int
	searchDraft       string
	editorPath        string
	lastRenderLines   int
	clearImages       bool
	clearAllImages    bool
	clearImagesFrames int
	renderedImages    bool
	inlineChatAvatars bool
	sidebarOffset     int
	sidebarMaxOffset  int
	pendingMsgScroll  int
	pendingSideScroll int
	deleteConfirmID   int64
	toastText         string
	toastSeq          int
	focusMessages     bool
}

func Run() {
	busChan := make(chan getter.DanmuMsg, 200)
	roomInfoChan := make(chan getter.RoomInfo, 50)
	m := newModel(config.Config.RoomId, busChan, roomInfoChan)
	p := tea.NewProgram(&m, tea.WithAltScreen())
	if err := p.Start(); err != nil {
		fmt.Println("failed to start tui:", err)
	}
}

func newModel(roomID int64, busChan chan getter.DanmuMsg, roomInfoChan chan getter.RoomInfo) model {
	input := textinput.New()
	input.Placeholder = "Send a message"
	input.Prompt = "> "
	input.CharLimit = 120
	input.Focus()
	applyThemeToInput(&input)

	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = false

	kitty := supportsKittyGraphics()

	m := model{
		roomID:         roomID,
		busChan:        busChan,
		roomInfoChan:   roomInfoChan,
		input:          input,
		viewport:       vp,
		styles:         newStyles(),
		kitty:          kitty,
		pendingUID:     make(map[int64]bool),
		avatarKeyBy:    make(map[int64]string),
		imageCache:     make(map[string]imageEntry),
		nameToUID:      make(map[string]int64),
		roomAvatarKey:  make(map[int64]string),
		recentRooms:    loadRecentRooms(),
		activeAudience: make(map[string]getter.AudienceUser),
	}
	m.addRecentRoom(roomID)
	return m
}

func applyThemeToInput(input *textinput.Model) {
	palette := resolveThemePalette()
	input.PromptStyle = lipgloss.NewStyle().Foreground(palette.accent).Bold(true)
	input.TextStyle = lipgloss.NewStyle().Foreground(palette.text)
	input.PlaceholderStyle = lipgloss.NewStyle().Foreground(palette.muted)
	input.Cursor.Style = lipgloss.NewStyle().
		Foreground(palette.cursorFg).
		Background(palette.cursorBg).
		Bold(true)
	input.Cursor.TextStyle = input.TextStyle
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		startServicesCmd(m.roomID, m.busChan, m.roomInfoChan),
		listenDanmuCmd(m.busChan),
		listenRoomInfoCmd(m.roomInfoChan),
		m.focusInput(),
		validateCookieCmd(),
	}
	for _, r := range m.recentRooms {
		if m.kitty {
			if m.loadRoomAvatar(r.ID) {
				continue
			}
		}
		if r.Name == "" || m.kitty {
			cmds = append(cmds, fetchRoomSummaryCmd(r.ID))
		}
	}
	cmds = append(cmds, refreshRoomsTickCmd())
	return tea.Batch(cmds...)
}

type startServicesMsg struct {
	stop func()
}

func startServicesCmd(roomID int64, busChan chan getter.DanmuMsg, roomInfoChan chan getter.RoomInfo) tea.Cmd {
	return func() tea.Msg {
		stop := getter.RunWithRoom(roomID, busChan, roomInfoChan)
		sender.Run()
		return startServicesMsg{stop: stop}
	}
}

func listenDanmuCmd(ch <-chan getter.DanmuMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return danmuMsg(msg)
	}
}

func listenRoomInfoCmd(ch <-chan getter.RoomInfo) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return roomInfoMsg(msg)
	}
}
