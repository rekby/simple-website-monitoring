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
)

const (
	SystemConfigFileName   = "system.yml"
	WebsitesConfigFileName = "websites.yml"
	StatisticFileName      = "stat.yml"
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
		go func() {
			checkWebsite(website)
			wait.Done()
		}()
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
	httpClient := http.Client{}

	httpClient.Timeout = website.Timeout
	if httpClient.Timeout == 0 {
		httpClient.Timeout = systemConfig.Timeout
	}
	resp, err := httpClient.Get(website.URL)
	if err != nil {
		notify(false, website, "ERROR: "+website.URL, "Can't get page:\n"+err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte(website.ContainString)) {
		notify(false, website, "ERROR: "+website.URL, "It doesn't find the string:\n"+website.ContainString)
		return
	}

	notify(true, website, "OK: "+website.URL, "OK")
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

	var website2 WebSite
	website2.URL = "https://example2.com"
	website2.ContainString = "fff"
	website2.Description = `Long text description
with newlines
many
times`

	websites := []WebSite{website1, website2}
	out, err = yaml.Marshal(websites)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(WebsitesConfigFileName+".example.yml", out, 0600)
}

func notify(ok bool, website WebSite, subject, body string) {
	globalStatus.mutex.Lock()
	state := globalStatus.Websites[website.URL]
	globalStatus.mutex.Unlock()

	state.LastCheckTime = time.Now()

	if state.OK != ok || !state.NotFirstTime {
		state.OK = ok
		state.TextMessages = append(state.TextMessages, body)
		state.TimeMessages = append(state.TimeMessages, time.Now())
		state.SubjectMessages = append(state.SubjectMessages, subject)
		state.NotFirstTime = true

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
	log.Println("Send email to:", emails)
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
		appendMessage := fmt.Sprintf("Website: %v; State: %v; Last check: %v; Messages: %v\n", website.URL, stringOk,
			state.LastCheckTime, strings.Join(state.TextMessages, ", "))

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
