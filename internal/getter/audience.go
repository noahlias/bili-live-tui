package getter

import (
	"fmt"

	myhttp "github.com/BYT0723/go-tools/http"

	"github.com/asmcos/requests"

	"github.com/tidwall/gjson"
)

func GetAllOnlineRankUsers(roomID int64) ([]OnlineRankUser, int64, error) {
	uid, err := GetRoomUID(roomID)
	if err != nil {
		return nil, 0, err
	}
	return fetchAllOnlineRankUsers(roomID, uid)
}

func GetKnownAudienceUsers(roomID int64) ([]AudienceUser, error) {
	uid, err := GetRoomUID(roomID)
	if err != nil {
		return nil, err
	}
	rankUsers, _, err := fetchAllOnlineRankUsers(roomID, uid)
	if err != nil && len(rankUsers) == 0 {
		return nil, err
	}
	return fetchKnownAudienceUsers(roomID, uid, rankUsers)
}

func GetRoomAdminUsers(roomID int64) ([]AudienceUser, error) {
	return fetchAdminAudienceUsers(roomID)
}

func fetchAllOnlineRankUsers(roomID int64, uid int64) ([]OnlineRankUser, int64, error) {
	return fetchPreferredOnlineRankUsers(roomID, uid, fetchContributionRankUsers, fetchLegacyOnlineRankUsers)
}

func fetchPreferredOnlineRankUsers(roomID int64, uid int64, primary onlineRankUsersFetcher, fallback onlineRankUsersFetcher) ([]OnlineRankUser, int64, error) {
	users, total, err := primary(roomID, uid)
	if err == nil || len(users) > 0 {
		return users, total, err
	}
	return fallback(roomID, uid)
}

func fetchContributionRankUsers(roomID int64, uid int64) ([]OnlineRankUser, int64, error) {
	const pageSize = 50
	page, err := fetchContributionRankPage(roomID, uid, 1, pageSize)
	if err != nil {
		return nil, 0, err
	}
	if page.total == 0 || page.total < int64(len(page.users)) {
		page.total = int64(len(page.users))
	}
	return page.users, page.total, nil
}

func fetchLegacyOnlineRankUsers(roomID int64, uid int64) ([]OnlineRankUser, int64, error) {
	const pageSize = 50
	return collectOnlineRankUsers(pageSize, func(page int) (onlineRankPage, error) {
		return fetchOnlineRankPage(roomID, uid, page, pageSize)
	})
}

func fetchKnownAudienceUsers(roomID int64, uid int64, rankUsers []OnlineRankUser) ([]AudienceUser, error) {
	users := make([]AudienceUser, 0, len(rankUsers))
	for _, user := range rankUsers {
		users = append(users, AudienceUser{
			UID:         user.UID,
			Name:        user.Name,
			Face:        user.Face,
			Sources:     []string{"rank"},
			Rank:        user.Rank,
			Score:       user.Score,
			MedalName:   user.MedalName,
			MedalLevel:  user.MedalLevel,
			GuardLevel:  user.GuardLevel,
			WealthLevel: user.WealthLevel,
		})
	}
	_ = roomID
	_ = uid
	return users, nil
}

func collectOnlineRankUsers(pageSize int, fetch func(page int) (onlineRankPage, error)) ([]OnlineRankUser, int64, error) {
	const maxPages = 200
	users := make([]OnlineRankUser, 0, pageSize)
	seen := make(map[string]bool)
	var total int64

	for page := 1; page <= maxPages; page++ {
		result, err := fetch(page)
		if err != nil {
			return users, total, err
		}
		if result.total > 0 {
			total = result.total
		}
		if len(result.users) == 0 {
			break
		}
		for _, user := range result.users {
			key := onlineRankUserKey(user)
			if seen[key] {
				continue
			}
			seen[key] = true
			users = append(users, user)
		}
		if total > 0 && int64(len(users)) >= total {
			break
		}
	}

	if total == 0 || total < int64(len(users)) {
		total = int64(len(users))
	}
	return users, total, nil
}

