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
	"net/mail"
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
		notify(systemConfig, website, "ERROR: " + website.URL, "Не найдена строка:\n" + website.ContainString)
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

	var website2 WebSite
	website2.URL = "https://example2.com"
	website2.ContainString = "fff"
	website2.SendTo = []string{"asd@mail.com", "test@gmail.com", "sss@ya.ru"}

	websites := []WebSite{website1, website2}
	out, err = yaml.Marshal(websites)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(WebsitesConfig + ".example.yml", out, 0600)
}

func notify(system System, website WebSite, subject, body string){

	emails := website.SendTo
	if len(emails) == 0 {
		emails = system.SendTo
	}

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

func encodeRFC2047(String string) string{
	// use mail's rfc2047 to encode any string
	// copy from https://gist.github.com/andelf/5004821
	addr := mail.Address{String, ""}
	return strings.Trim(addr.String(), " <>")
}