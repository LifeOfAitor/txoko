# Still screenshot of the island scene for the README.
Output docs/images/scene-island.gif
Source docs/tapes/settings.tape
Set Width 640
Set Height 520
Set WindowBar ""

Hide
Type "export PATH=$PWD/bin:$PATH && clear" Enter
Type "txoko --seed 1 --scene island --season summer --sky day --growth 1" Enter
Show
Sleep 3s
Screenshot docs/images/scene-island.png
Sleep 1s
Type "q"
