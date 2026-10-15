# Quay lại demo: make build && vhs docs/assets/demo.tape (cần vhs, ttyd, Chrome, ffmpeg).
Output docs/assets/demo.gif
Output docs/assets/demo.mp4
Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 720
Set Padding 24
Set Framerate 24
Set TypingSpeed 90ms
Set Theme "Catppuccin Mocha"

Hide
Type "export XDG_CONFIG_HOME=$(mktemp -d); rm -rf /tmp/saveme-demo-$(id -u); SAVEME_DEMO_CODE=K7MQ ./bin/saved -demo >/dev/null 2>&1 & DEMO_PID=$!; sleep 1; clear"
Enter
Sleep 1.5s
Type "./bin/saveme -demo"
Enter
Sleep 1.5s
Show

Sleep 2s
Type "a"
Sleep 400ms
Type "https://www.YouTube.com/watch"
Sleep 1.2s
Enter
Sleep 800ms
Type "a"
Type "reddit.com"
Enter
Sleep 800ms
Type "a"
Type "news.ycombinator.com"
Enter
Sleep 1.5s

Type "?"
Sleep 2.5s
Type "q"
Sleep 600ms
Type "/"
Type "red"
Sleep 1.5s
Escape
Sleep 800ms

Space
Down
Space
Sleep 1.2s
Enter
Sleep 1.2s
Type "2"
Sleep 1.2s
Type "5"
Sleep 2s
Type@400ms "K7MQ"
Sleep 800ms
Enter
Sleep 2.5s

Type "gg"
Enter
Sleep 4.5s
Type "q"
Sleep 800ms

Type "s"
Sleep 1.2s
Enter
Sleep 800ms
Down
Sleep 900ms
Down
Sleep 900ms
Up
Sleep 900ms
Enter
Sleep 800ms
Down
Enter
Sleep 800ms
Down 7
Sleep 1.5s
Enter
Sleep 800ms
Down
Enter
Sleep 800ms
Down 6
Space
Sleep 800ms
Down 6
Space
Sleep 1.5s
Enter
Sleep 1s
Type "q"
Sleep 600ms
Type "gg"
Enter
Sleep 5s
Type "q"
Sleep 800ms

Type "G"
Enter
Sleep 800ms
Type "1"
Sleep 1.5s
Type "2"
Sleep 2.5s
Enter
Sleep 2s
Type@400ms "K7MQ"
Sleep 800ms
Enter
Sleep 3s

Hide
Type "q"
Sleep 500ms
Type "kill $DEMO_PID; rm -rf /tmp/saveme-demo-$(id -u) $XDG_CONFIG_HOME"
Enter
