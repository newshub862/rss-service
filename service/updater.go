package service

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"newshub-rss-service/model"

	"golang.org/x/net/html/charset"
	"gorm.io/gorm"
)

const saveChanBufferSize = 200
const getChanBufferSize = 200

// Updater - service
type Updater struct {
	db              *gorm.DB
	config          model.Config
	getChan         chan getData
	saveChan        chan []model.Articles
	prepareSaveChan chan model.PreparedDataForSave
	httpClient      *http.Client
}

type getData struct {
	Rss  model.Feeds
	Body io.ReadCloser
}

// CreateUpdater - create and configure Updater struct
func CreateUpdater(cfg model.Config) *Updater {
	srv := new(Updater)
	srv.db = GetDb(cfg)
	srv.config = cfg
	srv.getChan = make(chan getData, getChanBufferSize)
	srv.saveChan = make(chan []model.Articles, saveChanBufferSize)
	srv.prepareSaveChan = make(chan model.PreparedDataForSave, saveChanBufferSize)
	srv.httpClient = &http.Client{
		Timeout: 2 * time.Minute,
		Transport: &http.Transport{
			Dial:                (&net.Dialer{Timeout: 5 * time.Second}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}

	go srv.saveArticle() // save articles after parsing

	return srv
}

// Close close all channels and conection
func (srv *Updater) Close() {
	close(srv.getChan)
	close(srv.saveChan)
	close(srv.prepareSaveChan)
}

// Update - get new feeds for users
func (srv *Updater) Update() {
	userIds, err := getUserIds(srv.db)
	if err != nil {
		log.Println("get user ids for update rss error:", err)
		return
	}

	if len(userIds) == 0 {
		log.Println("no users for update")
		return
	}

	feeds := make([]model.Feeds, 0)

	err = srv.db.Model(&feeds).
		Preload("Articles").
		Where("UserId in (?)", userIds).
		Find(&feeds).
		Error
	if err != nil {
		log.Println("get feeds for update rss error:", err)
		return
	}

	if len(feeds) == 0 {
		return
	}

	for _, feed := range feeds {
		rssBody, err := srv.getFeedBody(feed.Url)

		if err != nil {
			log.Println("get rss error: ", err.Error())
			continue
		}

		srv.getChan <- getData{Body: rssBody, Rss: feed}
	}
}

func (srv Updater) getFeedBody(url string) (io.ReadCloser, error) {
	response, err := srv.httpClient.Get(url)
	if err != nil {
		log.Println("get feed error:", err.Error())
		return nil, err
	}

	if response.StatusCode == 404 {
		return nil, fmt.Errorf("get body for %s error: not found", url)
	}

	return response.Body, nil
}

func (srv *Updater) updateArticles(data model.PreparedDataForSave) {
	links := make(map[string]bool, len(data.RSS.Articles))
	articles := make([]model.Articles, 0, len(data.RSS.Articles))

	// get existed post links
	for _, article := range data.RSS.Articles {
		links[article.Link] = true
	}

	for _, article := range data.XMLMode.Articles {
		if _, isExist := links[article.Link]; !isExist {
			// if existing links does not have current link - send to save channel
			newArticle := srv.rssArticleFromXML(article)
			newArticle.FeedId = data.RSS.Id

			articles = append(articles, newArticle)
		}
	}

	if len(articles) != 0 {
		srv.saveChan <- articles
	}
}

// rssArticleFromXML - create RssArticle from XMLArticle
func (srv *Updater) rssArticleFromXML(xmlArticle model.XMLArticle) model.Articles {
	articleTime, err := time.Parse(time.RFC1123, xmlArticle.Date)
	if err != nil {
		log.Printf("parse xml article date %s error: %s", xmlArticle.Date, err)
		articleTime = time.Now()
	}

	rssArticle := model.Articles{
		Body:   xmlArticle.Description,
		Title:  xmlArticle.Title,
		Link:   xmlArticle.Link,
		Date:   articleTime.Unix(),
		IsRead: false,
	}

	return rssArticle
}

func (srv *Updater) saveArticle() {
	for {
		select {

		// get oly new links for create
		case saveData := <-srv.prepareSaveChan:
			srv.updateArticles(saveData)

		// save new article for feed
		case articles := <-srv.saveChan:
			if err := srv.db.Save(&articles).Error; err != nil {
				log.Printf("save articles error: %s", err)
			}

		// read RSS message, set universal encoding^ parse ang send to save channel
		case data := <-srv.getChan:
			var xmlModel model.XMLFeed
			decoder := xml.NewDecoder(data.Body)
			decoder.CharsetReader = charset.NewReaderLabel

			if err := decoder.Decode(&xmlModel); err != nil {
				log.Printf("decode body for %s error: %s", data.Rss.Name, err)

				if err = data.Body.Close(); err != nil {
					log.Printf("close body for %s error: %s", data.Rss.Name, err)
				}

				continue
			}

			// update DB
			srv.prepareSaveChan <- model.PreparedDataForSave{
				RSS:     data.Rss,
				XMLMode: xmlModel,
			}

			if err := data.Body.Close(); err != nil {
				log.Println("close rss body error:", err)
			}
		}
	}
}

func getUserIds(db *gorm.DB) ([]int64, error) {
	ids := make([]int64, 0)

	err := db.
		Model(&model.Settings{}).
		Where(&model.Settings{RssEnabled: true}).
		Pluck("UserId", &ids).
		Error
	if err != nil {
		return nil, err
	}

	return ids, nil
}
