package service

import (
	"log"
	"time"

	"newshub-rss-service/model"

	"gorm.io/gorm"
)

// Cleaner struct for cleaner object
type Cleaner struct {
	db     *gorm.DB
	config model.Config
}

// CreateCleaner create cleaner object
func CreateCleaner(cfg model.Config) *Cleaner {
	srv := new(Cleaner)
	srv.db = GetDb(cfg)
	srv.config = cfg

	return srv
}

// Clean - remove articles where create date less month
func (srv *Cleaner) Clean() {
	// TODO: delete old records by SQL (big count)
	var feeds []model.Feeds

	if err := srv.db.Find(&feeds).Preload("Articles").Error; err != nil {
		log.Println("get feeds for clean error")
		return
	}

	for _, feed := range feeds {
		articlesCount := len(feed.Articles)
		if articlesCount > srv.config.ArticlesMaxCount {
			srv.deleteArticles()
		}
	}
}

func (srv *Cleaner) deleteArticles() {
	// fixme
	week := time.Hour * 168
	oldTime := time.Now().Add(-4 * week).Unix()

	err := srv.db.
		Where("Date < ? AND IsBookmark=0 AND IsRead=1", oldTime).
		Delete(model.Articles{}).
		Error

	if err != nil {
		log.Printf("delete old articles error: %s", err)
	}
}
