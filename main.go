package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"newshub-rss-service/model"
	"newshub-rss-service/service"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var conf model.Config

const defaultConfigPath = "./cfg.json"

func init() {
	// read config file
	pathPtr := flag.String("config", defaultConfigPath, "Path for configuration file")
	flag.Parse()

	if *pathPtr == "" {
		panic("No config path")
	}

	bytes, err := ioutil.ReadFile(*pathPtr)

	if err != nil {
		panic("Read config file error")
	}

	// set default values
	conf.UpdateMinutes = 30

	if err := json.Unmarshal(bytes, &conf); err != nil {
		panic(err.Error())
	}
}

func main() {
	log.Println("start rss services")

	updater := service.CreateUpdater(conf)
	cleaner := service.CreateCleaner(conf)

	updateTimer := time.Tick(time.Duration(conf.UpdateMinutes) * time.Minute)
	weekTimer := time.Tick(time.Hour * 168) // week

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	go updater.Update()
	go cleaner.Clean()

	for {
		select {
		case <-updateTimer:
			go updater.Update()
		case <-weekTimer:
			go cleaner.Clean()
		case <-sigs:
			log.Println("shutdown rss service")
			updater.Close()
			return
		}
	}
}
