package main

import (
	"time"
	"sync"
)

type System struct {
	EmailFrom         string `yaml:"EmailFrom"`
	EmailSmtpHost     string `yaml:"EmailSmtpHost"`
	EmailSmtpPort     string `yaml:"EmailSmtpPort"`
	EmailSmtpLogin    string `yaml:"EmailSmtpLogin"`
	EmailSmtpPassword string `yaml:"EmailSmtpPassword"`
	SendTo            []string `yaml:"SendTo"`
	Timeout           time.Duration `yaml:"Timeout"`
}

type WebSite struct {
	URL           string `yaml:"URL"`
	ContainString string `yaml:"ContainString"`
	SendTo        []string `yaml:"SendTo,omitempty"`
	Timeout time.Duration `yaml:"Timeout,omitempty"`
	Description string `yaml:"Description"`
}

type WebSiteStatus struct {
	TextMessages []string
	SubjectMessages []string
	TimeMessages []time.Time
	OK bool
	NotFirstTime bool
}

type Status struct {
	mutex sync.Mutex
	Websites map[string] WebSiteStatus
}

func (this *Status) Reset(){
	this.mutex.Lock()
	defer this.mutex.Unlock()

	this.Websites = make(map[string]WebSiteStatus)
}