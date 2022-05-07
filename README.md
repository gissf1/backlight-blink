# backlight-blink
This is a tool to reset quirky backlight hardware by flashing between 3 brightness levels
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
[23:35:01] H/L-VAL=1, TARGET=255, H/L-SLEEPUS=4096, TSLEEPUS=8192, SLEEPSCALE=2048, ZINTERVAL=0, HZ=19.
```
---OR---

```
[22:25:10] H/Lv=255/1, TGTv=255, H/LSus=65536/65536, TSus=65536, S=512, ZI=0, HZ=4.
```

The tool automatically switches to the shorter version when terminal width is limited to try to be informative while not being spammy with newlines.
The fields are as follows:

| Field | Default Value | Description |
|---:|:---:|:---|
| `[11:22:33]` | N/A | Current local time this display was last updated |
| `H/L-VAL` / `H/Lv` | 255/1 | "High" and "Low" brightness values |
| `TARGET` / `TGTv` | 216 | "Target" brightness value.  This is the alternate value that is used.  Also, this is the brightness value assigned on a clean exit.  This cannot be set to Zero. |
| `H/L-SLEEPUS` / `H/LSus` | 65,536 (0.066s) / 65,536 (0.066s) | "High" and "Low" sleep/hold times in Microseconds |
| `TSLEEPUS` / `TSus` | 65,536 (0.066s) | "Target" sleep/hold time in Microseconds |
| `SLEEPSCALE` / `S` | 512 | Amount by which to increase or decrease the sleep times |
| `ZINTERVAL` / `ZI` | 0 | Interval in which to set the brightness to Zero.  This is in units of "1/3 cycles".  Therefore a value of 6 here means 2 compete cycles (that is, it can try to Zero at a frequency of up to 3x the current Hz rate).  Generally you want this to be somewhat large, as most displays go into DPMS power save with brightness 0.  The choice of "1/3 cycles" allows the user to alternate zeroing in different patterns to try to help reset the display hardware.  A value of 0 disables this feature. |
| `HZ` | N/A | Current measured complete cycle time in Hz based on the SLEEP values above and current system load |

## Keys:

| Key | Description |
|:---:|:---|
| q | Quit the tool cleanly |
| A | Increase `HIGHVAL` by 1 |
| Z | Decrease `HIGHVAL` by 1 |
| a | Increase `LOWVAL` by 1 |
| z | Decrease `LOWVAL` by 1 |
| s | Increase `TARGET` by 1 |
| x | Decrease `TARGET` by 1 |
| E | Double `HSLEEPUS` |
| D | Increase `HSLEEPUS` by `SLEEPSCALE` |
| C | Decrease `HSLEEPUS` by `SLEEPSCALE`, but it cannot go lower than 0 |
| e | Double `LSLEEPUS` |
| d | Increase `LSLEEPUS` by `SLEEPSCALE` |
| c | Decrease `LSLEEPUS` by `SLEEPSCALE`, but it cannot go lower than 0 |
| r | Double `TSLEEPUS` |
| f | Increase `TSLEEPUS` by `SLEEPSCALE` |
| v | Decrease `TSLEEPUS` by `SLEEPSCALE`, but it cannot go lower than 1 |
| g | Double `SLEEPSCALE` |
| b | Half `SLEEPSCALE` |
| h | Increase `ZINTERVAL` to double + 1 |
| n | Half `ZINTERVAL` |

## History:
Originally I made the `backlight-blink.sh` shell script, but once I realized I needed tighter timings than could be accomplished in BASH, I decided to port the code to Go instead.
## Disclaimer:
This is a tool for expert users with expert hardware understanding.  If you did not build and design the display hardware yourself, or you are not fully aware of what you are doing, this may damage your hardware.  Any use of this tool is at your own risk.  There are no warranties of any kind.  If you do not take complete responsibility yourself, you will have bad things happen.  Don't complain to me if this tool damages your video card, monitor, LCD, laptop, video cables, television, receiver, or anything else.
