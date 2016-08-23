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
	SkipErrorsCount   int           `yaml:"SkipErrorsCount"`
	CheckInterval     time.Duration `yaml:"CheckInterval"`
}

type WebSite struct {
	URL             string        `yaml:"URL"`
	ContainString   string        `yaml:"ContainString,omitempty"`
	SendTo          []string      `yaml:"SendTo,omitempty"`
	SendStatisticTo []string      `yaml:"SendStatisticTo,omitempty"`
	Timeout         time.Duration `yaml:"Timeout,omitempty"`
	Description     string        `yaml:"Description,omitempty"`
	HttpStatusCode  int           `yaml:"HttpStatusCode,omitempty"`
	SkipErrorsCount int           `yaml:"SkipErrorsCount,omitempty"`
	CheckInterval   time.Duration `yaml:"CheckInterval,omitempty"`
}

type WebSiteStatus struct {
	TextMessages    []string
	SubjectMessages []string
	TimeMessages    []time.Time
	OK              bool
	NotifyWasSent   bool
	NotifyOkWasSent bool
	LastCheckTime   time.Time
	LastErrorsCount int
}

type Status struct {
	mutex                 sync.Mutex
	Websites              map[string]WebSiteStatus
	LastTimeSendStatistic time.Time
}

func (this *Status) Clean() {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	oldDate := time.Now().Add(-time.Hour * 24)

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