func fetchContributionRankPage(roomID int64, uid int64, page int, pageSize int) (onlineRankPage, error) {
	header := buildBaseHeader()
	header.Set("Origin", "https://live.bilibili.com")
	header.Set("Referer", fmt.Sprintf("https://live.bilibili.com/%d", roomID))

	query, err := signWbiParams(map[string]string{
		"ruid":         fmt.Sprintf("%d", uid),
		"room_id":      fmt.Sprintf("%d", roomID),
		"page":         fmt.Sprintf("%d", page),
		"page_size":    fmt.Sprintf("%d", pageSize),
		"type":         "online_rank",
		"switch":       "contribution_rank",
		"platform":     "web",
		"web_location": "444.8",
	}, header)
	if err != nil {
		return onlineRankPage{}, err
	}

	_, body, err := myhttp.Get("https://api.live.bilibili.com/xlive/general-interface/v1/rank/queryContributionRank?"+query, header, nil)
	if err != nil {
		return onlineRankPage{}, err
	}
	return parseContributionRankPage(body)
}

func parseContributionRankPage(body []byte) (onlineRankPage, error) {
	if gjson.GetBytes(body, "code").Int() != 0 {
		return onlineRankPage{}, fmt.Errorf("contribution rank response error")
	}

	pageResult := onlineRankPage{
		total: contributionRankTotal(body),
		users: make([]OnlineRankUser, 0),
	}
	rawUsers := gjson.GetBytes(body, "data.item").Array()
	for _, rawUser := range rawUsers {
		name := rawUser.Get("name").String()
		if name == "" {
			name = rawUser.Get("uinfo.base.name").String()
		}
		face := rawUser.Get("face").String()
		if face == "" {
			face = rawUser.Get("uinfo.base.face").String()
		}
		rank := rawUser.Get("rank").Int()
		if rank == 0 {
			rank = rawUser.Get("userRank").Int()
		}
		pageResult.users = append(pageResult.users, OnlineRankUser{
			UID:         rawUser.Get("uid").Int(),
			Name:        name,
			Face:        face,
			Score:       rawUser.Get("score").Int(),
			Rank:        rank,
			MedalName:   contributionMedalName(rawUser),
			MedalLevel:  contributionMedalLevel(rawUser),
			GuardLevel:  contributionGuardLevel(rawUser),
			WealthLevel: contributionWealthLevel(rawUser),
		})
	}
	return pageResult, nil
}

func contributionMedalName(rawUser gjson.Result) string {
	if v := rawUser.Get("medal_info.medal_name").String(); v != "" {
		return v
	}
	return rawUser.Get("uinfo.medal.name").String()
}

func contributionMedalLevel(rawUser gjson.Result) int64 {
	if v := rawUser.Get("medal_info.level").Int(); v > 0 {
		return v
	}
	return rawUser.Get("uinfo.medal.level").Int()
}

func contributionGuardLevel(rawUser gjson.Result) int64 {
	if v := rawUser.Get("guard_level").Int(); v > 0 {
		return v
	}
	if v := rawUser.Get("medal_info.guard_level").Int(); v > 0 {
		return v
	}
	return rawUser.Get("uinfo.guard.level").Int()
}

func contributionWealthLevel(rawUser gjson.Result) int64 {
	if v := rawUser.Get("wealth_level").Int(); v > 0 {
		return v
	}
	return rawUser.Get("uinfo.wealth.level").Int()
}

func contributionRankTotal(body []byte) int64 {
	paths := []string{
		"data.item_num",
		"data.count",
		"data.total",
		"data.online_num",
		"data.onlineNum",
	}
	for _, path := range paths {
		if total := gjson.GetBytes(body, path).Int(); total > 0 {
			return total
		}
	}
	return 0
}

func fetchOnlineRankPage(roomID int64, uid int64, page int, pageSize int) (onlineRankPage, error) {
	onlineRankAPI := fmt.Sprintf("https://api.live.bilibili.com/xlive/general-interface/v1/rank/getOnlineGoldRank?ruid=%d&roomId=%d&page=%d&pageSize=%d", uid, roomID, page, pageSize)
	r, err := requests.Get(onlineRankAPI)
	if err != nil {
		return onlineRankPage{}, err
	}
	if gjson.Get(r.Text(), "code").Int() != 0 {
		return onlineRankPage{}, fmt.Errorf("online rank response error")
	}

	pageResult := onlineRankPage{
		total: gjson.Get(r.Text(), "data.onlineNum").Int(),
		users: make([]OnlineRankUser, 0),
	}
	rawUsers := gjson.Get(r.Text(), "data.OnlineRankItem").Array()
	for _, rawUser := range rawUsers {
		pageResult.users = append(pageResult.users, OnlineRankUser{
			UID:   rawUser.Get("uid").Int(),
			Name:  rawUser.Get("name").String(),
			Face:  rawUser.Get("face").String(),
			Score: rawUser.Get("score").Int(),
			Rank:  rawUser.Get("userRank").Int(),
		})
	}
	return pageResult, nil
}

