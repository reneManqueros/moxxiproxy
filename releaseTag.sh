git tag -a v1.2.3 -m "Fixed issue on HTTP upstreams"
git push origin v1.2.3
curl -sfL https://goreleaser.com/static/run | bash -s -- release
