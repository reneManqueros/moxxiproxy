git tag -a v1.2.3 -m "Updated flag management, readme and release process"
git push origin v1.2.3
curl -sfL https://goreleaser.com/static/run | bash -s -- release
