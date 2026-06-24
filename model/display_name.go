package model

func GetDisplayNamesByUserIDs(userIDs []int) map[int]string {
	result := make(map[int]string, len(userIDs))
	if len(userIDs) == 0 {
		return result
	}
	var users []struct {
		Id          int    `gorm:"column:id"`
		DisplayName string `gorm:"column:display_name"`
	}
	if err := DB.Model(&User{}).Select("id, display_name").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return result
	}
	for _, user := range users {
		if user.DisplayName != "" {
			result[user.Id] = user.DisplayName
		}
	}
	return result
}

// FillUserDisplayNames 对带用户 id 的行集合去重后单次查询,批量回填 displayName,
// 供各界面复用,无需为每个行类型重写去重 + 查询样板。
// displayName 属展示信息,查不到或查询出错时静默跳过(故不返回 error)。
func FillUserDisplayNames[T any](rows []T, userID func(T) int, setDisplayName func(T, string)) {
	if len(rows) == 0 {
		return
	}
	ids := make([]int, 0, len(rows))
	seen := make(map[int]struct{}, len(rows))
	for _, row := range rows {
		id := userID(row)
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return
	}
	names := GetDisplayNamesByUserIDs(ids)
	for _, row := range rows {
		if name := names[userID(row)]; name != "" {
			setDisplayName(row, name)
		}
	}
}
