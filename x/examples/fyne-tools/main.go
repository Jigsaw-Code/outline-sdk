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
	"context"
	"image/color"
	"log"
	"net"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

func main() {
	fyneApp := app.New()
	if meta := fyneApp.Metadata(); meta.Name == "" {
		// App not packaged, probably from `go run`.
		meta.Name = "Net Tools"
		app.SetMetadata(meta)
	}
	fyneApp.Settings().SetTheme(&appTheme{theme.DefaultTheme()})

	mainWin := fyneApp.NewWindow(fyneApp.Metadata().Name)
	mainWin.Resize(fyne.Size{Width: 350})

	domainEntry := widget.NewEntry()
	domainEntry.SetPlaceHolder("Enter domain name to lookup")
	domainEntry.Text = "www.example.com."

	aBox := widget.NewLabel("")
	aBox.Wrapping = fyne.TextWrapWord
	aBox.TextStyle.Monospace = true
	aaaaBox := widget.NewLabel("")
	aaaaBox.Wrapping = fyne.TextWrapWord
	aaaaBox.TextStyle.Monospace = true
	cnameBox := widget.NewLabel("")
	cnameBox.Wrapping = fyne.TextWrapWord

	lookupButton := widget.NewButton("Lookup", func() {})
	lookupButton.Importance = widget.HighImportance
	lookupButton.OnTapped = func() {
		domain := domainEntry.Text
		log.Println("Lookup", domain)
		var resolver net.Resolver

		ips, err := resolver.LookupIP(context.Background(), "ip4", domain)
		if err != nil {
			aBox.SetText("❌ " + err.Error())
		} else {
			texts := make([]string, len(ips))
			for ii, ip := range ips {
				texts[ii] = ip.String()
			}
			aBox.SetText(strings.Join(texts, ", "))
		}

		ips, err = resolver.LookupIP(context.Background(), "ip6", domain)
		if err != nil {
			aaaaBox.SetText("❌ " + err.Error())
		} else {
			texts := make([]string, len(ips))
			for ii, ip := range ips {
				texts[ii] = ip.String()
			}
			aaaaBox.SetText(strings.Join(texts, ", "))
		}
		cname, err := lookupCNAME(context.Background(), domain)
		// This doesn't work on mobile:
		// cname, err := resolver.LookupCNAME(context.Background(), domain)
		if err != nil {
			cnameBox.SetText("❌ " + err.Error())
		} else {
			cnameBox.SetText(cname)
		}
	}

	content := container.NewVBox(
		makeAppHeader(fyneApp.Metadata().Name),
		container.NewPadded(
			container.NewVBox(
				widget.NewLabelWithStyle("Domain", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				domainEntry,
				container.NewHBox(layout.NewSpacer(), lookupButton),
				// widget.NewLabelWithStyle("Results", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				&widget.Separator{},
				widget.NewLabelWithStyle("A", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				aBox,
				widget.NewLabelWithStyle("AAAA", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				aaaaBox,
				widget.NewLabelWithStyle("CNAME", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				cnameBox,
			),
		),
	)
	mainWin.SetContent(content)
	mainWin.Show()
	fyneApp.Run()
}
