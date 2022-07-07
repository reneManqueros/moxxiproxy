package models

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

type User struct {
	UserID    string `yaml:"user_id"`
	AuthToken string `yaml:"auth_token"`
}

type Users struct{}

var UserMap map[string]User

func (u Users) Load(filename string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return
	}

	err = yaml.Unmarshal(data, &UserMap)
	if err != nil {
		log.Println(err)
	}
}

func (u Users) ByID(userID string) (User, bool) {
	user, ok := UserMap[userID]
	return user, ok
}
