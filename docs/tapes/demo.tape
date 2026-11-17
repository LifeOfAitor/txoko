# The animated demo at the top of the README.
Output docs/images/demo.gif
Source docs/tapes/settings.tape
Set Width 1000
Set Height 680

Hide
Type "export PATH=$PWD/bin:$PATH PS1='$ ' && clear" Enter
Sleep 300ms
Show

Type "txoko --seed 3 --season spring --sky day --growth 0.05" Sleep 400ms Enter
Sleep 1.5s

# Grow it by hand.
Type@450ms "+++++++++"
Sleep 2s

# Night falls: the tree dims, the moon and the stars come out.
Type "n" Sleep 3.5s

# Through the seasons.
Type "e" Sleep 2.5s
Type "e" Sleep 2.5s
Type "e" Sleep 3s

Type "q"
Sleep 300ms
