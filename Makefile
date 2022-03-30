all: backlight-blink

backlight-blink: backlight-blink.go
	go build backlight-blink.go

run: backlight-blink
	#go run backlight-blink.go
	sudo ./backlight-blink

clean:
	rm backlight-blink
