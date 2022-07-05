package models

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
)

func (p *Proxy) GetBackendByRegion(region string) (b string, err error) {
	if _, ok := p.ExitNodes.ByRegion[region]; ok && len(p.ExitNodes.ByRegion[region]) > 0 {
		slice := p.ExitNodes.ByRegion[region]
		p.Mutex.Lock()
		defer p.Mutex.Unlock()
		if len(slice) == 0 {
			err := errors.New("no exitNodes available")
			return "", err
		}
		randomIndex := rand.Intn(len(slice))
		return slice[randomIndex].Interface, nil
	}
	return b, err
}

func (p *Proxy) GetBackend() (b string, err error) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	if len(p.Backends) == 0 {
		err = errors.New("no backends available")
		return
	}
	randomBackendIndex := rand.Intn(len(p.Backends))
	b = p.Backends[randomBackendIndex]
	return
}

func (p *Proxy) GetBackendByInstanceID(id string) (b string, err error) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	if len(p.ExitNodes.ByInstanceID) == 0 {
		err = errors.New("no exitNodes available")
		return
	}

	b = p.ExitNodes.ByInstanceID[id].Interface
	return
}

func (p *Proxy) Exists(backend string) bool {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	for _, b := range p.Backends {
		if b == backend {
			return true
		}
	}
	return false
}
func (p *Proxy) ToDisk() {
	b, err := yaml.Marshal(&p.Backends)
	if err != nil {
		log.Println(err)
	}

	err = ioutil.WriteFile(p.BackendsFile, b, 0644)
	if err != nil {
		log.Fatalln("error writing backends file", err)
	}
}

func (p *Proxy) FromDisk() {
	if _, err := os.Stat(p.BackendsFile); errors.Is(err, os.ErrNotExist) {
		log.Println("no backends file, creating blank")
		_, err = os.Create(p.BackendsFile)
		if err != nil {
			log.Fatalln("couldnt create backends file")
		}
		return
	}

	b, err := ioutil.ReadFile(p.BackendsFile)
	if err != nil {
		log.Fatalln("error loading backends file", err)
		return
	}
	var backends []string
	_ = yaml.Unmarshal(b, &backends)
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	p.Backends = backends
}

func (p *Proxy) Add(backend string) bool {
	backendsToAdd := []string{backend}
	if strings.Contains(backend, "/") == true {
		backends, err := ExpandCIDRRange(backend)
		if err != nil {
			log.Println("ERROR ExpandCIDRRange", err)
			return false
		}

		backendsToAdd = backends
	}
	addedBackends := false
	for _, backend := range backendsToAdd {
		if p.Exists(backend) == false {
			p.Mutex.Lock()
			p.Backends = append(p.Backends, backend)
			p.ToDisk()
			p.Mutex.Unlock()
			addedBackends = true
		}
	}

	return addedBackends
}

func (p *Proxy) Remove(backend string) bool {
	backendsToRemove := []string{backend}
	if strings.Contains(backend, "/") == true {
		backends, err := ExpandCIDRRange(backend)
		if err != nil {
			log.Println("ERROR ExpandCIDRRange", err)
			return false
		}
		backendsToRemove = backends
	}
	removedBackends := false

	for _, backend := range backendsToRemove {
		if p.Exists(backend) == true {
			p.Mutex.Lock()

			var tmpBackends []string
			for _, b := range p.Backends {
				if b != backend {
					tmpBackends = append(tmpBackends, b)
				}
			}
			p.Backends = tmpBackends
			p.ToDisk()
			p.Mutex.Unlock()
			removedBackends = true
		}
	}
	return removedBackends
}

func (p *Proxy) getBackends() {
	data, _ := ioutil.ReadFile(p.BackendsFile)
	err := yaml.Unmarshal(data, &p.Backends)
	if err != nil {
		log.Fatalln(err)
	}
}
