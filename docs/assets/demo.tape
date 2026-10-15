# Quay lại demo.gif/demo.mp4 bằng VHS: vhs docs/assets/demo.tape (cần vhs, ttyd, ffmpeg).
Output docs/assets/demo.gif
Output docs/assets/demo.mp4
Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Hide
Type "rm -rf /tmp/saveme-demo-$(id -u); (./bin/saved -demo >/dev/null 2>&1 &) ; sleep 1; clear"
Enter
Show
Type "./bin/saveme -demo"
Enter
Sleep 1.5s
Type "a"
Type "https://www.YouTube.com/watch"
Sleep 800ms
Enter
Type "a"
Type "reddit.com"
Enter
Sleep 800ms
Type "?"
Sleep 2s
Type "q"
Type " "
Type "j"
Type " "
Sleep 800ms
Enter
Type "2"
Sleep 800ms
Type "5"
Sleep 2.5s
Escape
Sleep 800ms
Type "s"
Sleep 1.5s
Escape
Escape
Sleep 1s
Type "q"
Hide
Type "pkill -f 'saved -demo'"
Enter
