// Package main provides ...
package main

import (
	"fmt"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/Sirupsen/logrus"
	"github.com/franela/goreq"
	"github.com/nlopes/slack"
)

const (
	UBOT_ODDDAY_URL = "https://card.ubot.com.tw/eCard/activity_login/oddDay2016.aspx"
	TITLE           = "聯邦奇數日檢查"
)

func init() {
	// Output to stderr instead of stdout, could also be a file.
	log.SetOutput(os.Stderr)

	// Only log the warning severity or above.
	//log.SetLevel(log.DebugLevel)
}

func main() {

	slack_token := os.Getenv("SLACK_TOKEN")
	slack_group := os.Getenv("SLACK_GROUP")
	card_no := os.Getenv("CARD_NO")
	if slack_token == "" || slack_group == "" || card_no == "" {
		log.Fatal("SLACK_TOKEN=xxxx-1111111111-1111111111-11111111111-111111 SLACK_GROUP=GXXXXXXXX CARD_NO=1111222233334444 ./go-ubot-oddday-checker")
	}

	res, err := goreq.Request{Uri: UBOT_ODDDAY_URL}.Do()
	if err != nil {
		log.Fatal(err)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	tbCode, found := doc.Find("#tbCode").Attr("value")
	if !found {
		log.Error("Cannot find tbCode.")
		return
	}

	viewstate, found := doc.Find("#__VIEWSTATE").Attr("value")
	if !found {
		log.Error("Cannot find viewstate.")
		return
	}

	eventvalidation, found := doc.Find("#__EVENTVALIDATION").Attr("value")
	if !found {
		log.Error("Cannot find eventvalidation.")
		return
	}

	form := url.Values{}
	form.Add("__EVENTTARGET", "")
	form.Add("__EVENTARGUMENT", "")
	form.Add("__VIEWSTATE", viewstate)
	form.Add("tbCode", tbCode)
	form.Add("__CALLBACKID", "__Page")
	form.Add("__CALLBACKPARAM", "QRY%%"+card_no+"%%"+tbCode+"%%"+tbCode+"%%")
	form.Add("__EVENTVALIDATION", eventvalidation)

	res, err = goreq.Request{
		Method:      "POST",
		Uri:         UBOT_ODDDAY_URL,
		UserAgent:   "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:43.0) Gecko/20100101 Firefox/43.0",
		ContentType: "application/x-www-form-urlencoded",
		Body:        form.Encode(),
	}.WithHeader("Referer", UBOT_ODDDAY_URL).Do()
	if err != nil {
		log.Fatal(err)
	}
	body, _ := res.Body.ToString()

	// Parse the HTML into nodes
	rp := regexp.MustCompile(`LOGINOK@@[^@]+@@([^@]+)@@[^@]+`)
	m := rp.FindStringSubmatch(body)
	if m == nil {
		log.Fatalf("Cannot find expected response: %s", body)
	}

	log.Debugf("Response: %s", m[1])

	doc, err = goquery.NewDocumentFromReader(strings.NewReader(m[1]))
	if err != nil {
		log.Fatal(err)
	}

	api := slack.New(slack_token)
	mParams := slack.PostMessageParameters{}
	attachment := slack.Attachment{}

	doc.Find("tr").Each(func(i int, s *goquery.Selection) {
		sel := s.Find("td")
		month := strings.TrimSpace(sel.Nodes[0].FirstChild.Data)
		count := strings.TrimSpace(sel.Nodes[1].FirstChild.Data)
		money := strings.TrimSpace(sel.Nodes[2].FirstChild.Data)
		log.Debugf("%s,%s,%s", month, count, money)
		field := slack.AttachmentField{
			Title: month,
			Value: count + " (" + money + ")",
		}
		attachment.Fields = append(attachment.Fields, field)
	})

	// Query all logs in past 1 month
	hParams := slack.NewHistoryParameters()
	hParams.Oldest = fmt.Sprint(time.Now().AddDate(0, -1, 0).Unix())
	history, err := api.GetGroupHistory(slack_group, hParams)
	if err != nil {
		log.Fatal(err)
	}
	for _, msg := range history.Messages {
		if msg.Text == TITLE {
			for _, _attachement := range msg.Attachments {
				// Compare attachment
				if reflect.DeepEqual(_attachement, attachment) {
					log.Debug("Found exist in slack. Skip.")
					return
				}
			}
		}
	}

	// Notify new message
	mParams.Attachments = []slack.Attachment{attachment}
	_, _, err = api.PostMessage(slack_group, TITLE, mParams)
	if err != nil {
		log.Fatal(err)
	}
}
