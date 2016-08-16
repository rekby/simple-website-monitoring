package main

import "time"

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

type Status struct {

}