.PHONY: css build run clean docker

css:
	npx @tailwindcss/cli -i web/static/css/input.css -o web/static/css/output.css

css-watch:
	npx @tailwindcss/cli -i web/static/css/input.css -o web/static/css/output.css --watch

build: css
	go build -ldflags="-s -w" -o bin/deployhub .

run: build
	./bin/deployhub

docker:
	docker build -t deployhub .

clean:
	rm -rf bin/ data/*.db data/*.db-*
