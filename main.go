package main

import (
	"bytes"
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"
	"fmt"
	"errors"
	"math/rand"
)

const (
	SystemConfigFileName   = "system.yml"
	WebsitesConfigFileName = "websites.yml"
	StatisticFileName      = "stat.yml"
	CheckTimeShiftDivider = 5 // CheckInterval / CheckTimeShiftDivider. 5 = 20% shift
)

var (
	flagCreateTemplateConfigs = flag.Bool("create-template-configs", false, "Write to workdir example of config files")
)

var (
	systemConfig System
	globalStatus Status
	websitesConfig []WebSite
)

func main() {
	flag.Parse()
	if *flagCreateTemplateConfigs {
		createTemplateConfigs()
		return
	}

	bytesArr, err := ioutil.ReadFile(SystemConfigFileName)
	if err != nil {
		createTemplateConfigs()
		log.Println("Can't read system config: " + SystemConfigFileName + "\n" + err.Error())
		return
	}

	// read system config
	err = yaml.Unmarshal(bytesArr, &systemConfig)
	if err != nil {
		createTemplateConfigs()
		log.Println("Can't parse system config: " + SystemConfigFileName + "\n" + err.Error())
	}

	// read websites config
	bytesArr, err = ioutil.ReadFile(WebsitesConfigFileName)
	if err != nil {
		createTemplateConfigs()
		log.Println("Can't read websites config: " + WebsitesConfigFileName + "\n" + err.Error())
		return
	}

	err = yaml.Unmarshal(bytesArr, &websitesConfig)
	if err != nil {
		createTemplateConfigs()
		log.Println("Can't parse websites config: " + WebsitesConfigFileName + "\n" + err.Error())
		return
	}

	// read statistic, ignore errors
	bytesArr, _ = ioutil.ReadFile(StatisticFileName)
	yaml.Unmarshal(bytesArr, &globalStatus)
	bytesArr = nil
	if globalStatus.Websites == nil {
		globalStatus.Websites = make(map[string]WebSiteStatus)
	}

	var wait sync.WaitGroup
	for _, website := range websitesConfig {
		wait.Add(1)
		go func(local WebSite) {
			checkWebsite(local)
			wait.Done()
		}(website)
	}

	wait.Wait()

	// If doesn't sent statistic today
	if globalStatus.LastTimeSendStatistic.YearDay() != time.Now().YearDay() {
		sendStatistic()
	}

	// save statistic
	globalStatus.Clean()

	bytesArr, err = yaml.Marshal(globalStatus)
	if err != nil {
		log.Println("Error while marshal statistic:", err)
	}
	err = ioutil.WriteFile(StatisticFileName, bytesArr, 0600)
	if err != nil {
		log.Println("Error while save statistic:", err)
	}
}

func checkWebsite(website WebSite) {
	// Calc if need check by time.
	globalStatus.mutex.Lock()
	state := globalStatus.Websites[website.URL]
	globalStatus.mutex.Unlock()

	checkInterval := systemConfig.CheckInterval
	if website.CheckInterval != 0 {
		checkInterval = website.CheckInterval
	}

	if checkInterval != 0 {
		var nextCheck time.Time
		var pause time.Duration
		emptyTime := time.Time{}
		if state.LastCheckTime == emptyTime {
			pause = time.Duration(rand.Int63n(int64(checkInterval)))
		} else {
			rnd := rand.New( rand.NewSource(state.LastCheckTime.UnixNano()))
			pauseShift := website.CheckInterval / CheckTimeShiftDivider * 2// *2 - for +/- shift
			pauseShift = time.Duration(rnd.Int63n(int64(pauseShift)))
			pauseShift -= website.CheckInterval / CheckTimeShiftDivider
			pause = website.CheckInterval + pauseShift
		}
		nextCheck = time.Now().Add(pause)
		if time.Now().Before(nextCheck) {
			return // Skip check by time interval
		}
	}

	ok, err := httpCheck(website.URL, website.ContainString, website.HttpStatusCode, website.Timeout)
	var errorText string
	if err != nil {
		errorText = err.Error()
	}
	notify(ok, website, errorText)
}

