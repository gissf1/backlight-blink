all: backlight-blink

backlight-blink: backlight-blink.go
	go build -ldflags "-X main.BUILDDATE=`date -u +%Y%m%d%H%M%S`" backlight-blink.go

run: backlight-blink
	#go run backlight-blink.go
	sudo ./backlight-blink

clean:
	rm backlight-blink
