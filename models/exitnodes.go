package models

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

type ExitNode struct {
	Interface  string `yaml:"interface"`
	Region     string `yaml:"region"`
	InstanceID string `yaml:"instance_id"`
}

func (p *Proxy) ExitNodesFromDisk() {
	if _, err := os.Stat("exitNodes.yml"); errors.Is(err, os.ErrNotExist) {
		log.Println("no exitNodes file, creating blank")
		_, err = os.Create("exitNodes.yml")
		if err != nil {
			log.Fatalln("couldnt create exitNodes file")
		}
		return
	}

	b, err := ioutil.ReadFile("exitNodes.yml")
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
