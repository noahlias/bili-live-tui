package ui

import (
	_ "image/jpeg"
	"strings"

	"github.com/noahlias/bili-live-tui/internal/config"

	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	headerBar       lipgloss.Style
	headerSegment   lipgloss.Style
	headerSegmentHi lipgloss.Style
	headerHint      lipgloss.Style
	headerTitle     lipgloss.Style
	headerMeta      lipgloss.Style
	headerBadgeLive lipgloss.Style
	headerBadgeOff  lipgloss.Style
	panel           lipgloss.Style
	sidePanel       lipgloss.Style
	messageCard     lipgloss.Style
	messageCardHi   lipgloss.Style
	sideSection     lipgloss.Style
	sideSectionHi   lipgloss.Style
	inputBox        lipgloss.Style
	inputBoxHi      lipgloss.Style
	inputInner      lipgloss.Style
	statusBar       lipgloss.Style
	statusKey       lipgloss.Style
	statusSeg       lipgloss.Style
	statusSegFocus  lipgloss.Style
	statusSegLive   lipgloss.Style
	statusSegOff    lipgloss.Style
	statusSegMuted  lipgloss.Style
	statusSegWarn   lipgloss.Style
	toastTag        lipgloss.Style
	toastText       lipgloss.Style
	scrollTrack     lipgloss.Style
	scrollThumb     lipgloss.Style
	label           lipgloss.Style
	time            lipgloss.Style
	name            lipgloss.Style
	content         lipgloss.Style
	system          lipgloss.Style
	eventGift       lipgloss.Style
	eventLive       lipgloss.Style
	msgGutter       lipgloss.Style
	msgGutterHi     lipgloss.Style
	messageSelected lipgloss.Style
	rankName        lipgloss.Style
	roomName        lipgloss.Style
	roomSelected    lipgloss.Style
	roomOffline     lipgloss.Style
	roomLiveStatus  lipgloss.Style
	roomOffStatus   lipgloss.Style
	badgeLive       lipgloss.Style
	badgeOff        lipgloss.Style
	muted           lipgloss.Style
}

type themePalette struct {
	panelBg       lipgloss.Color
	surfaceBg     lipgloss.Color
	activeBg      lipgloss.Color
	keyBg         lipgloss.Color
	focusBg       lipgloss.Color
	panelBorder   lipgloss.Color
	sideBorder    lipgloss.Color
	messageBorder lipgloss.Color
	accent        lipgloss.Color
	cta           lipgloss.Color
	muted         lipgloss.Color
	system        lipgloss.Color
	hint          lipgloss.Color
	statusWarn    lipgloss.Color
	toastFg       lipgloss.Color
	toastTag      lipgloss.Color
	text          lipgloss.Color
	cursorFg      lipgloss.Color
	cursorBg      lipgloss.Color
}

