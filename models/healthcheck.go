package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type InstanceCheck struct {
	Instance        string `json:"instance"`
	LastUsed        int64  `json:"last_used"`
	LastKnownStatus bool   `json:"last_known_status"`
}

type HealthCheck struct {
	Instances map[string]InstanceCheck
	Mutex     *sync.Mutex
}

var ServerHealth HealthCheck

func (hc *HealthCheck) Init() {
	hc.Instances = make(map[string]InstanceCheck)
	hc.Mutex = &sync.Mutex{}
}

func (hc *HealthCheck) ToSlice() []InstanceCheck {
	hc.Mutex.Lock()
	defer hc.Mutex.Unlock()

	var instances []InstanceCheck
	for _, instance := range hc.Instances {
		instances = append(instances, instance)
	}
	return instances
}

func (hc *HealthCheck) ToJSON() ([]byte, error) {
	return json.Marshal(hc.ToSlice())
}

func (hc *HealthCheck) ToCSV() ([]byte, error) {
	csv := ""
	hc.Mutex.Lock()
	for _, instance := range hc.Instances {
		csv += fmt.Sprintf(`%s,%v,%v`, instance.Instance, instance.LastUsed, instance.LastKnownStatus) + "\n"
	}
	hc.Mutex.Unlock()
	return []byte(csv), nil
}

func (hc *HealthCheck) ToHTML() ([]byte, error) {
	htmlTable := `<link href="https://getbootstrap.com/docs/4.0/dist/css/bootstrap.min.css" rel="stylesheet" />
<link href="https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/4.5.2/css/bootstrap.css" rel="stylesheet" />
<link href="https://cdn.datatables.net/1.12.1/css/dataTables.bootstrap4.min.css" rel="stylesheet" />
					<div class="container-fluid">
				<table class="table table-hover">
				  <thead>
					<tr>
					  <th scope="col">Instance</th>
					  <th scope="col">Last Used</th>
					  <th scope="col">Last Known Status</th>
					</tr>
				  </thead>
				  <tbody>`
	hc.Mutex.Lock()
	for _, instance := range hc.Instances {
		lastUsed := time.Unix(instance.LastUsed, 0).Format(`2006-01-02T15:04:05`)
		lastKnownStatus := `<td class="table-success">Up</td>`
		if instance.LastKnownStatus == false {
			lastKnownStatus = `<td class=" table-danger">Down</td>`
		}
		htmlTable += fmt.Sprintf(` <tr><td>%s</td><td>%s</td>%s</tr>`,
			instance.Instance,
			lastUsed,
			lastKnownStatus,
		)
	}
	hc.Mutex.Unlock()
	htmlTable += `</tbody></table></div>
<script src="https://code.jquery.com/jquery-3.5.1.js" ></script>
<script src="https://cdn.datatables.net/1.12.1/js/jquery.dataTables.min.js" ></script>
<script src="https://cdn.datatables.net/1.12.1/js/dataTables.bootstrap4.min.js" ></script>
<script>
    jQuery('table').DataTable({});
</script>

`
	return []byte(htmlTable), nil
}

func (hc *HealthCheck) ReviveOld() {
	hc.Mutex.Lock()
	defer hc.Mutex.Unlock()
	for key, instance := range hc.Instances {
		if instance.LastKnownStatus == false && instance.LastUsed-time.Now().Unix() > 1800 {
			delete(hc.Instances, key)
		}
	}
}

func (hc *HealthCheck) IsUp(exitNode ExitNode) bool {
	instance := exitNode.Interface
	if exitNode.Upstream != "" {
		instance = exitNode.Upstream
	}
	hc.Mutex.Lock()
	defer hc.Mutex.Unlock()
	val, ok := hc.Instances[instance]

	// if its not on the health check, then try it
	if !ok {
		return true
	}

	// if its on the health check and it was up, use it
	if ok && val.LastKnownStatus == true {
		return true
	}

	return false
}

func (hc *HealthCheck) SetFromError(exitNode ExitNode, err error) {
	isUp := true
	if err != nil && strings.Contains(err.Error(), "use of closed network connection") == false {
		isUp = false
	}
	hc.Set(exitNode, isUp)
}

func (hc *HealthCheck) Set(exitNode ExitNode, isUp bool) {
	instance := exitNode.Interface
	if exitNode.Upstream != "" {
		instance = exitNode.Upstream
	}

	lastUsed := time.Now().Unix()

	ic := InstanceCheck{
		Instance:        instance,
		LastUsed:        lastUsed,
		LastKnownStatus: isUp,
	}

	hc.Mutex.Lock()
	defer hc.Mutex.Unlock()
	hc.Instances[instance] = ic
}
