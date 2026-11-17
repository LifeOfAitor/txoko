# Still screenshot of the autumn tree for the README.
Output docs/images/season-autumn.gif
Source docs/tapes/settings.tape
Set Width 640
Set Height 520
Set WindowBar ""

Hide
Type "export PATH=$PWD/bin:$PATH && clear" Enter
Type "txoko --seed 3 --season autumn --sky day --growth 1" Enter
Show
Sleep 3s
Screenshot docs/images/season-autumn.png
Sleep 1s
Type "q"
