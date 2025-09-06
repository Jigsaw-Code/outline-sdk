# Fetch UI

A terminal user interface (TUI) for fetching web pages with different transport configurations.

## Description

This tool provides a simple UI to test different transport configurations against a list of URLs. It's built with Go and the Bubble Tea library.

## Usage

To run the application, navigate to this directory and use `go run`:

```bash
cd x/tools/fetchui
go run .
```

### Interface

-   **URL Input**: Enter one or more URLs, separated by commas.
-   **Transport Input**: Enter one or more transport configurations (e.g., `socks5://localhost:1080`, `direct://`), separated by commas.
-   **Submit**: Press `Enter` to start fetching.
-   **Quit**: Press `Ctrl+C` or `Esc` to exit.

The application will attempt to fetch each URL with each specified transport in parallel. The status of each fetch operation will be displayed in real-time.

## Dependencies

-   [Bubble Tea](https://github.com/charmbracelet/bubbletea)
-   [Bubbles](https://github.com/charmbracelet/bubbles)

Dependencies are managed using Go modules and are listed in the `x/go.mod` file.