func newStyles() styles {
	palette := resolveThemePalette()
	timeColor := colorOrDefault(config.Config.TimeColor, palette.muted)
	nameColor := colorOrDefault(config.Config.NameColor, palette.text)
	contentColor := colorOrDefault(config.Config.ContentColor, palette.text)
	panelBorder := palette.panelBorder
	if strings.TrimSpace(config.Config.FrameColor) != "" {
		panelBorder = lipgloss.Color(config.Config.FrameColor)
	}
	infoColor := palette.hint
	if strings.TrimSpace(config.Config.InfoColor) != "" {
		infoColor = lipgloss.Color(config.Config.InfoColor)
	}
	rankColor := palette.accent
	if strings.TrimSpace(config.Config.RankColor) != "" {
		rankColor = lipgloss.Color(config.Config.RankColor)
	}

	return styles{
		headerBar:       lipgloss.NewStyle().Background(palette.panelBg).Foreground(palette.text),
		headerSegment:   lipgloss.NewStyle().Foreground(palette.text).Bold(true).Padding(0, 1),
		headerSegmentHi: lipgloss.NewStyle().Foreground(palette.text).Bold(true).Padding(0, 1),
		headerHint:      lipgloss.NewStyle().Foreground(infoColor).Padding(0, 1),
		headerTitle:     lipgloss.NewStyle().Background(palette.panelBg).Foreground(palette.text).Bold(true).Padding(0, 1),
		headerMeta:      lipgloss.NewStyle().Background(palette.panelBg).Foreground(palette.muted).Padding(0, 1),
		headerBadgeLive: lipgloss.NewStyle().Background(palette.panelBg).Foreground(palette.cta).Bold(true).Padding(0, 1),
		headerBadgeOff:  lipgloss.NewStyle().Background(palette.panelBg).Foreground(palette.muted).Bold(true).Padding(0, 1),
		panel:           lipgloss.NewStyle().Background(palette.panelBg).Border(lipgloss.RoundedBorder()).BorderForeground(panelBorder).Padding(0, 1),
		sidePanel:       lipgloss.NewStyle().Background(palette.panelBg).Border(lipgloss.RoundedBorder()).BorderForeground(palette.sideBorder).Padding(0, 1),
		messageCard:     lipgloss.NewStyle().Background(palette.surfaceBg).Border(lipgloss.RoundedBorder()).BorderForeground(palette.messageBorder),
		messageCardHi:   lipgloss.NewStyle().Background(palette.surfaceBg).Border(lipgloss.RoundedBorder()).BorderForeground(palette.accent).Bold(true),
		sideSection:     lipgloss.NewStyle().Background(palette.panelBg).Border(lipgloss.RoundedBorder()).BorderForeground(palette.sideBorder),
		sideSectionHi:   lipgloss.NewStyle().Background(palette.surfaceBg).Border(lipgloss.RoundedBorder()).BorderForeground(palette.accent).Bold(true),
		inputBox:        lipgloss.NewStyle().Background(palette.surfaceBg).Foreground(palette.text).Border(lipgloss.RoundedBorder()).BorderForeground(palette.messageBorder).Padding(0, 1),
		inputBoxHi:      lipgloss.NewStyle().Background(palette.surfaceBg).Foreground(palette.text).Border(lipgloss.RoundedBorder()).BorderForeground(palette.accent).Padding(0, 1),
		inputInner:      lipgloss.NewStyle().Background(palette.surfaceBg).Foreground(palette.text),
		statusBar:       lipgloss.NewStyle().Background(palette.panelBg).Foreground(palette.text),
		statusKey:       lipgloss.NewStyle().Background(palette.keyBg).Foreground(palette.text).Bold(true).Padding(0, 1),
		statusSeg:       lipgloss.NewStyle().Background(palette.surfaceBg).Foreground(palette.text).Padding(0, 1),
		statusSegFocus:  lipgloss.NewStyle().Background(palette.focusBg).Foreground(palette.text).Bold(true).Padding(0, 1),
		statusSegLive:   lipgloss.NewStyle().Background(palette.surfaceBg).Foreground(palette.cta).Bold(true).Padding(0, 1),
		statusSegOff:    lipgloss.NewStyle().Background(palette.surfaceBg).Foreground(palette.text).Padding(0, 1),
		statusSegMuted:  lipgloss.NewStyle().Background(palette.panelBg).Foreground(palette.muted).Padding(0, 1),
		statusSegWarn:   lipgloss.NewStyle().Background(palette.surfaceBg).Foreground(palette.statusWarn).Bold(true).Padding(0, 1),
		toastTag:        lipgloss.NewStyle().Background(palette.surfaceBg).Foreground(palette.toastTag).Bold(true).Padding(0, 1),
		toastText:       lipgloss.NewStyle().Background(palette.surfaceBg).Foreground(palette.toastFg).Padding(0, 1),
		scrollTrack:     lipgloss.NewStyle().Foreground(palette.sideBorder),
		scrollThumb:     lipgloss.NewStyle().Foreground(palette.accent),
		label:           lipgloss.NewStyle().Bold(true).Foreground(palette.accent),
		time:            lipgloss.NewStyle().Foreground(timeColor),
		name:            lipgloss.NewStyle().Foreground(nameColor).Bold(true),
		content:         lipgloss.NewStyle().Foreground(contentColor),
		system:          lipgloss.NewStyle().Foreground(palette.system).Bold(true),
		eventGift:       lipgloss.NewStyle().Foreground(palette.cta).Bold(true),
		eventLive:       lipgloss.NewStyle().Foreground(palette.statusWarn).Bold(true),
		msgGutter:       lipgloss.NewStyle().Foreground(palette.sideBorder),
		msgGutterHi:     lipgloss.NewStyle().Foreground(palette.accent),
		messageSelected: lipgloss.NewStyle().Background(palette.activeBg).Foreground(palette.text).Bold(true),
		rankName:        lipgloss.NewStyle().Foreground(rankColor),
		roomName:        lipgloss.NewStyle().Foreground(palette.text).Bold(true),
		roomSelected:    lipgloss.NewStyle().Background(palette.activeBg).Foreground(palette.text).Bold(true),
		roomOffline:     lipgloss.NewStyle().Foreground(palette.muted),
		roomLiveStatus:  lipgloss.NewStyle().Foreground(palette.cta).Bold(true),
		roomOffStatus:   lipgloss.NewStyle().Foreground(palette.muted),
		badgeLive:       lipgloss.NewStyle().Background(palette.surfaceBg).Foreground(palette.cta).Bold(true).Padding(0, 1),
		badgeOff:        lipgloss.NewStyle().Background(palette.surfaceBg).Foreground(palette.text).Bold(true).Padding(0, 1),
		muted:           lipgloss.NewStyle().Foreground(infoColor),
	}
}

