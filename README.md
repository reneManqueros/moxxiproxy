# moxxiproxy

#### The proxy desc

## Features 
- [x] Maps to network interfaces 
- [x] Low resource footprint 
- [x] Basic Authentication
- [x] Client whitelist 
- [x] Random load balance selection
- [ ] Session stickiness
- [ ] Logs
- [ ] Group backends by regions
- 
## Installation
Download binary:
```shell
wget https://github.com/reneManqueros/moxxiproxy/releases/download/v1.2.2/moxxiproxy_1.2.2_Linux_x86_64.tar.gz && tar xf moxxiproxy_1.2.1_Linux_x86_64.tar.gz 
```

Download source and compile:
```shell
git clone https://github.com/reneManqueros/moxxiproxy.git && cd moxxiproxy && go build .
````

## Usage

Execute as proxy server:
```shell
./moxxiproxy run
```
 
## Parameters

### When running as a server
Change listen address (default: 1989)
```shell
./moxxiproxy run --address=:9090
```

ExitNodes file location (default: ./exitnodes.yml)
```shell
./moxxiproxy run --exitnodes=./exitNodes.yml
```

Enable verbose mode (default: false)
```shell
./moxxiproxy run --verbose=true
```

Connection timeout in seconds (default: 0)
```shell
./moxxiproxy run --timeout=10
```

Whitelist (default: empty), comma separated values, can be only the prefix, if a value is passed, it will block everything not on the list
```shell
./moxxiproxy run --whitelist=127.0,10.42.0.1 
```

Set basic auth credentials (default: empty)
```shell
./moxxiproxy run --auth=username:password
```

Enable upstream mode (default: false)
```shell
./moxxiproxy run --upstream=true
```
  