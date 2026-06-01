package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/noahlias/bili-live-tui/internal/config"

	"github.com/noahlias/bili-live-tui/internal/getter"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestInputModePlainLettersUpdateInput(t *testing.T) {
	keyCases := []struct {
		name string
		key  tea.KeyMsg
		want string
	}{
		{name: "e", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}, want: "e"},
		{name: "v", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}, want: "v"},
		{name: "q", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, want: "q"},
		{name: "j", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, want: "j"},
		{name: "k", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, want: "k"},
	}

	for _, tc := range keyCases {
		t.Run(tc.name, func(t *testing.T) {
			m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
			m.kitty = false

			gotModel, _ := m.Update(tc.key)
			got := gotModel.(*model).input.Value()
			if got != tc.want {
				t.Fatalf("input value = %q, want %q", got, tc.want)
			}
			if gotModel.(*model).focusPicker {
				t.Fatalf("focusPicker = true, want false")
			}
		})
	}
}

func TestInputModeTabMovesFocusToPicker(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false

	gotModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	got := gotModel.(*model)
	if !got.focusPicker {
		t.Fatalf("focusPicker = false, want true")
	}
	if got.focusAudience {
		t.Fatalf("focusAudience = true, want false")
	}
	if got.focusMessages {
		t.Fatalf("focusMessages = true, want false")
	}
}

func TestInputCursorUsesLightBlockPalette(t *testing.T) {
	prevTheme := config.Config.Theme
	config.Config.Theme = 1
	defer func() { config.Config.Theme = prevTheme }()
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	if got := m.input.Cursor.Style.GetForeground(); got == nil || fmt.Sprint(got) != "#E8F6FF" {
		t.Fatalf("cursor foreground = %v, want #E8F6FF", got)
	}
	if got := m.input.Cursor.Style.GetBackground(); got == nil || fmt.Sprint(got) != "#1A1B26" {
		t.Fatalf("cursor background = %v, want #1A1B26", got)
	}
}

func TestNewStylesUsesLightThemePalette(t *testing.T) {
	prevTheme := config.Config.Theme
	config.Config.Theme = 6
	defer func() { config.Config.Theme = prevTheme }()

	s := newStyles()
	if got := s.panel.GetBackground(); got == nil || fmt.Sprint(got) != "#F3F3F3" {
		t.Fatalf("panel background = %v, want #F3F3F3", got)
	}
	if got := s.content.GetForeground(); got == nil || fmt.Sprint(got) != "#1F2328" {
		t.Fatalf("content foreground = %v, want #1F2328", got)
	}
}

func TestCtrlTCyclesThemeAtRuntime(t *testing.T) {
	prevTheme := config.Config.Theme
	config.Config.Theme = 1
	defer func() { config.Config.Theme = prevTheme }()

	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	gotModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	got := gotModel.(*model)

	if config.Config.Theme != 2 {
		t.Fatalf("config theme = %d, want 2", config.Config.Theme)
	}
	if bg := got.styles.panel.GetBackground(); bg == nil || fmt.Sprint(bg) != "#1E1E2E" {
		t.Fatalf("panel background = %v, want catppuccin mocha background", bg)
	}
	if cursorBg := got.input.Cursor.Style.GetBackground(); cursorBg == nil || fmt.Sprint(cursorBg) != "#1E1E2E" {
		t.Fatalf("cursor background = %v, want updated theme cursor background", cursorBg)
	}
}

func TestStartMessageSmoothScrollSchedulesAnimation(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	cmd := m.startMessageSmoothScroll(2)
	if cmd == nil {
		t.Fatal("cmd = nil, want animation cmd")
	}
	if m.pendingMsgScroll != 2 {
		t.Fatalf("pendingMsgScroll = %d, want 2", m.pendingMsgScroll)
	}
}

func TestHandleSmoothAudienceScrollMovesOffset(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.sidebarMaxOffset = 10
	m.pendingSideScroll = 2

	gotModel, cmd := m.handleSmoothScroll(smoothScrollMsg{target: "audience", step: 1})
	got := gotModel.(*model)
	if got.sidebarOffset != 1 {
		t.Fatalf("sidebarOffset = %d, want 1", got.sidebarOffset)
	}
	if got.pendingSideScroll != 1 {
		t.Fatalf("pendingSideScroll = %d, want 1", got.pendingSideScroll)
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want follow-up animation tick")
	}
}