func resolveThemePalette() themePalette {
	palette := paletteForTheme(config.Config.Theme)
	if strings.TrimSpace(config.Config.Background) != "" && strings.ToUpper(strings.TrimSpace(config.Config.Background)) != "NONE" {
		bg := lipgloss.Color(config.Config.Background)
		palette.panelBg = bg
		palette.surfaceBg = bg
	}
	return palette
}

func paletteForTheme(theme int64) themePalette {
	switch theme {
	case 2:
		return themeCatppuccinMocha()
	case 3:
		return themeGruvboxDark()
	case 4:
		return themeVSCodeDark()
	case 5:
		return themeCatppuccinLatte()
	case 6:
		return themeVSCodeLight()
	default:
		return themeTokyoNight()
	}
}

func themeTokyoNight() themePalette {
	return themePalette{
		panelBg:       lipgloss.Color("#1A1B26"),
		surfaceBg:     lipgloss.Color("#24283B"),
		activeBg:      lipgloss.Color("#2F3549"),
		keyBg:         lipgloss.Color("#2A3148"),
		focusBg:       lipgloss.Color("#283457"),
		panelBorder:   lipgloss.Color("#3B4261"),
		sideBorder:    lipgloss.Color("#33415C"),
		messageBorder: lipgloss.Color("#3D59A1"),
		accent:        lipgloss.Color("#7AA2F7"),
		cta:           lipgloss.Color("#9ECE6A"),
		muted:         lipgloss.Color("#7F849C"),
		system:        lipgloss.Color("#E0AF68"),
		hint:          lipgloss.Color("#A9B1D6"),
		statusWarn:    lipgloss.Color("#F7768E"),
		toastFg:       lipgloss.Color("#C0CAF5"),
		toastTag:      lipgloss.Color("#E0AF68"),
		text:          lipgloss.Color("#E2E8F0"),
		cursorFg:      lipgloss.Color("#E8F6FF"),
		cursorBg:      lipgloss.Color("#1A1B26"),
	}
}

func themeCatppuccinMocha() themePalette {
	return themePalette{
		panelBg:       lipgloss.Color("#1E1E2E"),
		surfaceBg:     lipgloss.Color("#313244"),
		activeBg:      lipgloss.Color("#45475A"),
		keyBg:         lipgloss.Color("#585B70"),
		focusBg:       lipgloss.Color("#1E3A5F"),
		panelBorder:   lipgloss.Color("#6C7086"),
		sideBorder:    lipgloss.Color("#7F849C"),
		messageBorder: lipgloss.Color("#89B4FA"),
		accent:        lipgloss.Color("#89B4FA"),
		cta:           lipgloss.Color("#A6E3A1"),
		muted:         lipgloss.Color("#9399B2"),
		system:        lipgloss.Color("#F9E2AF"),
		hint:          lipgloss.Color("#CDD6F4"),
		statusWarn:    lipgloss.Color("#F38BA8"),
		toastFg:       lipgloss.Color("#F5E0DC"),
		toastTag:      lipgloss.Color("#FAB387"),
		text:          lipgloss.Color("#E6E9EF"),
		cursorFg:      lipgloss.Color("#F5F7FF"),
		cursorBg:      lipgloss.Color("#1E1E2E"),
	}
}

