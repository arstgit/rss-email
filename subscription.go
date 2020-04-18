package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

var savedFilePath = path.Join("/", "rss-email", "user")

type userURLInfo struct {
	LastHash string
}

func newUserURLInfo() *userURLInfo {
	return &userURLInfo{}
}

type userSubscriptionType map[string]*userURLInfo

func newUserSubscription() *userSubscriptionType {
	return &userSubscriptionType{}
}

// printToUser produce email body to user
func (userSubscription *userSubscriptionType) printToUser() (str string, err error) {
	str += fmt.Sprintf("<div>your subscribed RSS count: %d </div><br>", len(*userSubscription))

	str += "<div>subscribed RSS url list:</div>"

	for k := range *userSubscription {
		str += "<div>" + k + "</div>"
	}

	return str, nil
}

type userSubscriptionsType struct {
	sync.RWMutex
	m map[string]*userSubscriptionType
}

func newUserSubscriptions() userSubscriptionsType {
	return userSubscriptionsType{m: make(map[string]*userSubscriptionType)}
}

// save userSubscription.m to disc.
func (userSubscriptions *userSubscriptionsType) saveToDisk() error {
	dir, _ := path.Split(savedFilePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	file, err := os.Create(savedFilePath)
	if err != nil {
		return err
	}

	b, err := json.Marshal(userSubscriptions.m)
	if err != nil {
		return err
	}
	if _, err := file.Write(b); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

// save userSubscription.m to disc.
func (userSubscriptions *userSubscriptionsType) restoreFromDisk() error {
	file, err := os.Open(savedFilePath)
	if err != nil {
		log.Println("no file to restore from")
		return nil
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b, &userSubscriptions.m); err != nil {
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

type urlInfo struct {
	raw        string
	lastUpdate time.Time
	feed       *gofeed.Feed
	error      error
}

func newURLInfo() *urlInfo {
	return &urlInfo{}
}

type subscriptionType struct {
	sync.RWMutex
	m map[string]*urlInfo
}

func newSubscription() subscriptionType {
	return subscriptionType{m: make(map[string]*urlInfo)}
}
