BUILDDIR=$(CURDIR)/output/build

GOBIN=$(CURDIR)/output/bin
GOMOBILE=$(GOBIN)/gomobile
GOBIND=env PATH="$(GOBIN):$(PATH)" "$(GOMOBILE)" bind

IMPORT_HOST=github.com
IMPORT_PATH=$(IMPORT_HOST)/outline_prototype

ROOT_GO_PKG=fullstack_app

.PHONY: android apple all
all: android apple

android: $(BUILDDIR)/android/fullstack_app.aar

$(BUILDDIR)/android/fullstack_app.aar: $(GOMOBILE) fullstack_app/*_templ.go
	mkdir -p "$(BUILDDIR)/android"
	$(GOBIND) -o $(BUILDDIR)/android/fullstack_app.aar -target=android ./fullstack_app

apple: $(BUILDDIR)/apple/fullstack_app.xcframework

$(BUILDDIR)/apple/fullstack_app.xcframework: $(GOMOBILE) fullstack_app/*_templ.go
	mkdir -p "$(BUILDDIR)/apple"
	$(GOBIND) \
		-o $(BUILDDIR)/apple/fullstack_app.xcframework \
		-target=ios,iossimulator,maccatalyst \
		-iosversion=13.1 \
		./fullstack_app

fullstack_app/*_templ.go:
	templ generate

$(GOMOBILE): go.mod
	env GOBIN="$(GOBIN)" go install golang.org/x/mobile/cmd/gomobile
	env GOBIN="$(GOBIN)" $(GOMOBILE) init

go.mod:
	go mod tidy