func TestRoomsFocusJMovesRoomSelection(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.focusPicker = true
	m.recentRooms = []recentRoom{{ID: 1}, {ID: 2}, {ID: 3}}

	gotModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := gotModel.(*model)
	if got.pickerIdx != 1 {
		t.Fatalf("pickerIdx = %d, want 1", got.pickerIdx)
	}
}

func TestTabCyclesRoomsAudienceMessagesInput(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false

	gotModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	got := gotModel.(*model)
	if !got.focusPicker || got.focusAudience || got.focusMessages {
		t.Fatalf("after first tab: focusPicker=%v focusAudience=%v focusMessages=%v", got.focusPicker, got.focusAudience, got.focusMessages)
	}

	gotModel, _ = got.Update(tea.KeyMsg{Type: tea.KeyTab})
	got = gotModel.(*model)
	if !got.focusPicker || !got.focusAudience || got.focusMessages {
		t.Fatalf("after second tab: focusPicker=%v focusAudience=%v focusMessages=%v", got.focusPicker, got.focusAudience, got.focusMessages)
	}

	gotModel, _ = got.Update(tea.KeyMsg{Type: tea.KeyTab})
	got = gotModel.(*model)
	if got.focusPicker || !got.focusMessages {
		t.Fatalf("after third tab: focusPicker=%v focusMessages=%v", got.focusPicker, got.focusMessages)
	}

	gotModel, _ = got.Update(tea.KeyMsg{Type: tea.KeyTab})
	got = gotModel.(*model)
	if got.focusPicker || got.focusAudience || got.focusMessages {
		t.Fatalf("after fourth tab: expected input focus state, got focusPicker=%v focusAudience=%v focusMessages=%v", got.focusPicker, got.focusAudience, got.focusMessages)
	}
}

func TestSelectModeJMovesSelection(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.msgs = []getter.DanmuMsg{
		{Author: "a", Content: "one"},
		{Author: "b", Content: "two"},
	}
	if !m.enterSelectMode() {
		t.Fatal("enterSelectMode = false, want true")
	}

	gotModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	got := gotModel.(*model)
	if got.selectIdx != 0 {
		t.Fatalf("selectIdx = %d, want 0", got.selectIdx)
	}
}

func TestEnterInteractionShowsTemporaryToast(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 30
	m.resize()

	gotModel, _ := m.Update(danmuMsg(getter.DanmuMsg{
		Author:  "alice",
		Content: "进入了房间",
		Type:    "INTERACT_WORD",
	}))
	got := gotModel.(*model)
	if got.toastText != "alice 进入了房间" {
		t.Fatalf("toastText = %q, want %q", got.toastText, "alice 进入了房间")
	}
	if len(got.msgs) != 0 {
		t.Fatalf("msgs len = %d, want 0", len(got.msgs))
	}
	if len(got.accessLines) != 1 {
		t.Fatalf("accessLines len = %d, want 1", len(got.accessLines))
	}

	gotModel, _ = got.Update(toastExpiredMsg{seq: got.toastSeq})
	got = gotModel.(*model)
	if got.toastText != "" {
		t.Fatalf("toastText = %q, want empty", got.toastText)
	}
}

func TestFollowInteractionDoesNotShowEntryToast(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 30
	m.resize()

	gotModel, _ := m.Update(danmuMsg(getter.DanmuMsg{
		Author:  "alice",
		Content: "关注了主播",
		Type:    "INTERACT_WORD",
	}))
	got := gotModel.(*model)
	if got.toastText != "" {
		t.Fatalf("toastText = %q, want empty", got.toastText)
	}
	if len(got.accessLines) != 1 {
		t.Fatalf("accessLines len = %d, want 1", len(got.accessLines))
	}
}

func TestCookieValidatedMsgAppendsSystemNotice(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 28
	m.resize()

	gotModel, _ := m.Update(cookieValidatedMsg{ok: false})
	got := gotModel.(*model)
	if len(got.msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(got.msgs))
	}
	if got.msgs[0].Author != "system" {
		t.Fatalf("author = %q, want system", got.msgs[0].Author)
	}
}

