package models

import (
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"os"
)

type Users struct{}

var UserMap map[string]string

func (u Users) Load(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal().Err(err).Str("filename", filename).Msg("Users.Load")
		return
	}

	err = yaml.Unmarshal(data, &UserMap)
	if err != nil {
		log.Fatal().Err(err).Str("filename", filename).Msg("Users.Load.Unmarshall")
	}
}
