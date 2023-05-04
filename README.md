# Moxxi Proxy

#### A REALLY fast proxy manager, battle tested with 200K concurrent connections on mid-sized (8 core + 16gb) hardware. 

## Features 
- [x] Maps to network interfaces 
- [x] Low resource footprint 
- [x] Basic Authentication
- [x] Client whitelist 
- [x] Random load balance selection
- [x] Session stickiness
- [x] Group backends by regions
- [x] Logs

## Installation
Download binary:
```shell
wget https://github.com/reneManqueros/moxxiproxy/releases/download/v1.2.9/moxxiproxy_1.2.9_Linux_x86_64.tar.gz && tar xf moxxiproxy_1.2.9_Linux_x86_64.tar.gz 
```

Or download source and compile:
```shell
git clone https://github.com/reneManqueros/moxxiproxy.git && cd moxxiproxy && make build
````

## Usage
 
```shell
./moxxiproxy run
```
 
## Parameters

| Flag       | Use                                                                     | Default         |
|------------|-------------------------------------------------------------------------|-----------------|
| address    | Set the listen address                                                  | 0.0.0.0:1989    |         
| exitnodes  | Path to config file                                                     | ./exitNodes.yml |         
| auth       | user/password for authentication                                        | <empty>         |         
| whitelist  | IP's to allow to use, allows all if blank                               | <empty>         |         
| timeout    | default timeout seconds for backen connection, 0 for infinite           | 0               |             
| upstream   | set upstream mode, uses upstream instead of interface on exitNodes file | <empty>         |             
| prettylogs | display colorful logs, if false display them as json                    | false           |             
| loglevel   | Minimum log level to print: trace, debug, info, warn, error, fatal      | info            |             

## ExitNodes file

The list of backends (exitnodes) to use

### File format

```yaml
  # [required if not in upstream mode] interface used as the exit IP, only works when not in upstream
  interface: 0.0.0.0
  
  # [optional] passed as part of authentication to group exit nodes
  region: us
  
  # [optional] passed as part of authentication to explicitly use this exit node
  instance_id: us.00

  # [required if in upstream mode] IP:port of upstream proxy
  upstream: 1.2.3.4:1080
```

#### Basic example:

Server with 4 interfaces, no grouping/targetting specific nodes

```yaml
- interface: 0.0.0.1
- interface: 0.0.0.2
- interface: 0.0.0.3
- interface: 0.0.0.4
```

Service ran as:
```sh
moxxiproxy run 
```

#### Basic upstream example :

Works as load balancer for remote proxies 

```yaml
- upstream: 0.0.0.1:1080
- upstream: 0.0.0.2:1080
- upstream: 0.0.0.3:1080
- upstream: 0.0.0.4:1080
```

Service ran as:
```shell
moxxiproxy run --upstream=true
```

#### region example :

uses regions to group proxies.
Example uses upstream but the same can be achieved with interface.

```yaml
- upstream: 0.0.0.1:1080
  region: us
- upstream: 0.0.0.2:1080
  region: us
- upstream: 0.0.0.3:1080
  region: ca
- upstream: 0.0.0.4:1080
  region: ca
```

Service ran as [set any value as the username]:
```shell
moxxiproxy run --upstream=true --auth=testuser:
```

Consumed as:
```shell
curl -kxhttp://testuser_region-us:@0.0.0.0:1989 http://page.com
```

#### instance example :

Instance is used to select the backend with a "friendly" name.
Example uses upstream but the same can be achieved with interface.

```yaml
- upstream: 0.0.0.1:1080
  instance: us.01
- upstream: 0.0.0.2:1080
  instance: us.02
- upstream: 0.0.0.3:1080
  instance: us.03
- upstream: 0.0.0.4:1080
  instance: us.04
```

Service ran as [set any value as the username]:
```shell
moxxiproxy run --upstream=true --auth=testuser:
```

Consumed as [just make sure the user matches]:
```shell
curl -kxhttp://testuser_instance-us.04:@0.0.0.0:1989 http://page.com
```
#### session example :

Session is an arbitrary value sent that moxxi with use as a key for session stickyness.
Example uses upstream but the same can be achieved with interface.

```yaml
- upstream: 0.0.0.1:1080
- upstream: 0.0.0.2:1080
- upstream: 0.0.0.3:1080
- upstream: 0.0.0.4:1080
```

Service ran as [set any value as the username]:
```shell
moxxiproxy run --upstream=true --auth=testuser:
```

Consumed as:
```shell
curl -kxhttp://testuser_session-1234:@0.0.0.0:1989 http://page.com
```

This will create a session under ID: 1234 and any request with that ID will use the same exit node
