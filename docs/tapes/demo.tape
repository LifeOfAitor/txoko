# The animated demo at the top of the README. It starts with txoko already
# running: the command is typed while the recording is hidden.
Output docs/images/demo.gif
Source docs/tapes/settings.tape
Set Width 1000
Set Height 680

Hide
Type "export PATH=$PWD/bin:$PATH && clear" Enter
Type "txoko --seed 1234 --season spring --sky day --growth 0.05" Enter
Sleep 1.2s
Show

Sleep 1s

# Grow it by hand.
Type@450ms "+++++++++"
Sleep 2s

# Night falls: the tree dims, the moon and the stars come out.
Type "n" Sleep 3.5s

# Through the seasons.
Type "e" Sleep 2.5s
Type "e" Sleep 2.5s
Type "e" Sleep 3s

Hide
Type "q"
