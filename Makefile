.PHONY: all build dev clean build-gui build-gui-windows generate-icon

all: build

dev-server:
	cd server && go run .

dev-admin:
	cd admin && npm run dev

dev-site:
	cd site && npm run dev

dev: dev-server

build-admin:
	cd admin && npm install && npm run build
	rm -rf ../server/static/embed/admin
	mv dist ../server/static/embed/admin

build-site:
	cd site && npm install && npm run build
	rm -rf ../server/static/embed/site
	mv dist ../server/static/embed/site

build-frontend: build-admin build-site

build-server:
	cd server && go build -tags sqlite_fts5 -ldflags="-s -w" -o imgbed .

build: build-frontend build-server

generate-icon:
	cd server/tools/png2ico && go run . ../../systray/icon.png ../../systray/icon.ico
	cd server && rsrc -ico systray/icon.ico -o windows/rsrc.syso

build-gui-windows: build-frontend generate-icon
	cd server && go build -tags "gui sqlite_fts5" -ldflags "-H=windowsgui -s -w" -o imgbed.exe .

# macOS 构建：使用 scripts/build-macos.sh（需要 M1/M2 Mac 本地编译）

clean:
	rm -rf server/static/embed/admin server/static/embed/site
	rm -f server/imgbed server/imgbed.exe server/imgbed-gui server/imgbed-gui.exe
	rm -f server/windows/rsrc.syso server/systray/icon.ico
