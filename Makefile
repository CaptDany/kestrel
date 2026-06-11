.PHONY: build run test clean

build:
	go build -o kestrel -ldflags="-s -w" .

run:
	go run .

test:
	go test ./...

clean:
	rm -f kestrel kestrel.exe
	rm -f cmd/scraper/scraper cmd/scraper/scraper.exe
	rm -rf data/

docker-build:
	docker build -t kestrel .

docker-run:
	docker compose up -d

docker-stop:
	docker compose down