func themeGruvboxDark() themePalette {
	return themePalette{
		panelBg:       lipgloss.Color("#282828"),
		surfaceBg:     lipgloss.Color("#3C3836"),
		activeBg:      lipgloss.Color("#504945"),
		keyBg:         lipgloss.Color("#665C54"),
		focusBg:       lipgloss.Color("#3C4F4B"),
		panelBorder:   lipgloss.Color("#7C6F64"),
		sideBorder:    lipgloss.Color("#928374"),
		messageBorder: lipgloss.Color("#83A598"),
		accent:        lipgloss.Color("#83A598"),
		cta:           lipgloss.Color("#B8BB26"),
		muted:         lipgloss.Color("#A89984"),
		system:        lipgloss.Color("#FABD2F"),
		hint:          lipgloss.Color("#EBDBB2"),
		statusWarn:    lipgloss.Color("#FB4934"),
		toastFg:       lipgloss.Color("#FBF1C7"),
		toastTag:      lipgloss.Color("#FE8019"),
		text:          lipgloss.Color("#F2E5BC"),
		cursorFg:      lipgloss.Color("#FFF4D4"),
		cursorBg:      lipgloss.Color("#282828"),
	}
}

func themeVSCodeDark() themePalette {
	return themePalette{
		panelBg:       lipgloss.Color("#1E1E1E"),
		surfaceBg:     lipgloss.Color("#252526"),
		activeBg:      lipgloss.Color("#2D2D30"),
		keyBg:         lipgloss.Color("#333333"),
		focusBg:       lipgloss.Color("#16324F"),
		panelBorder:   lipgloss.Color("#3C3C3C"),
		sideBorder:    lipgloss.Color("#4B4B4B"),
		messageBorder: lipgloss.Color("#007ACC"),
		accent:        lipgloss.Color("#4FC1FF"),
		cta:           lipgloss.Color("#89D185"),
		muted:         lipgloss.Color("#858585"),
		system:        lipgloss.Color("#D7BA7D"),
		hint:          lipgloss.Color("#CCCCCC"),
		statusWarn:    lipgloss.Color("#F48771"),
		toastFg:       lipgloss.Color("#F3F3F3"),
		toastTag:      lipgloss.Color("#D7BA7D"),
		text:          lipgloss.Color("#F3F3F3"),
		cursorFg:      lipgloss.Color("#FFFFFF"),
		cursorBg:      lipgloss.Color("#1E1E1E"),
	}
}

func themeCatppuccinLatte() themePalette {
	return themePalette{
		panelBg:       lipgloss.Color("#EFF1F5"),
		surfaceBg:     lipgloss.Color("#E6E9EF"),
		activeBg:      lipgloss.Color("#DCE0E8"),
		keyBg:         lipgloss.Color("#CCD0DA"),
		focusBg:       lipgloss.Color("#D8E7FF"),
		panelBorder:   lipgloss.Color("#BCC0CC"),
		sideBorder:    lipgloss.Color("#ACB0BE"),
		messageBorder: lipgloss.Color("#1E66F5"),
		accent:        lipgloss.Color("#1E66F5"),
		cta:           lipgloss.Color("#40A02B"),
		muted:         lipgloss.Color("#6C6F85"),
		system:        lipgloss.Color("#DF8E1D"),
		hint:          lipgloss.Color("#4C4F69"),
		statusWarn:    lipgloss.Color("#D20F39"),
		toastFg:       lipgloss.Color("#5C5F77"),
		toastTag:      lipgloss.Color("#FE640B"),
		text:          lipgloss.Color("#303446"),
		cursorFg:      lipgloss.Color("#1F2937"),
		cursorBg:      lipgloss.Color("#F8FAFC"),
	}
}

func themeVSCodeLight() themePalette {
	return themePalette{
		panelBg:       lipgloss.Color("#F3F3F3"),
		surfaceBg:     lipgloss.Color("#FFFFFF"),
		activeBg:      lipgloss.Color("#E8E8E8"),
		keyBg:         lipgloss.Color("#E1E4E8"),
		focusBg:       lipgloss.Color("#DCEBFF"),
		panelBorder:   lipgloss.Color("#C8C8C8"),
		sideBorder:    lipgloss.Color("#D0D0D0"),
		messageBorder: lipgloss.Color("#007ACC"),
		accent:        lipgloss.Color("#007ACC"),
		cta:           lipgloss.Color("#388A34"),
		muted:         lipgloss.Color("#6A737D"),
		system:        lipgloss.Color("#B57614"),
		hint:          lipgloss.Color("#24292E"),
		statusWarn:    lipgloss.Color("#D73A49"),
		toastFg:       lipgloss.Color("#24292E"),
		toastTag:      lipgloss.Color("#B57614"),
		text:          lipgloss.Color("#1F2328"),
		cursorFg:      lipgloss.Color("#111827"),
		cursorBg:      lipgloss.Color("#F8FAFC"),
	}
}

func colorOrDefault(v string, def lipgloss.Color) lipgloss.Color {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return lipgloss.Color(v)
}
