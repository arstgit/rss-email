package main

import (
	"errors"
	"fmt"
	"net/smtp"
)

type loginAuth struct {
	username, secret string
	stage            int
}

// LOGINAuth is not supported in standard packages, thereby we implement it here.
func LOGINAuth(username, secret string) smtp.Auth {
	return &loginAuth{username, secret, 0}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		if a.stage == 0 {
			a.stage = 1
			return []byte(fmt.Sprintf("%s", a.username)), nil
		}
		if a.stage == 1 {
			return []byte(fmt.Sprintf("%s", a.secret)), nil
		}
		return nil, errors.New("loginAuth unexpected stage")
	}
	return nil, nil
}
