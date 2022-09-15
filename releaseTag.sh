git tag -a v1.2.2 -m "Updated flag and readme"
git push origin v1.2.2
curl -sfL https://goreleaser.com/static/run | bash -s -- release
