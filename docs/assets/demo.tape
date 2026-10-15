# Quay lại demo: make build && vhs docs/assets/demo.tape (cần vhs, ttyd, Chrome, ffmpeg).
Output docs/assets/demo.gif
Output docs/assets/demo.mp4
Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 720
Set Padding 24
Set Framerate 24
Set TypingSpeed 110ms
Set Theme "Catppuccin Mocha"

Hide
Type "export XDG_CONFIG_HOME=$(mktemp -d); rm -rf /tmp/saveme-demo-$(id -u); SAVEME_DEMO_CODE=K7MQ ./bin/saved -demo >/dev/null 2>&1 & DEMO_PID=$!; sleep 1; clear"
Enter
Sleep 1.5s
Type "SAVEME_DEMO_KEYS=1 ./bin/saveme -demo"
Enter
Sleep 1.5s
Show

Sleep 2s
Type "a"
Sleep 1.5s
Type "https://www.YouTube.com/watch"
Sleep 1.5s
Enter
Sleep 1.5s
Type "a"
Type "reddit.com"
Enter
Sleep 1.5s
Type "a"
Type "news.ycombinator.com"
Enter
Sleep 1.5s

Type "?"
Sleep 2.5s
Type "q"
Sleep 1.5s
Type "/"
Type "red"
Sleep 1.5s
Escape
Sleep 1.5s

Space
Down
Space
Sleep 1.5s
Enter
Sleep 1.5s
Type "2"
Sleep 1.5s
Type "5"
Sleep 2s
Type@400ms "K7MQ"
Sleep 1.5s
Enter
Sleep 2.5s

Type "gg"
Enter
Sleep 4.5s
Type "q"
Sleep 1.5s

Type "s"
Sleep 1.5s
Enter
Sleep 1.5s
Down
Sleep 1.5s
Down
Sleep 1.5s
Up
Sleep 1.5s
Enter
Sleep 1.5s
Down
Enter
Sleep 1.5s
Down 7
Sleep 1.5s
Enter
Sleep 1.5s
Down
Enter
Sleep 1.5s
Down 6
Space
Sleep 1.5s
Down 6
Space
Sleep 1.5s
Enter
Sleep 1.5s
Type "q"
Sleep 1.5s
Type "gg"
Enter
Sleep 5s
Type "q"
Sleep 1.5s

Type "G"
Enter
Sleep 1.5s
Type "1"
Sleep 1.5s
Type "2"
Sleep 2.5s
Enter
Sleep 2s
Type@400ms "K7MQ"
Sleep 1.5s
Enter
Sleep 3s

Hide
Type "q"
Sleep 1.5s
Type "kill $DEMO_PID; rm -rf /tmp/saveme-demo-$(id -u) $XDG_CONFIG_HOME"
Enter