func TestRenderMessagesGroupsRepeatedSpeakerWithoutRepeatedHeader(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 28
	m.resize()
	m.msgs = []getter.DanmuMsg{
		{Author: "alice", Content: "hello", Type: "DANMU_MSG", Time: time.Date(2026, 3, 16, 12, 0, 0, 0, time.Local)},
		{Author: "alice", Content: "again", Type: "DANMU_MSG", Time: time.Date(2026, 3, 16, 12, 0, 1, 0, time.Local)},
		{Author: "bob", Content: "hi", Type: "DANMU_MSG", Time: time.Date(2026, 3, 16, 12, 0, 2, 0, time.Local)},
	}

	m.renderMessages()
	rendered := strings.Join(m.lines, "\n")
	if strings.Count(rendered, "alice") != 1 {
		t.Fatalf("rendered = %q, want alice header only once", rendered)
	}
}

func TestFormatMessageRendersEventAsSingleLine(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	lines := m.formatMessage(getter.DanmuMsg{}, getter.DanmuMsg{
		Author:  "system",
		Content: "Cookie appears invalid",
		Type:    "NOTICE_MSG",
		Time:    time.Now(),
	}, false)
	if len(lines) != 1 {
		t.Fatalf("len(lines) = %d, want 1 for event/system message", len(lines))
	}
}

func TestRenderMessagesKeepsSameSpeakerGroupedAcrossMinuteBoundary(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 28
	m.resize()
	m.msgs = []getter.DanmuMsg{
		{Author: "alice", Content: "hello", Type: "DANMU_MSG", Time: time.Date(2026, 3, 16, 12, 0, 59, 0, time.Local)},
		{Author: "alice", Content: "again", Type: "DANMU_MSG", Time: time.Date(2026, 3, 16, 12, 1, 1, 0, time.Local)},
	}

	m.renderMessages()
	rendered := strings.Join(m.lines, "\n")
	if strings.Count(rendered, "alice") != 1 {
		t.Fatalf("rendered = %q, want alice header only once across minute boundary", rendered)
	}
}

func TestRenderHeaderAvoidsFooterStyleStatusDuplication(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.roomInfo = getter.RoomInfo{
		RoomId:     1,
		Title:      "demo room",
		LiveStatus: 1,
		Online:     123,
		Attention:  45,
		Time:       "01:23:45",
	}

	header := m.renderHeader()
	if header == "" {
		t.Fatalf("header = %q, want non-empty title bar", header)
	}
	if w := lipgloss.Width(header); w != m.width {
		t.Fatalf("header width = %d, want full width %d", w, m.width)
	}
	if !strings.Contains(header, "demo room") {
		t.Fatalf("header = %q, want room title", header)
	}
	if !strings.Contains(header, "LIVE") {
		t.Fatalf("header = %q, want live badge", header)
	}
	for _, seg := range []string{"API", "WS", "AUD"} {
		if strings.Contains(header, seg) {
			t.Fatalf("header = %q, want no footer status segment %q", header, seg)
		}
	}
}

func TestRenderSidebarShowsAudienceTotalAndRoster(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 28
	m.roomInfo = getter.RoomInfo{
		RoomId:          1,
		Online:          321,
		OnlineRankTotal: 24,
		Title:           "demo",
		Attention:       12,
		AudienceUsers: []getter.AudienceUser{
			{UID: 1, Name: "alice", Rank: 1, Score: 66, MedalName: "糊弄", MedalLevel: 26, GuardLevel: 3, WealthLevel: 27},
			{UID: 2, Name: "bob", Rank: 2, Score: 12},
			{UID: 3, Name: "carol"},
		},
	}
	m.accessLines = []string{"10:00 visitor 进入了房间"}

	sidebar := m.renderSidebar(28)
	if !strings.Contains(sidebar, "Audience") {
		t.Fatalf("sidebar = %q, want Audience block", sidebar)
	}
	if !strings.Contains(sidebar, "24") {
		t.Fatalf("sidebar = %q, want audience total from audience source", sidebar)
	}
	if !strings.Contains(sidebar, "Heat") || !strings.Contains(sidebar, "321") {
		t.Fatalf("sidebar = %q, want heat shown separately from audience total", sidebar)
	}
	if !strings.Contains(sidebar, "alice") || !strings.Contains(sidebar, "66") {
		t.Fatalf("sidebar = %q, want ranked audience row with score", sidebar)
	}
	if !strings.Contains(sidebar, "糊弄26") || !strings.Contains(sidebar, "G3") || !strings.Contains(sidebar, "W27") {
		t.Fatalf("sidebar = %q, want compact audience decoration badges", sidebar)
	}
	if !strings.Contains(sidebar, "bob") || !strings.Contains(sidebar, "12") {
		t.Fatalf("sidebar = %q, want merged audience row with score", sidebar)
	}
	if !strings.Contains(sidebar, "visitor") {
		t.Fatalf("sidebar = %q, want access block preserved", sidebar)
	}
}

