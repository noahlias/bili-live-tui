package getter

import (
	"fmt"

	myhttp "github.com/BYT0723/go-tools/http"
	"github.com/tidwall/gjson"
)

type UserInfo struct {
	UID       int64
	Name      string
	Face      string
	TopPhoto  string
	LiveCover string
}

func GetUserInfo(uid int64) (UserInfo, error) {
	header := buildBaseHeader()
	query, err := signWbiParams(map[string]string{
		"mid": fmt.Sprintf("%d", uid),
	}, header)
	if err != nil {
		return UserInfo{}, err
	}
	_, body, err := myhttp.Get("https://api.bilibili.com/x/space/wbi/acc/info?"+query, header, nil)
	if err != nil {
		return UserInfo{}, err
	}
	if gjson.GetBytes(body, "code").Int() != 0 {
		return UserInfo{}, fmt.Errorf("user info response error")
	}
	info := UserInfo{
		UID:       uid,
		Name:      gjson.GetBytes(body, "data.name").String(),
		Face:      gjson.GetBytes(body, "data.face").String(),
		TopPhoto:  gjson.GetBytes(body, "data.top_photo").String(),
		LiveCover: gjson.GetBytes(body, "data.live_room.cover").String(),
	}
	return info, nil
}
