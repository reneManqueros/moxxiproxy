buildprod:
	go build -ldflags "-w -s" .

builddev:
	go build .