func TestRenderAudienceLineShowsCompactMeta(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	line := m.renderAudienceLine(1, getter.AudienceUser{
		Name:        "alice",
		Score:       66,
		MedalName:   "糊弄",
		MedalLevel:  26,
		GuardLevel:  3,
		WealthLevel: 27,
	}, 32)
	if !strings.Contains(line, "糊弄26") || !strings.Contains(line, "G3") || !strings.Contains(line, "W27") {
		t.Fatalf("line = %q, want compact audience metadata", line)
	}
	if !strings.Contains(line, "1 ") {
		t.Fatalf("line = %q, want numbered audience prefix", line)
	}
	if !strings.Contains(line, "66") {
		t.Fatalf("line = %q, want score", line)
	}
}

func TestRenderSidebarShowsCoverWhenKittyEnabled(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = true
	m.sidebarCover = "cover"
	m.imageCache["cover"] = imageEntry{payload: "abc", path: "/tmp/cover.png"}
	m.roomInfo = getter.RoomInfo{RoomId: 1, Title: "demo"}

	sidebar := m.renderSidebar(28)
	if !strings.Contains(sidebar, "\x1b_Ga=T") {
		t.Fatalf("sidebar = %q, want kitty cover sequence", sidebar)
	}
}

func TestRenderSidebarHidesAudienceRosterWhenOnlineIsZero(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.roomInfo = getter.RoomInfo{
		RoomId:          1,
		Online:          0,
		OnlineRankTotal: 0,
	}
	m.activeAudience["uid:2"] = getter.AudienceUser{UID: 2, Name: "bob", Score: 10}

	sidebar := m.renderSidebar(16)
	if !strings.Contains(sidebar, "Online  0") {
		t.Fatalf("sidebar = %q, want online total line", sidebar)
	}
	if !strings.Contains(sidebar, "No audience data") {
		t.Fatalf("sidebar = %q, want empty audience state", sidebar)
	}
	if strings.Contains(sidebar, "█") {
		t.Fatalf("sidebar = %q, want no audience scrollbar thumb in empty state", sidebar)
	}
	if strings.Contains(sidebar, "bob") {
		t.Fatalf("sidebar = %q, want no websocket-derived roster rows when online is zero", sidebar)
	}
}

func TestRenderSidebarShowsScrollbarWhenOverflowing(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 28
	m.roomInfo = getter.RoomInfo{RoomId: 1, OnlineRankTotal: 30}
	for i := 0; i < 30; i++ {
		m.roomInfo.AudienceUsers = append(m.roomInfo.AudienceUsers, getter.AudienceUser{UID: int64(i), Name: fmt.Sprintf("user-%02d", i), Score: int64(100 - i)})
	}

	sidebar := m.renderSidebar(28)
	if !strings.Contains(sidebar, "█") {
		t.Fatalf("sidebar = %q, want sidebar scrollbar thumb", sidebar)
	}
	if !strings.Contains(sidebar, "Access") {
		t.Fatalf("sidebar = %q, want access section still visible with capped audience viewport", sidebar)
	}
}

func TestAudienceDefaultViewportShowsTopFive(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 28
	m.roomInfo = getter.RoomInfo{RoomId: 1, OnlineRankTotal: 10}
	for i := 0; i < 10; i++ {
		m.roomInfo.AudienceUsers = append(m.roomInfo.AudienceUsers, getter.AudienceUser{
			UID:   int64(i),
			Name:  fmt.Sprintf("user-%02d", i),
			Score: int64(100 - i),
		})
	}

	sidebar := m.renderSidebar(28)
	for i := 0; i < 5; i++ {
		if !strings.Contains(sidebar, fmt.Sprintf("user-%02d", i)) {
			t.Fatalf("sidebar = %q, want top five audience user-%02d visible", sidebar, i)
		}
	}
	if strings.Contains(sidebar, "user-05") {
		t.Fatalf("sidebar = %q, want sixth audience row hidden by default viewport", sidebar)
	}
}

