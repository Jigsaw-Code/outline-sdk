package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type request struct {
	url       string
	transport string
	status    string
	spinner   spinner.Model
}

func (r *request) IsPending() bool {
	return r.status == ""
}

type model struct {
	urlsInput       textinput.Model
	transportsInput textinput.Model
	requests        []*request
}

func initialModel() model {
	ti1 := textinput.New()
	ti1.Placeholder = "https://example.com, https://example.org"
	ti1.Focus()
	ti1.CharLimit = 0 // no limit
	ti1.Width = 80

	ti2 := textinput.New()
	ti2.Placeholder = "socks5://localhost:1080, direct://"
	ti2.CharLimit = 0 // no limit
	ti2.Width = 80

	return model{
		urlsInput:       ti1,
		transportsInput: ti2,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

type fetchResultMsg struct {
	req    *request
	status string
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case fetchResultMsg:
		msg.req.status = msg.status
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "ctrl+l":
			m.requests = m.requests[:0]
			return m, nil

		case "tab":
			if m.urlsInput.Focused() {
				m.urlsInput.Blur()
				return m, m.transportsInput.Focus()
			} else {
				m.transportsInput.Blur()
				return m, m.urlsInput.Focus()
			}

		case "enter":
			urls := strings.Split(m.urlsInput.Value(), ",")
			transports := strings.Split(m.transportsInput.Value(), ",")

			var fetchCmds []tea.Cmd
			for _, urlStr := range urls {
				u := strings.TrimSpace(urlStr)
				if u == "" {
					continue
				}
				for _, transportConfig := range transports {
					t := strings.TrimSpace(transportConfig)
					s := spinner.New(spinner.WithSpinner(spinner.Dot))
					fetchCmds = append(fetchCmds, s.Tick)
					request := &request{
						url:       u,
						transport: t,
						spinner:   s,
					}
					m.requests = append(m.requests, request)
					fetchCmds = append(fetchCmds, doFetch(request))
				}
			}
			return m, tea.Batch(fetchCmds...)
		}
	}

	var cmd tea.Cmd
	var cmds []tea.Cmd
	// Update spinners.
	for i := range m.requests {
		// Only tick spinners for pending requests
		if m.requests[i].IsPending() {
			m.requests[i].spinner, cmd = m.requests[i].spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}
	// Update text inputs
	if m.urlsInput.Focused() {
		m.urlsInput, cmd = m.urlsInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.transportsInput, cmd = m.transportsInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString("Enter URLs (comma-separated):\n")
	b.WriteString(m.urlsInput.View())
	b.WriteString("\nEnter Transports (comma-separated):\n")
	b.WriteString(m.transportsInput.View())
	b.WriteString("\n\n")

	if len(m.requests) > 0 {
		columns := []table.Column{
			{Title: "URL", Width: 30},
			{Title: "Transport", Width: 20},
			{Title: "Status", Width: 50},
		}
		s := table.DefaultStyles()
		s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true)
		// By default, the first row is highlighted. Since we are not using the table
		// for interaction, we can disable the highlight by overriding the style.
		s.Selected = lipgloss.NewStyle()
		t := table.New(

			table.WithColumns(columns),
			table.WithHeight(10),
			table.WithStyles(s),
		)

		var rows []table.Row
		for i := len(m.requests) - 1; i >= 0; i-- {
			req := m.requests[i]
			transportStr := req.transport
			if transportStr == "" {
				transportStr = "direct"
			}
			status := req.status
			if req.IsPending() {
				status = "fetching " + req.spinner.View()
			}
			rows = append(rows, table.Row{req.url, transportStr, status})
		}
		t.SetRows(rows)
		b.WriteString(t.View())
	}

	b.WriteString("\n\n(press ctrl+c to quit, ctrl+l to clear)\n")

	return b.String()
}
