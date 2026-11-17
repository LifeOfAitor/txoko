# txoko in a corner of tmux, with `txoko status` in the status bar. It runs
# on its own tmux server (-L txoko-readme) with demo data, never the user's.
Output docs/images/tmux.gif
Source docs/tapes/settings.tape
Set Width 1100
Set Height 600

Hide
Type "export PATH=$PWD/bin:$PATH TXOKO_DB=/tmp/txoko-readme/txoko.db PS1='$ '" Enter
Type "mkdir -p /tmp/txoko-readme && docs/tapes/demo-data.sh && clear" Enter
Sleep 5s
# -u: the recording terminal doesn't announce UTF-8, so tell tmux.
Type "tmux -u -L txoko-readme -f docs/tapes/tmux.conf new-session" Enter
Sleep 1.5s
Type "tmux split-window -h -l 32 'txoko --work --seed 1234 --sky day' && tmux select-pane -L" Enter
Sleep 1s
Type "clear && git --no-pager log -20 --color --format='%C(yellow)%h%C(reset) %<(62,trunc)%s'" Enter
Sleep 5s
Show
Sleep 500ms
Screenshot docs/images/tmux.png
Sleep 1s

Hide
Type "tmux -L txoko-readme kill-server" Enter
Sleep 500ms
