package models

import (
	"errors"
	"os"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

type BlockedURLs struct {
	Prefixes []string `yaml:"prefixes"`
	Suffixes []string `yaml:"suffixes"`
	Terms    []string `yaml:"terms"`
	Hosts    []string `yaml:"hosts"`
}

func (p *Proxy) BlockedURLsFromDisk() {
	if p.BlockedURLsFile == "" {
		return
	}
	if _, err := os.Stat(p.BlockedURLsFile); errors.Is(err, os.ErrNotExist) {
		log.Fatal().Err(err).Str("method", "BlockedURLsFromDisk").Msg("no valid blockedURLs file")
	}

	b, err := os.ReadFile(p.BlockedURLsFile)
	if err != nil {
		log.Fatal().Err(err).Str("method", "BlockedURLsFromDisk").Msg("loading blockedURLs file")
	}

	if err = yaml.Unmarshal(b, &p.BlockedURLs); err != nil {
		log.Fatal().Err(err).Msg("can't parse blockedURLs file")
	}
}

func (p *Proxy) ShouldBlock(hostname string) bool {
	if slices.Contains(p.BlockedURLs.Hosts, hostname) {
		return true
	}
	for _, v := range p.BlockedURLs.Prefixes {
		if strings.HasPrefix(hostname, v) {
			return true
		}
	}
	for _, v := range p.BlockedURLs.Suffixes {
		if strings.HasSuffix(hostname, v) {
			return true
		}
	}
	for _, v := range p.BlockedURLs.Terms {
		if strings.Contains(hostname, v) {
			return true
		}
	}
	return false
}