func onlineRankUserKey(user OnlineRankUser) string {
	if user.UID > 0 {
		return fmt.Sprintf("uid:%d", user.UID)
	}
	return fmt.Sprintf("name:%s:rank:%d", user.Name, user.Rank)
}

func fetchGuardAudienceUsers(roomID int64, uid int64) ([]AudienceUser, error) {
	guardAPI := fmt.Sprintf("https://api.live.bilibili.com/xlive/app-room/v2/guardTab/topListNew?roomid=%d&page=1&ruid=%d", roomID, uid)
	r, err := requests.Get(guardAPI)
	if err != nil {
		return nil, err
	}
	if gjson.Get(r.Text(), "code").Int() != 0 {
		return nil, fmt.Errorf("guard audience response error")
	}

	users := make([]AudienceUser, 0)
	addGuardList := func(path string) {
		for _, raw := range gjson.Get(r.Text(), path).Array() {
			base := raw.Get("uinfo.base")
			users = append(users, AudienceUser{
				UID:     raw.Get("uinfo.uid").Int(),
				Name:    base.Get("name").String(),
				Face:    base.Get("face").String(),
				Sources: []string{"guard"},
			})
		}
	}
	addGuardList("data.top3")
	addGuardList("data.list")
	return users, nil
}

func fetchAdminAudienceUsers(roomID int64) ([]AudienceUser, error) {
	adminAPI := fmt.Sprintf("https://api.live.bilibili.com/xlive/web-room/v1/roomAdmin/get_by_room?roomid=%d", roomID)
	r, err := requests.Get(adminAPI)
	if err != nil {
		return nil, err
	}
	if gjson.Get(r.Text(), "code").Int() != 0 {
		return nil, fmt.Errorf("room admin response error")
	}
	users := make([]AudienceUser, 0)
	for _, raw := range gjson.Get(r.Text(), "data.data").Array() {
		users = append(users, AudienceUser{
			UID:     raw.Get("uid").Int(),
			Name:    raw.Get("uname").String(),
			Face:    raw.Get("face").String(),
			Sources: []string{"admin"},
		})
	}
	return users, nil
}

func mergeAudienceUsers(groups ...[]AudienceUser) []AudienceUser {
	merged := make(map[string]AudienceUser)
	order := make([]string, 0)
	for _, group := range groups {
		for _, user := range group {
			key := audienceUserKey(user)
			existing, ok := merged[key]
			if !ok {
				merged[key] = normalizeAudienceUser(user)
				order = append(order, key)
				continue
			}
			merged[key] = combineAudienceUser(existing, user)
		}
	}
	users := make([]AudienceUser, 0, len(order))
	for _, key := range order {
		users = append(users, merged[key])
	}
	return users
}

func audienceUserKey(user AudienceUser) string {
	if user.UID > 0 {
		return fmt.Sprintf("uid:%d", user.UID)
	}
	return "name:" + user.Name
}

func normalizeAudienceUser(user AudienceUser) AudienceUser {
	user.Sources = uniqueAudienceSources(user.Sources)
	return user
}

func combineAudienceUser(current AudienceUser, incoming AudienceUser) AudienceUser {
	if current.Name == "" {
		current.Name = incoming.Name
	}
	if current.Face == "" {
		current.Face = incoming.Face
	}
	if current.Rank == 0 || (incoming.Rank > 0 && incoming.Rank < current.Rank) {
		current.Rank = incoming.Rank
	}
	if incoming.Score > current.Score {
		current.Score = incoming.Score
	}
	if current.MedalName == "" {
		current.MedalName = incoming.MedalName
	}
	if incoming.MedalLevel > current.MedalLevel {
		current.MedalLevel = incoming.MedalLevel
	}
	if incoming.GuardLevel > current.GuardLevel {
		current.GuardLevel = incoming.GuardLevel
	}
	if incoming.WealthLevel > current.WealthLevel {
		current.WealthLevel = incoming.WealthLevel
	}
	current.Sources = uniqueAudienceSources(append(current.Sources, incoming.Sources...))
	return current
}

func uniqueAudienceSources(sources []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(sources))
	for _, source := range sources {
		if source == "" || seen[source] {
			continue
		}
		seen[source] = true
		out = append(out, source)
	}
	return out
}