func TestRoomsFocusJDoesNotScrollAudience(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 28
	m.focusPicker = true
	m.recentRooms = []recentRoom{{ID: 1}, {ID: 2}, {ID: 3}}
	m.roomInfo = getter.RoomInfo{RoomId: 1, Online: 100, OnlineRankTotal: 30}
	for i := 0; i < 30; i++ {
		m.roomInfo.AudienceUsers = append(m.roomInfo.AudienceUsers, getter.AudienceUser{UID: int64(i), Name: fmt.Sprintf("user-%02d", i), Score: int64(100 - i)})
	}
	_ = m.renderSidebar(20)

	gotModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := gotModel.(*model)
	if got.sidebarOffset != 0 {
		t.Fatalf("sidebarOffset = %d, want unchanged 0 without audience focus", got.sidebarOffset)
	}
	if got.pickerIdx != 1 {
		t.Fatalf("pickerIdx = %d, want moved to 1 in rooms focus", got.pickerIdx)
	}
}

func TestAudienceFocusJScrollsAudienceOnly(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 28
	m.focusPicker = true
	m.focusAudience = true
	m.recentRooms = []recentRoom{{ID: 1}, {ID: 2}, {ID: 3}}
	m.roomInfo = getter.RoomInfo{RoomId: 1, Online: 100, OnlineRankTotal: 30}
	for i := 0; i < 30; i++ {
		m.roomInfo.AudienceUsers = append(m.roomInfo.AudienceUsers, getter.AudienceUser{UID: int64(i), Name: fmt.Sprintf("user-%02d", i), Score: int64(100 - i)})
	}
	_ = m.renderSidebar(20)

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := gotModel.(*model)
	if got.sidebarOffset != 0 {
		t.Fatalf("sidebarOffset = %d, want initial offset unchanged before animation tick", got.sidebarOffset)
	}
	if got.pendingSideScroll <= 0 {
		t.Fatalf("pendingSideScroll = %d, want > 0 after smooth scroll start", got.pendingSideScroll)
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want animation tick cmd")
	}
	gotModel, _ = got.handleSmoothScroll(smoothScrollMsg{target: "audience", step: 1})
	got = gotModel.(*model)
	if got.sidebarOffset <= 0 {
		t.Fatalf("sidebarOffset = %d, want > 0 after animation tick", got.sidebarOffset)
	}
	if got.pickerIdx != 0 {
		t.Fatalf("pickerIdx = %d, want unchanged 0", got.pickerIdx)
	}
}

func TestRenderFooterUsesAudienceTotalNotHeat(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.width = 120
	m.roomInfo = getter.RoomInfo{RoomId: 1, Online: 8000, OnlineRankTotal: 24, APILatencyMs: 123, WSLatencyMs: 45}

	footer := m.renderFooter()
	if !strings.Contains(footer, "AUD 24") {
		t.Fatalf("footer = %q, want audience total", footer)
	}
	if strings.Contains(footer, "8000") {
		t.Fatalf("footer = %q, want no heat value in audience footer segment", footer)
	}
	if !strings.Contains(footer, "API 123ms") || !strings.Contains(footer, "WS 45ms") {
		t.Fatalf("footer = %q, want API and WS latency metrics", footer)
	}
}

func TestRenderFooterShowsLiveDurationWhenRoomIsLive(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.width = 120
	m.roomInfo = getter.RoomInfo{
		RoomId:          1,
		LiveStatus:      1,
		Time:            "2时15分",
		OnlineRankTotal: 24,
	}

	footer := m.renderFooter()
	if !strings.Contains(footer, "UP 2时15分") {
		t.Fatalf("footer = %q, want live duration segment", footer)
	}
}

