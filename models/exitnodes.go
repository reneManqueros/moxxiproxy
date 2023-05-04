package models

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
)

type ExitNode struct {
	Interface  string `yaml:"interface"`
	Region     string `yaml:"region"`
	InstanceID string `yaml:"instance_id"`
	Upstream   string `yaml:"upstream"`
}

func (p *Proxy) ExitNodesFromDisk() {
	if _, err := os.Stat(p.ExitNodesFile); errors.Is(err, os.ErrNotExist) {
		log.Fatalln("no exitNodes file")
	}

	b, err := ioutil.ReadFile(p.ExitNodesFile)
	if err != nil {
		log.Fatalln("error loading exitNodes file", err)
		return
	}
	var exitNodes []ExitNode
	_ = yaml.Unmarshal(b, &exitNodes)
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	p.ExitNodes.All = exitNodes
	for _, v := range exitNodes {
		p.ExitNodes.ByRegion[v.Region] = append(p.ExitNodes.ByRegion[v.Region], v)
		p.ExitNodes.ByInstanceID[v.InstanceID] = v
	}
}

func (p *Proxy) ByRegion(region string) (ExitNode, error) {
	var err error
	if _, ok := p.ExitNodes.ByRegion[region]; ok && len(p.ExitNodes.ByRegion[region]) > 0 {
		slice := p.ExitNodes.ByRegion[region]
		p.Mutex.Lock()
		defer p.Mutex.Unlock()
		if len(slice) >= 0 {

			randomIndex := rand.Intn(len(slice))
			return slice[randomIndex], nil
		}
		err = errors.New("no exitNodes available")
	}
	return ExitNode{}, err
}
func (p *Proxy) BySession(userID string, session string) (ExitNode, error) {
	var err error
	sessionKey := fmt.Sprintf(`%s-%s`, userID, session)
	if exitNode, ok := p.Sessions[sessionKey]; ok == true {
		return exitNode, nil
	}
	if exitNode, err := p.ByRandom(); err == nil {
		p.Sessions[sessionKey] = exitNode
		return exitNode, nil
	}
	return ExitNode{}, err
}

func (p *Proxy) ByRandom() (exitNode ExitNode, err error) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	if len(p.ExitNodes.All) == 0 {
		err = errors.New("no exitNodes available")
		return
	}

	randomIndex := rand.Intn(len(p.ExitNodes.All))
	exitNode = p.ExitNodes.All[randomIndex]

	return
}

func (p *Proxy) ByInstanceID(id string) (exitNode ExitNode, err error) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	if len(p.ExitNodes.ByInstanceID) == 0 {
		err = errors.New("no exitNodes available")
		return
	}

	exitNode = p.ExitNodes.ByInstanceID[id]
	return
}
