package getter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
	"time"
)

func TestFetchPreferredOnlineRankUsersFallsBackWhenPrimaryHasNoUsers(t *testing.T) {
	users, total, err := fetchPreferredOnlineRankUsers(
		1,
		2,
		func(roomID int64, uid int64) ([]OnlineRankUser, int64, error) {
			return nil, 0, errors.New("primary failed")
		},
		func(roomID int64, uid int64) ([]OnlineRankUser, int64, error) {
			return []OnlineRankUser{{UID: 9, Name: "fallback", Rank: 1}}, 1, nil
		},
	)
	if err != nil {
		t.Fatalf("fetchPreferredOnlineRankUsers returned error: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(users) != 1 || users[0].Name != "fallback" {
		t.Fatalf("users = %+v, want fallback result", users)
	}
}

func TestFetchPreferredOnlineRankUsersKeepsPrimaryPartialUsers(t *testing.T) {
	fallbackCalled := false
	users, total, err := fetchPreferredOnlineRankUsers(
		1,
		2,
		func(roomID int64, uid int64) ([]OnlineRankUser, int64, error) {
			return []OnlineRankUser{{UID: 1, Name: "primary", Rank: 1}}, 10, errors.New("partial")
		},
		func(roomID int64, uid int64) ([]OnlineRankUser, int64, error) {
			fallbackCalled = true
			return []OnlineRankUser{{UID: 9, Name: "fallback", Rank: 1}}, 1, nil
		},
	)
	if err == nil {
		t.Fatal("err = nil, want non-nil")
	}
	if fallbackCalled {
		t.Fatal("fallbackCalled = true, want false")
	}
	if total != 10 {
		t.Fatalf("total = %d, want 10", total)
	}
	if len(users) != 1 || users[0].Name != "primary" {
		t.Fatalf("users = %+v, want primary partial result", users)
	}
}

func TestCollectOnlineRankUsersMergesPages(t *testing.T) {
	pages := map[int]onlineRankPage{
		1: {
			total: 4,
			users: []OnlineRankUser{
				{UID: 1, Name: "a", Rank: 1},
				{UID: 2, Name: "b", Rank: 2},
			},
		},
		2: {
			total: 4,
			users: []OnlineRankUser{
				{UID: 2, Name: "b", Rank: 2},
				{UID: 3, Name: "c", Rank: 3},
			},
		},
		3: {
			total: 4,
			users: []OnlineRankUser{
				{UID: 4, Name: "d", Rank: 4},
			},
		},
	}

	users, total, err := collectOnlineRankUsers(2, func(page int) (onlineRankPage, error) {
		if result, ok := pages[page]; ok {
			return result, nil
		}
		return onlineRankPage{}, nil
	})
	if err != nil {
		t.Fatalf("collectOnlineRankUsers returned error: %v", err)
	}
	if total != 4 {
		t.Fatalf("total = %d, want 4", total)
	}
	if len(users) != 4 {
		t.Fatalf("len(users) = %d, want 4", len(users))
	}
}

func TestCollectOnlineRankUsersReturnsPartialUsersOnLaterError(t *testing.T) {
	users, total, err := collectOnlineRankUsers(2, func(page int) (onlineRankPage, error) {
		if page == 1 {
			return onlineRankPage{
				total: 3,
				users: []OnlineRankUser{
					{UID: 1, Name: "a", Rank: 1},
					{UID: 2, Name: "b", Rank: 2},
				},
			}, nil
		}
		return onlineRankPage{}, errors.New("boom")
	})
	if err == nil {
		t.Fatal("err = nil, want non-nil")
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
}

func TestCollectOnlineRankUsersFallsBackToUserCountWhenTotalMissing(t *testing.T) {
	users, total, err := collectOnlineRankUsers(2, func(page int) (onlineRankPage, error) {
		switch page {
		case 1:
			return onlineRankPage{
				users: []OnlineRankUser{
					{UID: 1, Name: "a", Rank: 1},
					{UID: 2, Name: "b", Rank: 2},
				},
			}, nil
		case 2:
			return onlineRankPage{
				users: []OnlineRankUser{
					{UID: 3, Name: "c", Rank: 3},
				},
			}, nil
		default:
			return onlineRankPage{}, nil
		}
	})
	if err != nil {
		t.Fatalf("collectOnlineRankUsers returned error: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(users) != 3 {
		t.Fatalf("len(users) = %d, want 3", len(users))
	}
}

func TestCollectOnlineRankUsersRaisesTotalToUserCountWhenApiUnderreports(t *testing.T) {
	users, total, err := collectOnlineRankUsers(2, func(page int) (onlineRankPage, error) {
		switch page {
		case 1:
			return onlineRankPage{
				total: 1,
				users: []OnlineRankUser{
					{UID: 1, Name: "a", Rank: 1},
					{UID: 2, Name: "b", Rank: 2},
				},
			}, nil
		default:
			return onlineRankPage{}, nil
		}
	})
	if err != nil {
		t.Fatalf("collectOnlineRankUsers returned error: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
}

func TestMergeAudienceUsersCombinesSources(t *testing.T) {
	users := mergeAudienceUsers(
		[]AudienceUser{{UID: 1, Name: "alice", Sources: []string{"rank"}, Rank: 3, MedalName: "糊弄", MedalLevel: 26}},
		[]AudienceUser{{UID: 1, Name: "alice", Sources: []string{"guard"}, GuardLevel: 3, WealthLevel: 27}},
		[]AudienceUser{{UID: 2, Name: "bob", Sources: []string{"admin"}}},
	)
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
	if len(users[0].Sources) != 2 {
		t.Fatalf("len(users[0].Sources) = %d, want 2", len(users[0].Sources))
	}
	if users[0].Rank != 3 {
		t.Fatalf("users[0].Rank = %d, want 3", users[0].Rank)
	}
	if users[0].MedalName != "糊弄" || users[0].MedalLevel != 26 || users[0].GuardLevel != 3 || users[0].WealthLevel != 27 {
		t.Fatalf("users[0] = %+v, want merged audience extras", users[0])
	}
}

func TestFetchKnownAudienceUsersKeepsRankOnlyBaseList(t *testing.T) {
	users, err := fetchKnownAudienceUsers(1, 2, []OnlineRankUser{
		{UID: 1, Name: "alice", Rank: 1},
		{UID: 2, Name: "bob", Rank: 2},
	})
	if err != nil {
		t.Fatalf("fetchKnownAudienceUsers returned error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
	if users[0].Sources[0] != "rank" {
		t.Fatalf("users[0].Sources = %v, want rank", users[0].Sources)
	}
}

func TestParseContributionRankPageMapsUsers(t *testing.T) {
	body := []byte(`{
		"code":0,
		"data":{
			"item_num":31,
			"item":[
				{
					"uid":337186356,
					"name":"为什么泡面那么好吃",
					"face":"https://example.com/face.jpg",
					"rank":1,
					"score":481,
					"guard_level":3,
					"wealth_level":27,
					"medal_info":{"medal_name":"糊弄","level":26}
				},
				{
					"uid":42,
					"rank":2,
					"score":300,
					"uinfo":{
						"base":{"name":"fallback-name","face":"https://example.com/fallback.jpg"},
						"medal":{"name":"测试牌","level":8},
						"guard":{"level":1},
						"wealth":{"level":9}
					}
				}
			]
		}
	}`)

	page, err := parseContributionRankPage(body)
	if err != nil {
		t.Fatalf("parseContributionRankPage returned error: %v", err)
	}
	if page.total != 31 {
		t.Fatalf("page.total = %d, want 31", page.total)
	}
	if len(page.users) != 2 {
		t.Fatalf("len(page.users) = %d, want 2", len(page.users))
	}
	if page.users[0].UID != 337186356 || page.users[0].Name != "为什么泡面那么好吃" || page.users[0].Score != 481 || page.users[0].Rank != 1 {
		t.Fatalf("page.users[0] = %+v, want mapped top-level fields", page.users[0])
	}
	if page.users[0].MedalName != "糊弄" || page.users[0].MedalLevel != 26 || page.users[0].GuardLevel != 3 || page.users[0].WealthLevel != 27 {
		t.Fatalf("page.users[0] = %+v, want mapped contribution extras", page.users[0])
	}
	if page.users[1].Name != "fallback-name" || page.users[1].Face != "https://example.com/fallback.jpg" {
		t.Fatalf("page.users[1] = %+v, want uinfo.base fallback fields", page.users[1])
	}
	if page.users[1].MedalName != "测试牌" || page.users[1].MedalLevel != 8 || page.users[1].GuardLevel != 1 || page.users[1].WealthLevel != 9 {
		t.Fatalf("page.users[1] = %+v, want fallback extras from uinfo", page.users[1])
	}
}

func TestContributionRankTotalFallsBackAcrossKnownFields(t *testing.T) {
	body := []byte(`{"data":{"count":34}}`)
	if total := contributionRankTotal(body); total != 34 {
		t.Fatalf("total = %d, want 34", total)
	}
}

func TestFormatLiveDurationReturnsBlankForOfflineRoom(t *testing.T) {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.Local)
	if got := formatLiveDuration(0, "", now); got != "" {
		t.Fatalf("formatLiveDuration = %q, want empty for offline room", got)
	}
}

func TestFormatLiveDurationReturnsReadableLiveDuration(t *testing.T) {
	now := time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC)
	got := formatLiveDuration(1, "2026-04-03 10:30:00", now)
	if got == "" {
		t.Fatal("formatLiveDuration = empty, want non-empty live duration")
	}
}

func TestFormatLiveDurationUsesBilibiliLocalTime(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("LoadLocation returned error: %v", err)
	}
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, loc)
	got := formatLiveDuration(1, "2026-04-03 10:30:00", now)
	if got != "1时30分" {
		t.Fatalf("formatLiveDuration = %q, want %q", got, "1时30分")
	}
}

func TestUpdateWebsocketLatencyFromHeartbeatReply(t *testing.T) {
	d := &DanmuClient{}
	d.lastHeartbeatSentNs.Store(time.Now().Add(-25 * time.Millisecond).UnixNano())

	var buf bytes.Buffer
	for _, v := range []interface{}{
		uint32(20),
		uint16(16),
		uint16(1),
		uint32(3),
		uint32(1),
		uint32(0),
	} {
		if err := binary.Write(&buf, binary.BigEndian, v); err != nil {
			t.Fatalf("binary.Write returned error: %v", err)
		}
	}

	d.updateWebsocketLatency(buf.Bytes())
	if got := d.wsLatencyMs.Load(); got <= 0 {
		t.Fatalf("wsLatencyMs = %d, want > 0", got)
	}
}
