# Still screenshot of the summer tree for the README.
Output docs/images/season-summer.gif
Source docs/tapes/settings.tape
Set Width 640
Set Height 520
Set WindowBar ""

Hide
Type "export PATH=$PWD/bin:$PATH && clear" Enter
Type "txoko --seed 3 --season summer --sky night --growth 1" Enter
Show
Sleep 3s
Screenshot docs/images/season-summer.png
Sleep 1s
Type "q"
