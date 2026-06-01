package getter

import (
	"encoding/json"
	"fmt"
)

func mapInt64(m map[string]interface{}, key string) int64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	return toInt64(v)
}

func mapInt(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

func mapString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return fmt.Sprintf("%v", s)
	}
}

func interactionContent(data map[string]interface{}) string {
	switch mapInt(data, "msg_type") {
	case 1:
		return "进入了房间"
	case 2:
		return "关注了主播"
	case 3:
		return "分享了直播间"
	default:
		return "进行了互动"
	}
}

func parseDanmuUser(info []interface{}) (string, int64) {
	if len(info) < 3 {
		return "", 0
	}
	user, ok := info[2].([]interface{})
	if !ok || len(user) < 2 {
		return "", 0
	}
	name, _ := user[1].(string)
	uid := toInt64(user[0])
	return name, uid
}

func parseDanmuContent(info []interface{}) string {
	if len(info) < 2 {
		return ""
	}
	content, _ := info[1].(string)
	return content
}

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}
