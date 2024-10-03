bin/snip: cmd/snip/main.go
	go build -o $@ $<

.PHONY: bin/snip
