<div align="center">

<pre>
        @ ~|*
           |  %  @*
           \&amp;_|%%%%
         %%\&amp;@&amp;&amp;&amp;/&amp;@
           @\%\|&amp;&amp;&amp;@
            &amp;@@\@@
             ,  \&amp;
                 ||
                 ||
          \_____~_:__:/
           \_________/
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
> txoko is in early development. There's no release yet, but you can already [run it from source](#try-it) — or star the repo to watch it grow.

*Txoko* means "little corner" in Basque. It's an ambient TUI that lives in a corner of your terminal — a tmux pane, a split, or a splash screen when you open your shell — and slowly comes alive as you code: commits, focus time and everyday activity make your scene grow.

## Planned features

- 🌳 **Living scenes** — a bonsai that really grows branch by branch, a garden, and soon a city lighting up at night, a pond, a forest… pick the one that fits your setup.
- 🪴 **Your own scenes** — design custom scenes with a simple config file, the same way you'd customize neofetch's ASCII art.
- 🖥️ **Three ways to use it** — a live animated view, a one-shot neofetch-style splash, and a single-line status for tmux or your prompt.
- 📊 **Real activity** — git commits, active time and your GitHub contribution calendar.
- 🎨 **Fits your rice** — uses your terminal's own colors, so it matches Catppuccin, Gruvbox, Rosé Pine or whatever you run.
- 🔒 **Local first** — your activity data never leaves your machine.

## Roadmap

- [x] **0 · Foundations** — project layout, tests, linting, CI
- [x] **1 · First scenes** — a growing bonsai and an animated garden
- [x] **2 · Real activity** — git hooks and a local SQLite store
- [ ] **3 · Everywhere** — compact mode, neofetch-style splash and tmux status line
- [ ] **4 · First release** — `brew install` via a Homebrew tap
- [ ] **5 · More scenes** — city, pond and forest, plus custom scenes from config
- [ ] **6 · Your year** — a landscape built from your GitHub contribution calendar
- [ ] **7 · Activity daemon** — active-time tracking as a background service
- [ ] **8 · MCP server** — ask your AI assistant about your progress
- [ ] **9 · Image to ASCII** — turn wallpapers into scene backgrounds

## Try it

You'll need [Go](https://go.dev/dl/) 1.27+.

```sh
git clone https://github.com/LifeOfAitor/txoko.git
cd txoko
make build

./bin/txoko                       # open your scene and shape it with + and -
```

txoko is a visual toy first: by default you decide how grown the scene is. If you'd rather have it **grow with your work**, let it follow your commits:

```sh
./bin/txoko hook install          # every commit from now on is recorded
./bin/txoko import ~/code/*       # optional: bring in your last 30 days of commits
./bin/txoko --work                # the scene now grows with your activity
```

Make a commit in any repository and watch the tree grow a little within a few seconds.

### Focus sessions 🍅

```sh
txoko --pomodoro 50               # a new tree grows from a seed to full bloom in 50 minutes
txoko --pomodoro 25 --break 5     # classic pomodoro with a 5-minute rest
```

