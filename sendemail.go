package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"html"
	"log"
	"mime/quotedprintable"
	"net/smtp"
	"strings"
	"text/template"

	"github.com/mmcdole/gofeed"
)

const feedSubject = "[rss-email] feed"

const mailTemplate = `To: {{.To}}
Subject: {{.Subject}}
Content-Type: text/html; charset="UTF-8"
Content-Transfer-Encoding: quoted-printable

{{.Body}}
`

type templateParams struct {
	To      string
	Subject string
	Body    string
}

func sendSubscription(config *emailConfig) error {
	for to, userUrls := range userSubscriptions.m {
		param := &bodyParam{}
		var urlHashMap = make(map[string]string)

		for url, userURLInfo := range *userUrls {
			urlInfo, ok := subscription.m[url]
			if !ok {
				return errors.New("feed url not in map")
			}

			// make sure we have populated feed
			if urlInfo.lastUpdate.IsZero() {
				continue
			}

			filteredFeed, nextHash, err := filterFeed(urlInfo.feed, userURLInfo)
			if err != nil {
				log.Print(err)
				continue
			}
			param.Feeds = append(param.Feeds, filteredFeed)
			urlHashMap[url] = nextHash
		}

		// make sure we have contents to send
		var itemNum int
		for _, feed := range param.Feeds {
			itemNum += len(feed.Items)
		}
		if itemNum == 0 {
			return nil
		}

		param.Expect = len(*userUrls)
		param.Actual = len(param.Feeds)
		param.ShowErr = param.Expect != param.Actual

		parsedFeed, err := parsefeed(param)
		if err != nil {
			log.Print(err)
			continue
		}

		body, err := toQuotedPrintable(html.UnescapeString(parsedFeed))
		if err != nil {
			return err
		}
		if err := sendemail(config, to, feedSubject, body); err != nil {
			return err
		}

		// Update visited hash
		for url, hash := range urlHashMap {
			(*userUrls)[url].LastHash = hash
		}
	}

	return nil
}

func sendemail(config *emailConfig, to, subject, body string) error {
	src := strings.ReplaceAll(mailTemplate, "\n", "\r\n")

	t := template.Must(template.New("mailTemplate").Parse(src))
	msg := &bytes.Buffer{}
	params := templateParams{to, subject, body}
	err := t.Execute(msg, params)
	if err != nil {
		return err
	}
	auth := LOGINAuth(config.username, config.password)

	err = smtp.SendMail(config.smtpServer, auth, config.from, []string{to}, msg.Bytes())
	if err != nil {
		return err
	}

	log.Printf("one email sent to %s", to)

	return nil
}

func toQuotedPrintable(s string) (string, error) {
	var buf bytes.Buffer
	w := quotedprintable.NewWriter(&buf)
	_, err := w.Write([]byte(s))
	if err != nil {
		return "", err
	}
	err = w.Close()
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func filterFeed(feed *gofeed.Feed, userURLInfo *userURLInfo) (*gofeed.Feed, string, error) {
	var res = new(gofeed.Feed)
	*res = *feed

	nextHash := ""
	for index, item := range feed.Items {
		hasher := sha1.New()
		_, err := hasher.Write([]byte(item.Title))
		if err != nil {
			return nil, "", nil
		}
		hasher.Write([]byte(item.Published))
		if err != nil {
			return nil, "", nil
		}
		hashBytes := hasher.Sum(nil)
		hexSha1 := hex.EncodeToString(hashBytes)

		if index == 0 {
			nextHash = hexSha1
		}
		if hexSha1 == userURLInfo.LastHash {
			res.Items = res.Items[:index]
			break
		}
	}

	return res, nextHash, nil
}
