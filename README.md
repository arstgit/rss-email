# rss-email

[![Build Status](https://travis-ci.org/derekchuank/rss-email.svg?branch=master)](https://travis-ci.org/derekchuank/rss-email)

## Demo email address

rss-demo@outlook.com

## Usage

Run the server yourself:
```
$ go get github.com/derekchuank/rss-email
$ rss-email -email your-email -smtpServer outlook.office365.com:587 -username your-email -password your-password  -imapServer=outlook.office365.com:993 -sendemailInterval 10
```

Or just use the demo email I provided.

## Subscribe your interested RSS

![rss-email](https://ftp.bmp.ovh/imgs/2020/04/b0b40eef0471e789.png)

Send one email to your-email, or the demo email if you haven't run the server, with subject: `rss-email subscribe`, write your RSS URLs in the message body, newline seperated.

Wait for your feed, don't forget to check the Junk inbox.

## Other operations

- Unsubscribe. Send email with subject: `rss-email unsubscribe`.
- List your subscribed RSS. Send email with subject: `rss-email list`.

### See also

[https://www.tiaoxingyubolang.com/article/2020-04-23_rss-email](https://www.tiaoxingyubolang.com/article/2020-04-23_rss-email)