The footer counts down, <kbd>space</kbd> pauses (paused time doesn't count), and quitting mid-session asks twice so a stray <kbd>q</kbd> never throws away your focus. When the time is up you get a desktop notification, the tree is in full bloom, and <kbd>enter</kbd> plants another one. Completed sessions are saved too, so they also grow your `--work` scene: 25 minutes of focus count like one commit.

```
╭─ txoko · bonsai · night · paused ──────────────────────────────────────╮
│                                                ·                    ☾  │
│           ·       ·                                                    │
│                        ·          ·                                    │
│                   ·   @& #%%               ·                           │
│                     @ ## *  %      ✦     #@  @                         │
│                     &% @*|&#@           &@   %                         │
│                  %   & & /         %%  *%# |&&%@ @                     │
│               @&@&%@#@& \# @& %%&&@#&@&%&@&%@&                         │
│              &&%&&*&&@%* |@%%@&&_/&&@/%&##&&%%&&                       │
│           @#&@&&#&%@&&\@ \&@%__\_|&%|&%*@\%/&*%&                       │
│         &&&&&**• @_|&@%/& _/@&&&@\~%#&|&@///=%  *%%                    │
│          &@&\*%#@%@\  *%&%  *& *&_#|#|/~_=_#&&%                        │
│        &@%# %\|#_|  & %%\& %  # &%&|@/  #       &                      │
│          %@@&%_____~_\@ &\&  &@&&@ |@&*@                               │
│           @•& %% @     ~______&~_\\/&&  @                              │
│                               %&  ~||&                                 │
│                                 *&#||                                  │
│                                    ||                                  │
│                      \_____,:_._,_,_:~..~,_,__:_/                      │
│                       \________________________/                       │
├────────────────────────────────────────────────────────────────────────┤
│ growth ▰▰▰▰▰▰▰▰▰▰ 100%  manual                        +/- grow  q quit │
╰────────────────────────────────────────────────────────────────────────╯
```

| Key | Action |
|---|---|
| <kbd>+</kbd> / <kbd>-</kbd> | grow or shrink the scene (not with `--work`) |
| <kbd>s</kbd> | switch scene (bonsai, garden) |
| <kbd>e</kbd> | cycle the season: spring → summer → autumn → winter → today's |
| <kbd>n</kbd> | cycle the sky: day → night → follow the clock |
| <kbd>r</kbd> | plant a new layout |
| <kbd>space</kbd> | pause the animation |
| <kbd>q</kbd> | quit |

Flags: `--scene bonsai|garden` picks the scene, `--seed N` picks a layout (by default every machine gets its own tree), `--growth 0..1` sets the starting growth and `--work` grows it from your activity instead. The capture above is `--seed 3` at night.

### Seasons

The tree dresses for the time of year, following the calendar (northern hemisphere) unless you pick one with <kbd>e</kbd> or `--season`:

| Season | The tree | In the air |
|---|---|---|
| 🌸 Spring | cherry blossom pinks, a few whole flowers | petals drifting on the breeze |
| ☀️ Summer | deep layered greens | a leaf now and then, fireflies at night |
| 🍂 Autumn | reds, oranges and golds; some leaves already on the soil | leaves falling |
| ❄️ Winter | bare branches, snowy tufts, snow on the soil | snowfall across the whole sky |

Colors come from your terminal's own palette, so every season matches your theme.

```
╭─ txoko · bonsai · winter · night · paused ───────────────────╮
│                                 ·                         ☾  │
│       *      *               ··                         *  · │
│                  \  .                                        │
│.           ~*. *_____\                                *      │
│              _ _~\.   \ .           .*                       │
│                 * \.. .__\**        /                 .      │
│                    ~\*\.**·\*   \ =/#  */ *                  │
│      .   .        ·  _\= *  \ ** |\    /                     │
│                     ·   _ \_/|/=__ /  |                      │
│                            _\|.   \& \&                      │
│                              \\      ·            .          │
│                               ||               . ·           │
│           ·                *  ||           *                 │
│             *                 ||                             │
│                   \__.__.__.__.__.__.__._/                   │
│                    \____________________/                    │
├──────────────────────────────────────────────────────────────┤
│ growth ▰▰▰▰▰▰▰▰▰▰ 100%  manual              +/- grow  q quit │
╰──────────────────────────────────────────────────────────────╯
```

### Commands

| Command | What it does |
|---|---|
| `txoko` | open your scene and grow it by hand |
| `txoko --work` | open your scene, grown by your real activity |
| `txoko --season winter` | start in a season instead of today's |
| `txoko --pomodoro M [--break M]` | focus session: a new tree blooms as the minutes pass |
| `txoko hook install` | record every commit through a global git hook |
| `txoko hook uninstall` | remove the hook (your history is kept) |
| `txoko hook status` | check whether the hook is installed |
| `txoko import [dir...]` | import your commits from the last 30 days (`--days N`) |
| `txoko stats` | print growth, today's commits and your streak |

### How your activity becomes growth (`--work`)

- **Every commit adds a point**, and so does every 25 minutes of completed pomodoro focus. Points fade with a two-week half-life, so the tree reflects your recent rhythm: keep going and it stays lush, take a break and it slowly thins out — it never dies.
- **Growth eases in:** `growth = 1 − e^(−points / 25)`. Your first commits make a big difference; about three a day keeps the tree near 90%.
- **Streak** counts consecutive days with at least one commit, and stays alive until the day ends.

### Privacy and safety

- Everything stays on your machine, in a single SQLite file: `~/.local/share/txoko/txoko.db` (respects `XDG_DATA_HOME`, or set `TXOKO_DB`).
- Only the repository's folder name, the commit hash and its time are stored — never code, messages or paths.
- The hook installs by pointing git's global `core.hooksPath` at `~/.config/txoko/hooks`. It **refuses** to replace a `core.hooksPath` you already set, and it passes every hook through to each repository's own `.git/hooks`, so existing hooks keep working.
- Commits are recorded in the background: committing never gets slower.
- Repositories that set their own local `core.hooksPath` (Husky, for example) bypass global hooks; use `txoko import` for those.

**How the bonsai grows:** the tree is simulated tip by tip from the seed — the trunk climbs with a slow S-shaped lean, limbs reach sideways, twigs fork and burst into clumps of leaves. Progress reveals that growth in order, so every bit of work adds a branch or a few leaves.

## Development

Requires [Go](https://go.dev/dl/) 1.27+ and [golangci-lint](https://golangci-lint.run/).

```sh
make run     # build and run
make check   # format, lint and test — run before committing
make cover   # open the coverage report
make help    # list every task
```

Scenes are tested with **golden files**: their expected ASCII lives in `testdata/*.golden`. After changing the art on purpose, regenerate them and review the diff:

```sh
go test ./... -update
```

## License

[MIT](LICENSE)
