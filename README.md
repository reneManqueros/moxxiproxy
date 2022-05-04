# moxxiproxy

#### The proxy desc

## Features 
- [x] Maps to network interfaces
- [x] Hot swap of backends
- [x] Low resource footprint
- [x] Telnet management console
- [x] Basic Authentication
- [x] Client whitelist 
- [x] Round robin load balancer
- [ ] session stickiness

## Installation
Download:
```shell
git clone https://github.com/reneManqueros/moxxiproxy
```

Compile:
```shell
cd moxxiproxy && go build .
````

## Usage

Execute as proxy server:
```shell
./moxxiproxy run
```

Add a backend:
```shell
./moxxiproxy add 127.0.0.1:8080
```

Add a backend via telnet - Send a "+" and the address:
```shell
telnet 127.0.0.1 33333
+127.0.0.1:8080
```

Remove a backend:
```shell
./moxxiproxy add 127.0.0.1:8080
```

Remove a backend via telnet - Send a "-" and the address:
```shell
telnet 127.0.0.1 33333
-127.0.0.1:8080
```

## Parameters

### When running as a server
Change listen address (default: 1989)
```shell
./moxxiproxy run address=:9090
```
Disable management console
```shell
./moxxiproxy run management=""
```

Backends file location (default: ./backends.yml)
```shell
./moxxiproxy run backends="/etc/moxxiproxy/backends.yml"
```

Enable verbose mode (default: false)
```shell
./moxxiproxy run verbose=true
```

Connection timeout in seconds (default: 0)
```shell
./moxxiproxy run timeout=10
```

Whitelist (default: none), comma separated values, can be only the prefix, if a value is passed, it will block everything not on the list
```shell
./moxxiproxy run whitelist=127.0,10.42.0.1 
```

Set basic auth credentials (default: disabled)
```shell
./moxxiproxy run auth=username:password
```
 
Change management console host/port
```shell
./moxxiproxy run management="127.0.0.1:12345"
```