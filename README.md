# backlight-blink
This is a tool to reset quirky backlight hardware by flashing between 2 brightness levels
## Building/Running:

```
$ make
$ ./backlight-blink
```
---OR---

```
$ go build -o backlight-blink
$ ./backlight-blink
```
---OR---

```
$ go run backlight-blink.go
```

## Command line Usage:
This tool takes no arguments and assumes that it is running on a terminal.  Piping in/out of the process is undefined.
## Status Line:
The tool displays a status line while executing to keep you informed of the current state.  The status line looks like either of:

```
[23:35:01] HIGHVAL=1, TARGET=255, HSLEEPUS=4096, TSLEEPUS=8192, ZINTERVAL=0, SLEEPSCALE=2048, HZ=19.
```
---OR---

```
[22:25:10] HIv=1, TGTv=255, HSus=65536, TSus=65536, ZI=0, S=512, HZ=4.
```

The tool automatically switches to the shorter version when terminal width is limited to try to be informative while not being spammy with newlines.
The fields are as follows:

| Field | Default Value | Description |
|---:|:---:|:---|
| `[11:22:33]` | N/A | Current local time this display was last updated |
| `HIGHVAL` / `HIv` | 255 | "High" brightness value (or low, as this is the only brightness value that can be set to zero) |
| `TARGET` / `TGTv` | 216 | "Target" brightness value.  This is the alternate value that is used.  Also, this is the brightness value assigned on a clean exit. |
| `HSLEEPUS` / `HSus` | 65,536 (0.066s) | "High" sleep/hold time in Microseconds |
| `TSLEEPUS` / `TSus` | 65,536 (0.066s) | "Target" sleep/hold time in Microseconds |
| `ZINTERVAL` / `ZI` | 0 | Interval in which to set the brightness to Zero.  This is in units of "half cycles".  Therefore a value of 4 here means 2 compete cycles (that is, roughly double the current Hz value).  Generally you want this to be somewhat large, as most displays go into DPMS power save with brightness 0.  The choice of "half cycles" allows the user to alternate zeroing in different patterns to try to help reset the display hardware.  A value of 0 disables this feature. |
| `SLEEPSCALE` / `S` | 512 | Amount by which to increase or decrease the sleep times |
| `HZ` | N/A | Current measured complete cycle time in Hz based on the SLEEP values above and current system load |

## Keys:

| Key | Description |
|:---:|:---|
| q | Quit the tool cleanly |
| a | Increase `HIGHVAL` by 1 |
| z | Decrease `HIGHVAL` by 1 |
| s | Increase `TARGET` by 1 |
| x | Decrease `TARGET` by 1 |
| e | Double `HSLEEPUS` |
| d | Increase `HSLEEPUS` by `SLEEPSCALE` |
| c | Decrease `HSLEEPUS` by `SLEEPSCALE`, but it cannot go lower than 0 |
| r | Double `TSLEEPUS` |
| f | Increase `TSLEEPUS` by `SLEEPSCALE` |
| v | Decrease `TSLEEPUS` by `SLEEPSCALE`, but it cannot go lower than 1 |
| g | Increase `ZINTERVAL` to double + 1 |
| b | Half `ZINTERVAL` |
| h | Double `SLEEPSCALE` |
| n | Half `SLEEPSCALE` |

## History:
Originally I made the `backlight-blink.sh` shell script, but once I realized I needed tighter timings than could be accomplished in BASH, I decided to port the code to Go instead.
## Disclaimer:
This is a tool for expert users with expert hardware understanding.  If you did not build and design the display hardware yourself, or you are not fully aware of what you are doing, this may damage your hardware.  Any use of this tool is at your own risk.  There are no warranties of any kind.  If you do not take complete responsibility yourself, you will have bad things happen.  Don't complain to me if this tool damages your video card, monitor, LCD, laptop, video cables, television, receiver, or anything else.
