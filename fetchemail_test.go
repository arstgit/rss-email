package main

import (
	"net/mail"
	"strings"
	"testing"
)

const testSrc = `
--0000000000003d878605a3bb155f
Content-Type: text/plain; charset="UTF-8"

https://golang.org/pkg/time/#Time

--0000000000003d878605a3bb155f
Content-Type: text/html; charset="UTF-8"
Content-Transfer-Encoding: quoted-printable

<div dir=3D"ltr"><a href=3D"https://golang.org/pkg/time/#Time" rel=3D"noref=
errer" target=3D"_blank">https://golang.org/pkg/time/#Time</a><div class=3D=
"gmail-adL"><br></div><div class=3D"gmail-adL"><a href=3D"https://golang.or=
g/pkg/time/#Time" rel=3D"noreferrer" target=3D"_blank">https://golang.org/p=
kg/time/#Time</a><div class=3D"gmail-adL"><a href=3D"https://github.com/der=
ekchuank/rss-email">https://github.com/derekchuank/rss-email</a></div><div =
class=3D"gmail-adL"><br></div></div></div>

--0000000000003d878605a3bb155f--
`

func Test_parseMultipart(t *testing.T) {
	cases := []struct {
		in   *mail.Message
		want string
	}{
		{&mail.Message{
			Header: map[string][]string{
				"Content-Type": {`multipart/alternative; boundary="0000000000003d878605a3bb155f"`},
			},
			Body: strings.NewReader(testSrc),
		}, "https://golang.org/pkg/time/#Time\n"},
	}
	for _, c := range cases {
		got, err := parseMultipart(c.in)
		if err != nil {
			t.Errorf("error %q", err)
		}
		if string(got) != c.want {
			t.Errorf("parseMultipart(%q) == %q, want %q", c.in, got, c.want)
		}
	}
}
