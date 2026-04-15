package model

import (
	"os"
	"strings"
	"sync"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

var (
	blockedTagList     []string
	blockedTagListOnce sync.Once
)

func getBlockedTagList() []string {
	blockedTagListOnce.Do(func() {
		blockedTags := os.Getenv("BLOCKED_TAGS")
		if blockedTags != "" {
			blockedTagList = lo.Map(strings.Split(blockedTags, ","), func(s string, _ int) string {
				return strings.TrimSpace(strings.ToLower(s))
			})
		}
	})
	return blockedTagList
}

func getAbilityDB() *gorm.DB {
	db := DB.Model(&Ability{})
	if blockedTags := getBlockedTagList(); len(blockedTags) > 0 {
		db = db.Where("(tag IS NULL OR LOWER(tag) NOT IN (?))", blockedTags)
	}
	return db
}
