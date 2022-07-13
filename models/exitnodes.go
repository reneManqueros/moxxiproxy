package models

import (
	"errors"
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

func (ps *ProxyServer) ExitNodesFromDisk(filename string) {
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		log.Println("no exitNodes file, creating blank")
		_, err = os.Create(filename)
		if err != nil {
			log.Fatalln("couldnt create exitNodes file")
		}
		return
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln("error loading exitNodes file", err)
		return
	}
	var exitNodes []ExitNode
	_ = yaml.Unmarshal(b, &exitNodes)
	ps.ExitNodes.All = exitNodes
	for _, v := range exitNodes {
		ps.ExitNodes.ByRegion[v.Region] = append(ps.ExitNodes.ByRegion[v.Region], v)
		ps.ExitNodes.ByInstanceID[v.InstanceID] = v
	}
}

func (ps *ProxyServer) ByRegion(region string) (ExitNode, error) {
	var err error
	if _, ok := ps.ExitNodes.ByRegion[region]; ok && len(ps.ExitNodes.ByRegion[region]) > 0 {
		slice := ps.ExitNodes.ByRegion[region]
		ps.Mutex.Lock()
		defer ps.Mutex.Unlock()
		if len(slice) >= 0 {

			randomIndex := rand.Intn(len(slice))
			return slice[randomIndex], nil
		}
		err = errors.New("no exitNodes available")

	}
	return ExitNode{}, err
}

func (ps *ProxyServer) ByRandom() (exitNode ExitNode, err error) {
	ps.Mutex.Lock()
	defer ps.Mutex.Unlock()
	if len(ps.ExitNodes.All) == 0 {
		err = errors.New("no exitNodes available")
		return
	}

	if ps.HideDown == false {
		randomIndex := rand.Intn(len(ps.ExitNodes.All))
		exitNode = ps.ExitNodes.All[randomIndex]
	} else {
		for i := 0; i < 10; i++ {
			randomIndex := rand.Intn(len(ps.ExitNodes.All))
			exitNode = ps.ExitNodes.All[randomIndex]
			if ServerHealth.IsUp(exitNode) == true {
				break
			}
		}
	}
	return
}

func (ps *ProxyServer) ByInstanceID(id string) (exitNode ExitNode, err error) {
	ps.Mutex.Lock()
	defer ps.Mutex.Unlock()
	if len(ps.ExitNodes.ByInstanceID) == 0 {
		err = errors.New("no exitNodes available")
		return
	}

	exitNode = ps.ExitNodes.ByInstanceID[id]
	return
}
