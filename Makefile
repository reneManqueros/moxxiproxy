build:
	go build -ldflags "-w -s" .

release:
	git tag -a $(tag) -m "$(tag)" && git push origin $(tag)  && goreleaser release --clean

tag:
	tag=1.4.5 make release