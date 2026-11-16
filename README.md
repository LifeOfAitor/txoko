<div align="center">

<pre>
     ·   ☾      ·
        ·  ✿        ·
   ❀   \|/   ,@,   ✿
  \|/  -*-  (@@@) \|/
  _|_  /|\   \|/  _|_
 ▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔
</pre>

# txoko

**A cozy corner for your terminal that grows as you work.**

[![CI](https://github.com/LifeOfAitor/txoko/actions/workflows/ci.yml/badge.svg)](https://github.com/LifeOfAitor/txoko/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/LifeOfAitor/txoko?logo=go&logoColor=white)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-a6e3a1)](LICENSE)
[![Status](https://img.shields.io/badge/status-sprouting%20🌱-f9e2af)](#roadmap)

</div>

---

> [!NOTE]
> txoko is in early development. Nothing to install yet — follow along or star the repo to watch it grow.

*Txoko* means "little corner" in Basque. It's an ambient TUI that lives in a corner of your terminal — a tmux pane, a split, or a splash screen when you open your shell — and slowly comes alive as you code: commits, focus time and everyday activity make your scene grow.

## Planned features

- 🌱 **Living scenes** — a garden, a city lighting up at night, a pond, a forest… pick the one that fits your setup.
- 🪴 **Your own scenes** — design custom scenes with a simple config file, the same way you'd customize neofetch's ASCII art.
- 🖥️ **Three ways to use it** — a live animated view, a one-shot neofetch-style splash, and a single-line status for tmux or your prompt.
- 📊 **Real activity** — git commits, active time and your GitHub contribution calendar.
- 🎨 **Fits your rice** — uses your terminal's own colors, so it matches Catppuccin, Gruvbox, Rosé Pine or whatever you run.
- 🔒 **Local first** — your activity data never leaves your machine.

## Roadmap

- [x] **0 · Foundations** — project layout, tests, linting, CI
- [ ] **1 · First scene** — an animated garden in the terminal
- [ ] **2 · Real activity** — git hooks and a local SQLite store
- [ ] **3 · Everywhere** — compact mode, neofetch-style splash and tmux status line
- [ ] **4 · First release** — `brew install` via a Homebrew tap
- [ ] **5 · More scenes** — city, pond and forest, plus custom scenes from config
- [ ] **6 · Your year** — a landscape built from your GitHub contribution calendar
- [ ] **7 · Activity daemon** — active-time tracking as a background service
- [ ] **8 · MCP server** — ask your AI assistant about your progress
- [ ] **9 · Image to ASCII** — turn wallpapers into scene backgrounds

## Development

Requires [Go](https://go.dev/dl/) 1.27+ and [golangci-lint](https://golangci-lint.run/).

```sh
git clone https://github.com/LifeOfAitor/txoko.git
cd txoko
make run     # build and run
make check   # format, lint and test — run before committing
make help    # list every task
```

## License

[MIT](LICENSE)
