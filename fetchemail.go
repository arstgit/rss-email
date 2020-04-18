package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/url"
	"regexp"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

const responseSubscribeSubject = "[rss-email] successfully subscribe"
const responseSubscribeSubjectFail = "[rss-email] unsuccessfully subscribe"
const responseSubscribeBody0Url = "no valid RSS urls"
const responseListSubject = "[rss-email] list command response"
const responseUnsubscribeSubject = "[rss-email] successfully unsubscribe"
const responseNotSubscribeSubject = "[rss-email] you haven't subscribed yet."
const responseNotSubscribeBody = "you haven't subscribed yet."
const responseSubjectHelp = "[rss-email] unrecognized command"
const responseBodyHelp = `
<h3>Usage:</h3>
<p>Email subject: rss-email [COMMAND]</p>
<p>COMMAND is one of : subscribe, list, unsubscribe</p>
<br>
<p>For more details: https://github.com/derekchuank/rss-email</p>
`

const httpRegex = `(?:http(s)?:\/\/)?[\w.-]+(?:\.[\w\.-]+)+[\w\-\._~:/?#[\]@!\$&'\(\)\*\+,;=.]+\r\n`

func fetchemail(config *emailConfig) error {
	c, err := client.DialTLS(config.imapServer, nil)
	if err != nil {
		return err
	}

	defer c.Logout()

	if err := c.Login(config.username, config.password); err != nil {
		return err
	}

	if err := fetchFromBox(config, c, "INBOX"); err != nil {
		log.Printf("error fetchFromBox INBOX")
	}

	// Junk can contain messages useful
	if err := fetchFromBox(config, c, "Junk"); err != nil {
		log.Printf("error fetchFromBox Junk")
	}

	return nil
}

func fetchFromBox(config *emailConfig, c *client.Client, boxType string) error {
	mbox, err := c.Select(boxType, false)
	if err != nil {
		return err
	}

	if mbox.UnseenSeqNum == 0 {
		log.Println("no unseen emails")
		return nil
	}

	from := mbox.UnseenSeqNum
	to := mbox.Messages

	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	// Get the whole message body
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}
	go func() {
		done <- c.Fetch(seqset, items, messages)
	}()

	for msg := range messages {
		r := msg.GetBody(section)
		if r == nil {
			return errors.New("Server didn't returned message body")
		}

		msg, err := mail.ReadMessage(r)
		if err != nil {
			return err
		}

		header := msg.Header
		fromAddress, err := mail.ParseAddress(header.Get("From"))
		if err != nil {
			log.Println("errors parsing fromAddress")
		}
		fromAddressAddress := fromAddress.Address
		subject := header.Get("Subject")

		subjectArgs := strings.SplitN(subject, " ", 2)
		if subjectArgs[0] != "rss-email" {
			log.Println("get one non-rss-email email")
			continue
		}

		log.Println("get one rss-email email")

		var command string
		if len(subjectArgs) == 2 {
			command = subjectArgs[1]
		}

		// process emails recieved
		if command == "subscribe" {
			slurp, err := parseMultipart(msg)
			if err != nil {
				if err := sendemail(config, fromAddressAddress, responseSubscribeSubjectFail, err.Error()); err != nil {
					log.Printf("error sendemail in subscribe response")
					continue
				}
				continue
			}

			re := regexp.MustCompile(httpRegex)
			urlStrs := re.FindAll(slurp, -1)
			var validUrls []string
			for _, urlStr := range urlStrs {
				urlStr = urlStr[:len(urlStr)-2]

				u, err := url.Parse(string(urlStr))
				if err != nil {
					log.Println(err)
					continue
				}

				validUrls = append(validUrls, u.String())
			}

			if len(validUrls) == 0 {
				if err := sendemail(config, fromAddressAddress, responseSubscribeSubjectFail, responseSubscribeBody0Url); err != nil {
					log.Printf("error sendemail in subscribe response")
					continue
				}
				continue
			}

			userSubscriptions.m[fromAddressAddress] = newUserSubscription()

			for _, url := range validUrls {
				(*userSubscriptions.m[fromAddressAddress])[url] = newUserURLInfo()
			}

			responseBody, err := userSubscriptions.m[fromAddressAddress].printToUser()
			if err != nil {
				log.Println("error printToUser")
				continue
			}

			if err := sendemail(config, fromAddressAddress, responseSubscribeSubject, responseBody); err != nil {
				log.Printf("error sendemail in subscribe response")
				continue
			}
			continue
		}
		if command == "list" {
			_, ok := userSubscriptions.m[fromAddressAddress]
			if !ok {
				if err := sendemail(config, fromAddressAddress, responseNotSubscribeSubject, responseNotSubscribeBody); err != nil {
					log.Printf("error sendemail in failed list response")
					continue
				}
				continue
			}

			responseBody, err := userSubscriptions.m[fromAddressAddress].printToUser()
			if err != nil {
				log.Println("error printToUser")
				continue
			}

			if err := sendemail(config, fromAddressAddress, responseListSubject, responseBody); err != nil {
				log.Printf("error sendemail in list response")
				continue
			}
			continue
		}
		if command == "unsubscribe" {
			_, ok := userSubscriptions.m[fromAddressAddress]
			if !ok {
				if err := sendemail(config, fromAddressAddress, responseNotSubscribeSubject, responseNotSubscribeBody); err != nil {
					log.Printf("error sendemail in failed unsubscribe response")
					continue
				}
			}

			delete(userSubscriptions.m, fromAddressAddress)

			if err := sendemail(config, fromAddressAddress, responseUnsubscribeSubject, ""); err != nil {
				log.Printf("error sendemail in unsubscribe response")
				continue
			}
			continue
		}

		if err := sendemail(config, fromAddressAddress, responseSubjectHelp, responseBodyHelp); err != nil {
			log.Printf("error sendemail in response help")
			continue
		}
	}

	if err := <-done; err != nil {
		return err
	}
	return nil
}

func parseMultipart(msg *mail.Message) ([]byte, error) {
	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(msg.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			slurp, err := ioutil.ReadAll(p)
			if err != nil {
				return nil, err
			}

			contentType := p.Header.Get("Content-Type")
			if strings.HasPrefix(contentType, "text/plain") {
				if strings.Contains(contentType, "UTF-8") || strings.Contains(contentType, "ascii") || strings.Contains(contentType, "iso-8859-1") {
					return slurp, nil
				}
				return nil, fmt.Errorf("error reveived email Content-Type: %s", contentType)
			}
		}

		return nil, errors.New("received email not valid mime multipart")
	}
	return nil, errors.New("received email not multipart content type")
}