func httpCheck(url, needle string, httpCode int, timeout time.Duration)(bool, error){
	httpClient := http.Client{}

	// doesn't work with redirect when check status code
	if httpCode != 0 {
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	if httpClient.Timeout == 0 {
		httpClient.Timeout = systemConfig.Timeout
	} else {
		httpClient.Timeout = timeout
	}
	resp, err := httpClient.Get(url)
	if err != nil {
		return false, errors.New("Can't get page:\n"+err.Error())
	}
	defer resp.Body.Close()

	if httpCode != 0 {
		if resp.StatusCode != httpCode {
			return false, fmt.Errorf("Status code is '%v' instead of '%v'", resp.StatusCode, httpCode)
		}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte(needle)) {
		return false, errors.New("It doesn't find the string: '"+needle + "'")
	}

	return true, nil
}

func createTemplateConfigs() {
	var system System
	system.EmailFrom = "from@mail.ru"
	system.EmailSmtpHost = "smtp.mail.ru"
	system.EmailSmtpPort = "25"
	system.EmailSmtpLogin = "from@mail.ru"
	system.EmailSmtpPassword = "1234"
	system.Timeout = time.Second * 10
	system.SendTo = []string{"aaa@bbb.com", "ccc@ddd.com"}
	system.SendStatisticTo = []string{"aaa@bbb.com", "asdf@bbb.org"}
	system.SkipErrorsCount = 2
	system.CheckInterval = 0

	out, err := yaml.Marshal(system)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(SystemConfigFileName+".example.yml", out, 0600)

	var website1 WebSite
	website1.URL = "http://example.com"
	website1.ContainString = "asd"
	website1.Timeout = 72 * time.Second
	website1.Description = "Short description"
	website1.SendTo = []string{"asd@mail.com", "test@gmail.com", "sss@ya.ru"}
	website1.SendStatisticTo = []string{"asd@mail.com", "bob@jack.com"}
	website1.SkipErrorsCount = 5
	website1.CheckInterval = time.Hour*48 + time.Minute * 5

	var website2 WebSite
	website2.URL = "https://example2.com"
	website2.ContainString = "fff"
	website2.Description = `Long text description
with newlines
many
times
http://test.example2.com
`

	website3 := WebSite{}
	website3.URL = "http://example3.com"
	website3.HttpStatusCode = 301
	website3.Description = "Check redirect code"

	websites := []WebSite{website1, website2, website3}
	out, err = yaml.Marshal(websites)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(WebsitesConfigFileName+".example.yml", out, 0600)
}

func notify(ok bool, website WebSite, body string) {
	var subject string
	if ok {
		subject = "OK: " + website.URL
	} else {
		subject = "ERROR: " + website.URL
	}

	globalStatus.mutex.Lock()
	state := globalStatus.Websites[website.URL]
	globalStatus.mutex.Unlock()
	if !ok {
		state.LastErrorsCount++
	}

	state.LastCheckTime = time.Now()

	var skipErrorsCount int
	if website.SkipErrorsCount != 0 {
		skipErrorsCount = website.SkipErrorsCount
	} else {
		skipErrorsCount = systemConfig.SkipErrorsCount
	}

	if ok != state.OK {
		state.NotifyWasSent = false
	}
	if ok {
		state.LastErrorsCount = 0
	}

	/* send message when
	1. Check have OK result after fail checks (or check first time)
	2. Last n checks was error, where n more then skipErrorCount
	 */
	if !state.NotifyWasSent  &&  (ok || state.LastErrorsCount > skipErrorsCount) {
		state.OK = ok
		state.TextMessages = append(state.TextMessages, body)
		state.TimeMessages = append(state.TimeMessages, time.Now())
		state.SubjectMessages = append(state.SubjectMessages, subject)
		state.NotifyWasSent = true

		body += "\n\n" + website.Description
		sendEmails(summStringsArrays(systemConfig.SendTo, website.SendTo), subject, body)
	}
	globalStatus.mutex.Lock()
	globalStatus.Websites[website.URL] = state
	globalStatus.mutex.Unlock()
}

func sendEmails(emails []string, subject, body string) {
	message := `From: ` + systemConfig.EmailFrom + "\n"
	message += "To: " + strings.Join(emails, ",") + "\n"
	message += "Subject: " + subject + "\n"
	message += `Content-Type: text/plain; charset="utf-8"` + "\n"
	message += "\n"
	message += body

	auth := smtp.PlainAuth("", systemConfig.EmailSmtpLogin, systemConfig.EmailSmtpPassword, systemConfig.EmailSmtpHost)
	log.Printf("Send email '%v' to: %v\n", subject, emails)
	log.Println(systemConfig.EmailSmtpHost + ":" + systemConfig.EmailSmtpPort)
	err := smtp.SendMail(systemConfig.EmailSmtpHost+":"+systemConfig.EmailSmtpPort, auth, systemConfig.EmailFrom, emails, []byte(message))
	if err != nil {
		log.Println("Can't send email:", err)
	}
}

func sendStatistic() {
	messages := make(map[string]string)

	globalStatus.mutex.Lock()
	for _, website := range websitesConfig {
		state := globalStatus.Websites[website.URL]

		stringOk := "OK"
		if !state.OK {
			stringOk = "FAILED"
		}
		appendLines := []string{fmt.Sprintf("Website: %v; State: %v; Last check: %v;\n", website.URL, stringOk,
			state.LastCheckTime)}
		for i := range state.TextMessages {
			appendLines = append(appendLines, state.TimeMessages[i].String(), " ", state.SubjectMessages[i],
			": ", state.TextMessages[i], "\n")
		}
		appendLines = append(appendLines, "\n")
		appendMessage := strings.Join(appendLines, "")

		for _, email := range summStringsArrays(systemConfig.SendStatisticTo, website.SendStatisticTo) {
			messages[email] = messages[email] + appendMessage
		}
	}
	globalStatus.LastTimeSendStatistic = time.Now()
	globalStatus.mutex.Unlock()

	for email, mess := range messages {
		sendEmails([]string{email}, "MONITORING STAT", mess)
	}
}

/*
Combine all strings from sarrs without duplicates.
Order of result strings undefined.
*/
func summStringsArrays(sarrs ...[]string) []string {
	sMap := make(map[string]bool)
	for _, arr := range sarrs {
		for _, s := range arr {
			sMap[s] = true
		}
	}

	res := make([]string, 0, len(sMap))
	for k := range sMap {
		res = append(res, k)
	}
	return res
}
