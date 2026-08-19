Output docs/demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 960
Set Height 640
Set Theme "Catppuccin Mocha"
Set Padding 20
Set Framerate 30

Hide
Type "cd /Users/samybaouche/tfguard && make build -q"
Enter
Sleep 2s
Show

Type "./bin/tfguard scan --plan testdata/plan_mixed.json --no-ai --no-banner"
Enter
Sleep 4s
