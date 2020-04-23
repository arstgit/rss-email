package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// the interval checking email inbox, in seconds
const fetchemailInterval = 5 * 60

// the interval fetching RSS feed
const fetchfeedInterval = 30 * 60

// the interval printing running info
const statsInterval = 20

type emailConfig struct {
	from       string
	smtpServer string
	imapServer string
	username   string
	password   string
}

var userSubscriptions = newUserSubscriptions()
var subscription = newSubscription()

func main() {
	var config emailConfig
	var sendemailInterval int

	flag.StringVar(&config.from, "email", "", "`email` address serving rss-email service")
	flag.StringVar(&config.smtpServer, "smtpServer", "", "smtp mail relay, `server[:port]`")
	flag.StringVar(&config.imapServer, "imapServer", "", "imap server, `server[:port]`")
	flag.StringVar(&config.username, "username", "", "authentication `user` (for SMTP/IMAP authentication)")
	flag.StringVar(&config.password, "password", "", "authentication `password` (for SMTP/IMAP authentication)")

	flag.IntVar(&sendemailInterval, "sendemailInterval", 10, "specify email sending interval, in `minutes`")

	flag.Parse()

	if err := verifyConfig(&config); err != nil {
		flag.Usage()
		os.Exit(0)
	}
	if err := userSubscriptions.restoreFromDisk(); err != nil {
		log.Panic("error restore from disk", err)
	}
	log.Print("user info restored from file")

	fetchemailTicker := time.NewTicker(fetchemailInterval * time.Second)
	fetchfeedTicker := time.NewTicker(fetchfeedInterval * time.Second)
	sendemailTicker := time.NewTicker(time.Duration(sendemailInterval) * time.Minute)
	statsTicker := time.NewTicker(statsInterval * time.Second)
	defer fetchemailTicker.Stop()
	defer fetchfeedTicker.Stop()
	defer sendemailTicker.Stop()
	defer statsTicker.Stop()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// make sure all goroutines finish executing when main goroutine is adout to exit
	var wg sync.WaitGroup

	// only one job each type is executing
	var statsRunning = false
	var fetchemailRunning = false
	var fetchfeedRunning = false
	var sendemailRunning = false
	for {
		select {
		case signal := <-signalChan:
			fmt.Printf("signal %v received, waiting goroutines finish\n", signal)
			wg.Wait()

			if err := userSubscriptions.saveToDisk(); err != nil {
				log.Panicln("error save to disk")
			}
			log.Println("user info saved to disk")

			os.Exit(0)
		case <-statsTicker.C:
			go func() {
				wg.Add(1)
				defer wg.Done()

				if statsRunning {
					return
				}
				statsRunning = true
				defer func() { statsRunning = false }()

				userSubscriptions.RLock()
				defer func() { userSubscriptions.RUnlock() }()

				subscription.RLock()
				defer func() { subscription.RUnlock() }()

				log.Printf("user count: %v, subscription count: %v\n", len(userSubscriptions.m), len(subscription.m))
			}()
		case <-fetchemailTicker.C:
			go func() {
				wg.Add(1)
				defer wg.Done()

				if fetchemailRunning {
					return
				}
				fetchemailRunning = true
				defer func() { fetchemailRunning = false }()

				userSubscriptions.Lock()
				defer func() { userSubscriptions.Unlock() }()

				log.Println("fetchemail ...")
				if err := fetchemail(&config); err != nil {
					log.Println(err)
				}

				if err := userSubscriptions.saveToDisk(); err != nil {
					log.Panicln("error save to disk")
				}
				log.Println("user info saved to disk")
			}()
		case <-fetchfeedTicker.C:
			go func() {
				wg.Add(1)
				defer wg.Done()

				if fetchfeedRunning {
					return
				}
				fetchfeedRunning = true
				defer func() { fetchfeedRunning = false }()

				for _, v := range userSubscriptions.m {
					for url := range *v {
						if _, ok := subscription.m[url]; !ok {
							subscription.m[url] = newURLInfo()
						}
					}
				}

				log.Println("fetchfeed ...")
				if err := fetchfeed(&config); err != nil {
					log.Println(err)
				}
			}()
		case <-sendemailTicker.C:
			go func() {
				wg.Add(1)
				defer wg.Done()

				if sendemailRunning {
					return
				}
				sendemailRunning = true
				defer func() { sendemailRunning = false }()

				userSubscriptions.Lock()
				defer func() { userSubscriptions.Unlock() }()

				subscription.RLock()
				defer func() { subscription.RUnlock() }()

				log.Println("sendemail ...")
				if err := sendSubscription(&config); err != nil {
					log.Println(err)
				}
			}()
		}

	}
}

func verifyConfig(config *emailConfig) error {
	if config.from == "" {
		return errors.New("missing flag")
	}
	if config.imapServer == "" {
		return errors.New("missing flag")
	}
	if config.smtpServer == "" {
		return errors.New("missing flag")
	}
	if config.username == "" {
		return errors.New("missing flag")
	}
	if config.password == "" {
		return errors.New("missing flag")
	}
	return nil
}
