// Copyright 2024 Jigsaw Operations LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"image/color"
	"log"
	"net"
	"net/http"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"github.com/Jigsaw-Code/outline-sdk/x/httpproxy"
)

type runningProxy struct {
	server  *http.Server
	Address string
}

func (p *runningProxy) Close() {
	p.server.Close()
}

func runServer(address, transport string) (*runningProxy, error) {
	dialer, err := config.NewStreamDialer(transport)
	if err != nil {
		return nil, fmt.Errorf("could not create dialer: %v", err)
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("could not listen on address %v: %v", address, err)
	}

	server := http.Server{Handler: httpproxy.NewProxyHandler(dialer)}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Serve failed: %v\n", err)
		}
	}()
	return &runningProxy{server: &server, Address: listener.Addr().String()}, nil
}

type appTheme struct {
	fyne.Theme
}

const ColorNameHeaderForeground = "headerForeground"

func (t *appTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameHeaderBackground:
		return color.RGBA{R: 0x00, G: 0x45, B: 0x60, A: 255}
	case ColorNameHeaderForeground:
		return color.White
	default:
		return t.Theme.Color(name, variant)
	}
}

func makeAppHeader(title string) *fyne.Container {
	titleLabel := &widget.RichText{Scroll: container.ScrollNone, Segments: []widget.RichTextSegment{
		&widget.TextSegment{Text: title, Style: widget.RichTextStyle{
			Alignment: fyne.TextAlignCenter,
			ColorName: ColorNameHeaderForeground,
			SizeName:  theme.SizeNameHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		}},
	}}
	return container.NewStack(canvas.NewRectangle(theme.HeaderBackgroundColor()), titleLabel)
}

func main() {
	fyneApp := app.New()
	if meta := fyneApp.Metadata(); meta.Name == "" {
		// App not packaged, probably from `go run`.
		meta.Name = "Local Proxy"
		app.SetMetadata(meta)
	}
	fyneApp.Settings().SetTheme(&appTheme{theme.DefaultTheme()})

	mainWin := fyneApp.NewWindow(fyneApp.Metadata().Name)
	mainWin.Resize(fyne.Size{Width: 350})

	addressEntry := widget.NewEntry()
	addressEntry.SetPlaceHolder("Enter proxy local address")
	addressEntry.Text = "localhost:8080"

	configEntry := widget.NewMultiLineEntry()
	configEntry.Wrapping = fyne.TextWrapBreak
	configEntry.SetPlaceHolder("Enter transport config")

	statusBox := widget.NewLabel("")
	statusBox.Wrapping = fyne.TextWrapWord

	var proxy *runningProxy
	startStopButton := widget.NewButton("", func() {})
	setProxyUI := func(proxy *runningProxy, err error) {
		if proxy != nil {
			statusBox.SetText("Proxy listening on " + proxy.Address)
			addressEntry.Disable()
			configEntry.Disable()
			startStopButton.SetText("Stop")
			startStopButton.SetIcon(theme.MediaStopIcon())
			return
		}
		if err != nil {
			statusBox.SetText("❌ ERROR: " + err.Error())
		} else {
			statusBox.SetText("Proxy not running")
		}
		addressEntry.Enable()
		configEntry.Enable()
		startStopButton.SetText("Start")
		startStopButton.SetIcon(theme.MediaPlayIcon())
	}
	startStopButton.OnTapped = func() {
		log.Println(startStopButton.Text)
		var err error
		if proxy == nil {
			// Start proxy.
			proxy, err = runServer(addressEntry.Text, configEntry.Text)
		} else {
			// Stop proxy
			proxy.Close()
			proxy = nil
		}
		setProxyUI(proxy, err)
	}
	setProxyUI(proxy, nil)

	content := container.NewVBox(
		makeAppHeader(fyneApp.Metadata().Name),
		container.NewPadded(
			container.NewVBox(
				widget.NewLabelWithStyle("Local address", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				addressEntry,
				widget.NewRichTextFromMarkdown("**Transport config** ([format](https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk/x/config#hdr-Config_Format))"),
				configEntry,
				container.NewHBox(layout.NewSpacer(), startStopButton),
				widget.NewLabelWithStyle("Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				statusBox,
			),
		),
	)
	mainWin.SetContent(content)
	mainWin.Show()
	fyneApp.Run()
}
