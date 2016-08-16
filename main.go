package main

import (
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"sync"
	"time"
	"net/http"
	"net/smtp"
	"strings"
	"log"
	"bytes"
)

const (
	SystemConfig = "system.yml"
	WebsitesConfig = "websites.yml"
)

var (
	flagCreateTemplateConfigs = flag.Bool("create-template-configs", false, "Write to workdir example of config files")
)

func main(){
	flag.Parse()
	if *flagCreateTemplateConfigs {
		createTemplateConfigs()
		return
	}

	readBytes, err := ioutil.ReadFile(SystemConfig)
	if err != nil {
		createTemplateConfigs()
		os.Stderr.WriteString("Can't read system config: " + SystemConfig + "\n" + err.Error() + "\n")
		return
	}
	var systemConfig System
	err = yaml.Unmarshal(readBytes, &systemConfig)
	if err != nil {
		createTemplateConfigs()
		os.Stderr.WriteString("Can't parse system config: " + SystemConfig + "\n" + err.Error() + "\n")
	}

	readBytes, err = ioutil.ReadFile(WebsitesConfig)
	if err != nil {
		createTemplateConfigs()
		os.Stderr.WriteString("Can't read websites config: " + WebsitesConfig + "\n" + err.Error() + "\n")
		return
	}
	var websitesConfig []WebSite
	err = yaml.Unmarshal(readBytes, &websitesConfig)
	if err != nil {
		createTemplateConfigs()
		os.Stderr.WriteString("Can't parse websites config: " + WebsitesConfig + "\n" + err.Error() + "\n")
		return
	}

	var wait sync.WaitGroup
	for _, website := range websitesConfig{
		wait.Add(1)
		go func(){
			checkWebsite(systemConfig, website)
			wait.Done()
		}()
	}

	wait.Wait()
}

func checkWebsite(systemConfig System, website WebSite){
	httpClient := http.Client{}

	httpClient.Timeout = website.Timeout
	if httpClient.Timeout == 0 {
		httpClient.Timeout = systemConfig.Timeout
	}
	resp, err := httpClient.Get(website.URL)
	if err != nil {
		notify(systemConfig, website, "ERROR: " + website.URL, "Can't get page:\n" + err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte(website.ContainString)) {
		notify(systemConfig, website, "ERROR: " + website.URL, "It doesn't find the string:\n" + website.ContainString)
		return
	}
}

func createTemplateConfigs(){
	var system System
	system.EmailFrom = "from@mail.ru"
	system.EmailSmtpHost = "smtp.mail.ru"
	system.EmailSmtpPort = "25"
	system.EmailSmtpLogin = "from@mail.ru"
	system.EmailSmtpPassword = "1234"
	system.Timeout = time.Second * 10
	system.SendTo = []string{"aaa@bbb.com", "ccc@ddd.com"}

	out, err := yaml.Marshal(system)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(SystemConfig + ".example.yml", out, 0600)

	var website1 WebSite
	website1.URL = "http://example.com"
	website1.ContainString = "asd"
	website1.Timeout = 72 * time.Second
	website1.Description = "Short description"
	website1.SendTo = []string{"asd@mail.com", "test@gmail.com", "sss@ya.ru"}

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
	ioutil.WriteFile(WebsitesConfig + ".example.yml", out, 0600)
}

func notify(system System, website WebSite, subject, body string){
	body += "\n\n" + website.Description

	sendEmails(system, summStringsArrays(system.SendTo, website.SendTo), subject, body)
}

func sendEmails(system System, emails []string, subject, body string){
	message := `From: ` + system.EmailFrom + "\n"
	message += "To: " + strings.Join(emails, ",") + "\n"
	message += "Subject: " + subject + "\n"
	message += `Content-Type: text/plain; charset="utf-8"` + "\n"
	message += "\n"
	message += body

	auth := smtp.PlainAuth("", system.EmailSmtpLogin, system.EmailSmtpPassword, system.EmailSmtpHost)
	log.Println("Send email to:", emails)
	err := smtp.SendMail(system.EmailSmtpHost + ":" + system.EmailSmtpPort, auth, system.EmailFrom, emails, []byte(message))
	if err != nil {
		log.Println("Can't send email:", err)
	}
}

/*
Combine all strings from sarrs without duplicates.
Order of strings undefined.
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