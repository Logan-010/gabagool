.PHONY: build
build:
	mkdir -p bin
	go build -ldflags="-s -w" -o bin/gabagool .

.PHONY: clean
clean:
	rm -rf bin result

.PHONY: run
run:
	go run .
