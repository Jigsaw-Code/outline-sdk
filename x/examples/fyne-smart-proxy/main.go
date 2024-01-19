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
	"image/color"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
)

type appTheme struct {
	fyne.Theme
}

const ColorNameOnPrimary = "OnPrimary"

func (t *appTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameHeaderBackground:
		return t.Color(theme.ColorNamePrimary, variant)
	case theme.ColorNamePrimary:
		if variant == theme.VariantLight {
			return color.RGBA{R: 0x00, G: 0x67, B: 0x7F, A: 255}
		} else {
			return color.RGBA{R: 0x7C, G: 0xD2, B: 0xF0, A: 255}
		}
	case ColorNameOnPrimary:
		if variant == theme.VariantLight {
			return color.White
		} else {
			return color.RGBA{R: 0x00, G: 0x35, B: 0x43, A: 255}
		}
	default:
		return t.Theme.Color(name, variant)
	}
}

func makeAppHeader(title string) *fyne.Container {
	titleLabel := &widget.RichText{Scroll: container.ScrollNone, Segments: []widget.RichTextSegment{
		&widget.TextSegment{Text: title, Style: widget.RichTextStyle{
			Alignment: fyne.TextAlignCenter,
			ColorName: ColorNameOnPrimary,
			SizeName:  theme.SizeNameHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		}},
	}}
	return container.NewStack(canvas.NewRectangle(theme.HeaderBackgroundColor()), titleLabel)
}

type bindingWriter struct {
	binding.String
}

func (w bindingWriter) WriteString(text string) (int, error) {
	value, err := w.String.Get()
	if err != nil {
		return 0, nil
	}
	w.String.Set(value + text)
	return len(text), nil
}

func (w bindingWriter) Clear() {
	w.String.Set("")
}

func main() {
	fyneApp := app.New()
	if meta := fyneApp.Metadata(); meta.Name == "" {
		// App not packaged, probably from `go run`.
		meta.Name = "Smart Proxy"
		app.SetMetadata(meta)
	}
	fyneApp.Settings().SetTheme(&appTheme{theme.DefaultTheme()})

	mainWin := fyneApp.NewWindow(fyneApp.Metadata().Name)
	mainWin.Resize(fyne.Size{Width: 350})

	addressEntry := widget.NewEntry()
	addressEntry.SetPlaceHolder("Enter proxy local address")
	addressEntry.Text = "localhost:8080"

	domainsEntry := widget.NewEntry()
	domainsEntry.SetPlaceHolder("Enter test domains")
	domainsEntry.Text = "example.com"

	configEntry := widget.NewMultiLineEntry()
	configEntry.Wrapping = fyne.TextWrapBreak
	configEntry.SetPlaceHolder("Enter config")

	statusString := binding.NewString()
	statusWriter := bindingWriter{statusString}
	statusBox := widget.NewLabelWithData(statusString)
	statusBox.Wrapping = fyne.TextWrapWord
	statusBox.TextStyle.Monospace = true

	startStopButton := widget.NewButton("", func() {})
	startStopButton.Importance = widget.HighImportance
	setProxyUI := func(proxy *mobileproxy.Proxy, err error) {
		if proxy != nil {
			statusWriter.WriteString("Proxy listening on " + proxy.Address())
			addressEntry.Disable()
			domainsEntry.Disable()
			configEntry.Disable()
			startStopButton.SetText("Stop")
			startStopButton.SetIcon(theme.MediaStopIcon())
			return
		}
		if err != nil {
			statusWriter.WriteString("❌ ERROR: " + err.Error())
		} else {
			statusWriter.WriteString("Proxy not running")
		}
		addressEntry.Enable()
		domainsEntry.Enable()
		configEntry.Enable()
		startStopButton.SetText("Start")
		startStopButton.SetIcon(theme.MediaPlayIcon())
	}
	var proxy *mobileproxy.Proxy
	startStopButton.OnTapped = func() {
		log.Println(startStopButton.Text)
		statusWriter.Clear()
		var err error
		if proxy == nil {
			proxy, err = func() (*mobileproxy.Proxy, error) {
				// Start proxy.
				testDomainsList := &mobileproxy.StringList{}
				for _, domain := range strings.Split(domainsEntry.Text, ",") {
					testDomainsList.Append(strings.TrimSpace(domain))
				}
				dialer, err := mobileproxy.NewStreamDialerFromSearch(testDomainsList, configEntry.Text, statusWriter)
				if err != nil {
					return nil, err
				}
				return mobileproxy.RunProxy(addressEntry.Text, dialer)
			}()
		} else {
			// Stop proxy
			proxy.Stop(1)
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
				widget.NewLabelWithStyle("Test Domains", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				domainsEntry,
				widget.NewRichTextFromMarkdown("**Strategies Config** ([example](https://github.com/Jigsaw-Code/outline-sdk/blob/fortuna-smart/x/examples/smart-proxy/config.json))"),
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
