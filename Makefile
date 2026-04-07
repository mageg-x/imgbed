.PHONY: all build dev clean

all: build

dev-server:
	cd server && go run ./cmd/server

dev-admin:
	cd admin && npm run dev

dev-site:
	cd site && npm run dev

dev: dev-server

build-admin:
	cd admin && npm install && npm run build

build-site:
	cd site && npm install && npm run build

build-frontend: build-admin build-site

build-server:
	cd server && go build -o imgbed ./cmd/server

build: build-frontend build-server

clean:
	rm -rf server/embed/admin server/embed/site
	rm -f server/imgbed server/imgbed.exe
