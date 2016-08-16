package main

import (
	"sync"
	"time"
)

type System struct {
	EmailFrom         string        `yaml:"EmailFrom"`
	EmailSmtpHost     string        `yaml:"EmailSmtpHost"`
	EmailSmtpPort     string        `yaml:"EmailSmtpPort"`
	EmailSmtpLogin    string        `yaml:"EmailSmtpLogin"`
	EmailSmtpPassword string        `yaml:"EmailSmtpPassword"`
	SendTo            []string      `yaml:"SendTo"`
	SendStatisticTo   []string      `yaml:"SendStatisticTo"`
	Timeout           time.Duration `yaml:"Timeout"`
}

type WebSite struct {
	URL             string        `yaml:"URL"`
	ContainString   string        `yaml:"ContainString"`
	SendTo          []string      `yaml:"SendTo,omitempty"`
	SendStatisticTo []string      `yaml:"SendStatisticTo"`
	Timeout         time.Duration `yaml:"Timeout,omitempty"`
	Description     string        `yaml:"Description"`
}

type WebSiteStatus struct {
	TextMessages    []string
	SubjectMessages []string
	TimeMessages    []time.Time
	OK              bool
	NotFirstTime    bool
	LastCheckTime   time.Time
}

type Status struct {
	mutex                 sync.Mutex
	Websites              map[string]WebSiteStatus
	LastTimeSendStatistic time.Time
}

func (this *Status) Clean() {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	oldDate := time.Now().Add(-time.Hour * 48)

	newMap := make(map[string]WebSiteStatus)
	for url, state := range this.Websites {
		// skip removed sites
		if state.LastCheckTime.Before(oldDate) {
			continue
		}

		state.SubjectMessages = nil
		state.TextMessages = nil
		state.TimeMessages = nil

		newMap[url] = state
	}
}