func TestRenderFooterOmitsLiveDurationWhenRoomIsOffline(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.width = 120
	m.roomInfo = getter.RoomInfo{
		RoomId:     1,
		LiveStatus: 0,
		Time:       "2时15分",
	}

	footer := m.renderFooter()
	if strings.Contains(footer, "UP 2时15分") {
		t.Fatalf("footer = %q, want no live duration segment for offline room", footer)
	}
}

func TestRenderFooterOmitsLiveDurationWhenDurationIsMissing(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.width = 120
	m.roomInfo = getter.RoomInfo{
		RoomId:     1,
		LiveStatus: 1,
		Time:       "",
	}

	footer := m.renderFooter()
	if strings.Contains(footer, "UP ") {
		t.Fatalf("footer = %q, want no live duration segment when duration is missing", footer)
	}
}

func TestSidebarAudienceUsersSortDeterministically(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.roomInfo.AudienceUsers = []getter.AudienceUser{
		{UID: 2, Name: "bob", Score: 10},
		{UID: 1, Name: "alice", Rank: 2, Score: 20},
		{UID: 3, Name: "carol", Rank: 1, Score: 5},
		{UID: 4, Name: "dave", Score: 30},
	}

	users := m.sidebarAudienceUsers()
	if len(users) != 4 {
		t.Fatalf("len(users) = %d, want 4", len(users))
	}
	if users[0].Name != "carol" || users[1].Name != "alice" || users[2].Name != "dave" || users[3].Name != "bob" {
		t.Fatalf("sorted names = %q, %q, %q, %q", users[0].Name, users[1].Name, users[2].Name, users[3].Name)
	}
}

func TestRenderRoomsLinesStayTextOnlyWithKittyEnabled(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = true
	key := roomAvatarKey(1)
	m.roomAvatarKey[1] = key
	m.imageCache[key] = imageEntry{payload: "abc", path: "/tmp/a.png"}
	m.recentRooms = []recentRoom{{ID: 1, Name: "demo"}}

	lines := m.renderRoomsLines(24)
	if len(lines) != 1 {
		t.Fatalf("len(lines) = %d, want 1", len(lines))
	}
	if strings.Contains(lines[0], "\x1b_Ga=T") || strings.Contains(lines[0], "\x1b\\") {
		t.Fatalf("line = %q, want no kitty image protocol in room list", lines[0])
	}
}

func TestApplySearchAndJumpSearch(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 20
	m.resize()
	m.msgs = []getter.DanmuMsg{
		{Author: "a", Content: "hello"},
		{Author: "b", Content: "find me"},
		{Author: "c", Content: "nothing"},
		{Author: "d", Content: "find me too"},
	}
	m.renderMessages()

	m.applySearch("find")
	if len(m.searchMatches) != 2 {
		t.Fatalf("len(searchMatches) = %d, want 2", len(m.searchMatches))
	}
	if !m.isSearchCurrent(1) {
		t.Fatal("expected first search hit at index 1")
	}

	m.jumpSearch(1)
	if !m.isSearchCurrent(3) {
		t.Fatal("expected second search hit at index 3")
	}
}

func TestRenderViewportScrollbarShowsThumb(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.viewport.Height = 5
	m.lines = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	m.viewport.YOffset = 2

	bar := m.renderViewportScrollbar()
	if !strings.Contains(bar, "▌") {
		t.Fatalf("scrollbar = %q, want thumb", bar)
	}
}

func TestRenderViewportPaneUsesInnerCard(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 30
	m.resize()
	m.lines = []string{"one", "two", "three"}
	m.viewport.SetContent(strings.Join(m.lines, "\n"))

	pane := m.renderViewportPane()
	if !strings.Contains(pane, "╭") || !strings.Contains(pane, "╮") {
		t.Fatalf("pane = %q, want rounded inner card border", pane)
	}
}

func TestRenderMessagesKeepsInlineAvatarsForShortBuffer(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = true
	m.avatarKeyBy[1] = "avatar"
	m.imageCache["avatar"] = imageEntry{payload: "abc", path: "/tmp/avatar.png"}
	m.msgs = []getter.DanmuMsg{{UID: 1, Author: "alice", Content: "hello", Type: "DANMU_MSG"}}

	m.renderMessages()
	if !m.inlineChatAvatars {
		t.Fatal("inlineChatAvatars = false, want true for short buffer")
	}
	if !strings.Contains(strings.Join(m.lines, "\n"), "\x1b_Ga=T") {
		t.Fatalf("lines = %q, want kitty image sequence for short buffer", strings.Join(m.lines, "\n"))
	}
}

