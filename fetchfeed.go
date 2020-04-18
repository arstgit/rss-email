package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
)

type httpGetRes struct {
	url   string
	txt   string
	error error
}

func fetchfeed(config *emailConfig) error {
	ch := make(chan *httpGetRes)

	var goNum int
	subscription.RLock()
	for url := range subscription.m {
		// url not a valid feed
		if subscription.m[url].error != nil && subscription.m[url].error.Error() == "Failed to detect feed type" {
			continue
		}
		go func() {
			res, err := httpGet(url)
			ch <- &httpGetRes{
				url:   url,
				txt:   res,
				error: err,
			}
		}()
		goNum++
	}
	subscription.RUnlock()

	for i := 0; i < goNum; i++ {
		// block here
		res := <-ch
		url := res.url
		txt := res.txt

		subscription.Lock()

		if res.error != nil {
			log.Printf("%s", res.error)
			subscription.m[url].error = res.error

			subscription.Unlock()
			continue
		}

		feedParser := gofeed.NewParser()
		feed, err := feedParser.ParseString(txt)
		if err != nil {
			subscription.m[url].error = err

			subscription.Unlock()
			continue
		}

		// save to global variable
		subscription.m[url].raw = txt
		subscription.m[url].lastUpdate = time.Now()
		subscription.m[url].feed = feed

		subscription.Unlock()
	}

	return nil
}

func httpGet(url string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	output, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(output), nil
}
