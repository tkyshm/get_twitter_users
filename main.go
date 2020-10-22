package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

var (
	consumerKey    string
	consumerSecret string
	accessToken    string
	accessSecret   string
)

func init() {
	consumerKey = os.Getenv("TWITTER_CONSUMER_KEY")
	consumerSecret = os.Getenv("TWITTER_CONSUMER_SECRET")
	accessToken = os.Getenv("TWITTER_ACCESS_TOKEN")
	accessSecret = os.Getenv("TWITTER_ACCESS_SECRET")

}

const (
	XRateLimitRemaining = "X-Rate-Limit-Remaining"
	XRateLimitReset     = "X-Rate-Limit-Reset"
)

func main() {
	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)

	hc := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(hc)
	b := bufio.NewScanner(os.Stdin)
	f, err := os.OpenFile("users.tsv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("[error] failed to os.OpenFile:", err)
		return
	}
	defer f.Close()
	for b.Scan() {
		id, err := strconv.ParseInt(b.Text(), 10, 64)
		if err != nil {
			log.Println("[error] ", err)
			continue
		}

		user, resp, err := client.Users.Show(&twitter.UserShowParams{
			UserID: id,
		})
		if err != nil {
			log.Println("[error] ", err)
			continue
		}
		defer resp.Body.Close()

		remaining := resp.Header.Get(XRateLimitRemaining)
		if remaining == "0" {
			log.Println("exceeded limit rate")
			waitUntilResetTime(resp.Header)
		}

		if resp.StatusCode >= 400 {
			var b []byte
			if _, err := resp.Body.Read(b); err != nil {
				log.Println("failed resp.Body.Read:", err)
				continue
			}
			log.Println(string(b))
			continue
		}

		log.Printf("id:%s name:%s screen_name:%s email:%s description:%s", user.IDStr, user.Name, user.ScreenName, user.Email, user.Description)
		// NOTE: ここはfileじゃなくてDBでも良さそう？
		_, err = f.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n", user.IDStr, user.Name, user.ScreenName, user.Email, strings.ReplaceAll(user.Description, "\n", " ")))
		if err != nil {
			log.Println(err)
		}

		time.Sleep(250 * time.Millisecond)
	}
}

func waitUntilResetTime(header http.Header) {
	resetEpoch := header.Get(XRateLimitReset)
	nextTime, _ := strconv.ParseInt(resetEpoch, 10, 64)
	t := time.Unix(nextTime, 0)
	log.Println("waiting reset time:", t.String())
	now := time.Now().Unix()
	time.Sleep(time.Duration(nextTime-now) * time.Second)
}