func TestRenderMessagesDisablesInlineAvatarsForLongBuffer(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = true
	m.inlineChatAvatars = true
	m.renderedImages = true
	m.avatarKeyBy[1] = "avatar"
	m.imageCache["avatar"] = imageEntry{payload: "abc", path: "/tmp/avatar.png"}
	for i := 0; i < maxInlineAvatarMessages+1; i++ {
		m.msgs = append(m.msgs, getter.DanmuMsg{UID: 1, Author: "alice", Content: "hello", Type: "DANMU_MSG"})
	}

	m.renderMessages()
	joined := strings.Join(m.lines, "\n")
	if m.inlineChatAvatars {
		t.Fatal("inlineChatAvatars = true, want false for long buffer")
	}
	if strings.Contains(joined, "\x1b_Ga=T") {
		t.Fatalf("lines = %q, want no kitty avatar sequence for long buffer", joined)
	}
	if !strings.Contains(joined, "\x1b_Ga=d,d=A,q=1") {
		t.Fatalf("lines = %q, want kitty cleanup sequence when disabling avatars", joined)
	}
}

func TestFocusInputStartsCursorBlink(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.input.Cursor.BlinkSpeed = 0

	cmd := m.focusInput()
	if cmd == nil {
		t.Fatal("focusInput returned nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(cursor.BlinkMsg); !ok {
		t.Fatalf("focus cmd msg = %T, want cursor.BlinkMsg", msg)
	}

	gotModel, nextCmd := m.Update(msg)
	_ = gotModel.(*model)
	if nextCmd == nil {
		t.Fatal("cursor blink update returned nil next cmd")
	}
}

func TestLayoutLeavesRightMarginForBody(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.width = 120
	m.height = 30

	chatW, sideW, _, _, _ := m.layout()
	if got := chatW + sideW + panelGap + bodyRightMargin + panelBorderSize*2; got != m.width {
		t.Fatalf("layout width sum = %d, want %d", got, m.width)
	}
}

func TestRenderBodyFitsTerminalWidth(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 28
	m.roomInfo = getter.RoomInfo{RoomId: 1, Online: 12}
	m.resize()
	m.msgs = []getter.DanmuMsg{
		{Author: "alice", Content: "hello"},
		{Author: "bob", Content: "world"},
	}
	m.renderMessages()

	body := m.renderBody()
	for _, line := range strings.Split(body, "\n") {
		if lipgloss.Width(line) > m.width {
			t.Fatalf("line width = %d, want <= %d: %q", lipgloss.Width(line), m.width, line)
		}
	}
}

func TestRenderBodyMatchesAvailableHeight(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = false
	m.width = 120
	m.height = 28
	m.roomInfo = getter.RoomInfo{RoomId: 1, Online: 12}
	m.resize()
	m.msgs = []getter.DanmuMsg{
		{Author: "alice", Content: "hello"},
		{Author: "bob", Content: "world"},
	}
	m.renderMessages()

	body := m.renderBody()
	expected := m.height - m.headerHeight() - footerHeight
	if got := lipgloss.Height(body); got != expected {
		t.Fatalf("body height = %d, want %d", got, expected)
	}
}

func TestViewDoesNotEmitImageCleanupWithoutRenderedImages(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = true
	m.width = 120
	m.height = 28
	m.clearAllImages = true

	view := m.View()
	if strings.Contains(view, "\x1b_Ga=d,d=A,q=1") {
		t.Fatalf("view = %q, want no kitty cleanup sequence without rendered images", view)
	}
}

func TestViewEmitsImageCleanupAfterRenderedImages(t *testing.T) {
	m := newModel(1, make(chan getter.DanmuMsg, 1), make(chan getter.RoomInfo, 1))
	m.kitty = true
	m.width = 120
	m.height = 28
	m.clearAllImages = true
	m.renderedImages = true

	view := m.View()
	if !strings.Contains(view, "\x1b_Ga=d,d=A,q=1") {
		t.Fatalf("view = %q, want kitty cleanup sequence after rendered images", view)
	}
}
