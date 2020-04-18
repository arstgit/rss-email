package main

import (
	"bytes"
	"text/template"

	"github.com/mmcdole/gofeed"
)

type bodyParam struct {
	Feeds   []*gofeed.Feed
	Expect  int
	Actual  int
	ShowErr bool
}

func parsefeed(param *bodyParam) (string, error) {
	tmpl, err := template.ParseFiles("email-template.html")
	if err != nil {
		return "", err
	}

	out := &bytes.Buffer{}
	err = tmpl.Execute(out, param)
	if err != nil {
		return "", err
	}

	return string(out.Bytes()), nil
}
