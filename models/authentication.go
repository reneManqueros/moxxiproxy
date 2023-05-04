package models

import (
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"os"
)

type User struct {
	UserID    string `yaml:"user_id"`
	AuthToken string `yaml:"auth_token"`
}

type Users struct{}

var UserMap map[string]User

func (u Users) Load(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Err(err).Str("filename", filename).Msg("Users.Load")
		return
	}

	err = yaml.Unmarshal(data, &UserMap)
	if err != nil {
		log.Err(err).Str("filename", filename).Msg("Users.Load.Unmarshall")
	}
}

func (u Users) ByID(userID string) (User, bool) {
	user, ok := UserMap[userID]
	return user, ok
}
