# Still screenshot of the winter tree for the README.
Output docs/images/season-winter.gif
Source docs/tapes/settings.tape
Set Width 640
Set Height 520
Set WindowBar ""

Hide
Type "export PATH=$PWD/bin:$PATH && clear" Enter
Type "txoko --seed 3 --season winter --sky night --growth 1" Enter
Show
Sleep 3s
Screenshot docs/images/season-winter.png
Sleep 1s
Type "q"
