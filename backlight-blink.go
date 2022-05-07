package main
// reset quirky backlight hardware by flashing between 2 brightness levels

import (
	"fmt"
	"os"
	"io/fs"
	"strconv"
	"path"
	"time"
	"strings"
	toml "github.com/pelletier/go-toml"
	"github.com/mattn/go-tty"
)

const PathSeparator = string(os.PathSeparator)

type ConfigFileType struct {
	VERSION    string
	HIGHVAL    int
	LOWVAL     int
	TARGET     int
	HSLEEPUS   int
	LSLEEPUS   int
	TSLEEPUS   int
	SLEEPSCALE int
	ZINTERVAL  int
}

func DefaultConfig() (ConfigFileType) {
	return ConfigFileType{
		"",
		255,
		1,
		216,
		65536,
		65536,
		65536,
		512,
		0,
	}
}

var (
	BUILDDATE string
	configVersionLevel int = 0
	ENV = os.Environ()
	UID = os.Getuid()
	EUID = os.Geteuid()
	// default values
	RUN=1
	cfg ConfigFileType
	ZCOUNTER=0
	HZ=0
	LASTHZSEC int64 = 0
	// initial value as a base
	MAX = -1
	DIR = "/sys/class/backlight/"
	TMPFN string
	stdin *tty.TTY
	KEYBUFFER = make(chan string, 256)
	COLUMNS int
	BrightnessFile string
	openIntFiles map[string] *os.File = make(map[string] *os.File)
)

func ReadDirUnsorted(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	return list, nil
}

func ReadSubdirs(basepath string) ([]os.FileInfo, error) {
	var subdirs []os.FileInfo
	contents, err := ReadDirUnsorted(DIR)
	if err != nil {
		return nil, err
	}
	subdirs = make([]os.FileInfo, 0)
	for _, entry := range contents {
		//fmt.Printf("Name=%s permissions=%#o: ", entry.Name(), entry.Mode().Perm())
		switch mode := entry.Mode(); {
			case mode.IsRegular():
				//fmt.Println("regular file")
			case mode.IsDir():
				//fmt.Println("directory")
				subdirs = append(subdirs, entry)
			case mode & fs.ModeSymlink != 0:
				//fmt.Println("symbolic link")
				subdirs = append(subdirs, entry)
			default:
				//fmt.Println("(ignored type)")
		}
	}
	return subdirs, nil
}

func PathExists(path string) (bool, os.FileInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, nil, err
	}
	return true, fi, nil
}

func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ReadFileInt(path string) (int, error) {
	ba, err := os.ReadFile(path)
	if err != nil {
		return -1, err
	}
	s := string(ba)
	s = strings.TrimRight(s, "\t\r\n ")
	i, err := strconv.Atoi(s)
	if err != nil {
		return -1, err
	}
	return i, nil
}

func CreateOrReplaceFileContent(path string, data string) (error) {
	err := os.WriteFile(path, []byte(data), 0644)
	return err
}

func ReplaceFileContent(path string, data string) (error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(data))
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

func OpenIntFile(path string) (*os.File, error) {
	f, ok := openIntFiles[path]
	if ok {
		return f, nil
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return nil, err
	}
	openIntFiles[path] = f
	return f, nil
}

func ReplaceFileIntContent(path string, i int) (error) {
	f, err := OpenIntFile(path)
	if err != nil {
		return err
	}
	s := strconv.Itoa(i)
	_, err = f.WriteAt([]byte(s), 0)
	if err1 := f.Sync(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

func FindAndVerifyBacklightInterface() {
	subdirs, err := ReadSubdirs(DIR)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: error reading backlight interface directory: %s: %v\n", DIR, err)
		os.Exit(1)
	}
	if len(subdirs) == 0 {
		fmt.Fprintf(os.Stderr, "FATAL: cannot find any backlight devices in %s\n", DIR)
		os.Exit(1)
	}
	if len(subdirs) > 1 {
		fmt.Fprintf(os.Stderr, "WARNING: found multiple backlight devices:\n")
		for idx, device := range subdirs {
			fmt.Fprintf(os.Stderr, "    %d: %s\n", (idx+1), device.Name())
		}
		fmt.Fprintf(os.Stderr, "defaulting to device #%d: %s\n", 1, subdirs[0].Name())
		subdirs = subdirs[0:0]
	}
	fmt.Printf("Found backlight device: %s\n", subdirs[0].Name())
	DIR = DIR + PathSeparator + subdirs[0].Name()
	// now that we have our device directory, test the contents
	for _, target := range []string{ "max_brightness", "brightness", "actual_brightness" } {
		targetPath := DIR + PathSeparator + target
		exists, _, err := PathExists(targetPath)
		if !exists {
			fmt.Fprintf(os.Stderr, "FATAL: cannot find file: %s: %v\n", targetPath, err)
			os.Exit(1)
		}
	}
	// get MAX brightness
	targetPath := DIR + PathSeparator + "max_brightness"
	MAXstr, err := ReadFile(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: cannot read file: %s\n", targetPath)
		os.Exit(1)
	}
	MAXstr = strings.TrimRight(MAXstr, "\t\r\n ")
	MAX, err = strconv.Atoi(MAXstr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: cannot parse file: %s: %v\n", targetPath, err)
		fmt.Fprintf(os.Stderr, "FATAL: data: %q\n", MAXstr)
		os.Exit(1)
	}
	// get TARGET brightness
	targetPath = DIR + PathSeparator + "brightness"
	brightness, err := ReadFile(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: cannot read file: %s\n", targetPath)
		os.Exit(1)
	}
	err = ReplaceFileContent(targetPath, brightness)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: cannot write file: %s\n", targetPath)
		os.Exit(1)
	}
	// get TARGET actual_brightness
	targetPath = DIR + PathSeparator + "actual_brightness"
	TARGETstr, err := ReadFile(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: cannot read file: %s\n", targetPath)
	} else {
		TARGETstr = strings.TrimRight(TARGETstr, "\t\r\n ")
		cfg.TARGET, err = strconv.Atoi(TARGETstr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: cannot parse file: %s: %v\n", targetPath, err)
			fmt.Fprintf(os.Stderr, "FATAL: data: %q\n", TARGETstr)
			os.Exit(1)
		}
	}
	// store in variable
	BrightnessFile = DIR + PathSeparator + "brightness"
}

func setBrightness(brightness int) {
	err := ReplaceFileIntContent(BrightnessFile, brightness)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to write file: %s: %v\n", BrightnessFile, err)
	}
}

func GetConfigFilename() (string) {
	if configVersionLevel == 0 {
		return "/tmp/" + path.Base(os.Args[0]) + ".tmp"
	}
	keepDate := configVersionLevel << 2
	return "/tmp/" + path.Base(os.Args[0]) + "." + (BUILDDATE[:keepDate]) + ".tmp"
}

func LoadConfig() {
	cfg = DefaultConfig()
	// read last config
	TMPFN = GetConfigFilename()
	configData, err := os.ReadFile(TMPFN)
	if err == nil {
		err = toml.Unmarshal(configData, &cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to parse config: %v\n", err)
			os.Exit(1)
		}
		// empty version means it was unspecified
		if cfg.VERSION == "" {
			fmt.Fprintf(os.Stderr, "rejecting unversioned cfg: %v\n", cfg)
			cfg = DefaultConfig()
			configVersionLevel++
			LoadConfig()
		} else if cfg.VERSION < "202205" {
			fmt.Fprintf(os.Stderr, "rejecting old cfg: %v\n", cfg)
			cfg = DefaultConfig()
			configVersionLevel++
			LoadConfig()
		} else if cfg.VERSION > BUILDDATE {
			fmt.Fprintf(os.Stderr, "rejecting newer cfg: %v\n", cfg)
			cfg = DefaultConfig()
			configVersionLevel++
			LoadConfig()
		} else {
			fmt.Fprintf(os.Stderr, "loaded cfg=%v\n", cfg)
			cfg.VERSION = BUILDDATE
		}
	} else {
		fmt.Fprintf(os.Stderr, "using default cfg=%#v\n", cfg)
		cfg.VERSION = BUILDDATE
	}
}

func save() {
	configBytes, err := toml.Marshal(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save 1: %v\n", err)
	}
	err = CreateOrReplaceFileContent(TMPFN, string(configBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save 2: %v\n", err)
	}
}

func setupTerminal() {
	var err error
	var ROWS int
	stdin, err = tty.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open tty: %v\n", err)
		os.Exit(1)
	}
	if stdin == nil {
		fmt.Fprintf(os.Stderr, "Failed to configure tty\n")
		os.Exit(1)
	}
	ColumnsStr := os.Getenv("COLUMNS")
	if ColumnsStr > "" {
		COLUMNS, err = strconv.Atoi(ColumnsStr)
		if err == nil {
			fmt.Printf("Read COLUMNS=%d from environment\n", COLUMNS);
			return
		}
		fmt.Printf("Unable to parse COLUMNS=%q from environment: %v\n", ColumnsStr, err);
	}
	ROWS, COLUMNS, err = stdin.Size()
	if err == nil {
		fmt.Printf("Read ROWS=%d COLUMNS=%d from tty\n", ROWS, COLUMNS);
	} else {
		COLUMNS = 80
		fmt.Printf("Using default COLUMNS=%d\n", COLUMNS);
	}
}

func kbThread() {
	for RUN == 1 {
		r, err := stdin.ReadRune()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read tty: %v\n", err)
		}
		s := string(r)
		if len(s) == 1 {
			KEYBUFFER <- s
		} else {
			fmt.Println("Key press => " + s)
		}
	}
}

func shutdownTerminal() {
	stdin.Close()
	stdin = nil
	time.Sleep(100 * time.Millisecond)
}

func checkAndSave() {
	if cfg.HIGHVAL > MAX {
		cfg.HIGHVAL = MAX
	} else if cfg.HIGHVAL < 0 {
		cfg.HIGHVAL = 0
	} else if cfg.TARGET < 16 && cfg.HIGHVAL < 1 {
		cfg.HIGHVAL = 1
	}
	if cfg.LOWVAL > MAX {
		cfg.LOWVAL = MAX
	} else if cfg.LOWVAL < 0 {
		cfg.LOWVAL = 0
	} else if cfg.TARGET < 16 && cfg.LOWVAL < 1 {
		cfg.LOWVAL = 1
	}
	if cfg.TARGET > MAX {
		cfg.TARGET = MAX
	} else if cfg.TARGET < 1 {
		cfg.TARGET = 1
	}
	if cfg.HSLEEPUS > 1048576 {
		cfg.HSLEEPUS = 1048576
	} else if cfg.HSLEEPUS < 10 {
		cfg.HSLEEPUS = 10
	}
	if cfg.LSLEEPUS > 1048576 {
		cfg.LSLEEPUS = 1048576
	} else if cfg.LSLEEPUS < 10 {
		cfg.LSLEEPUS = 10
	}
	if cfg.TSLEEPUS > 1048576 {
		cfg.TSLEEPUS = 1048576
	} else if cfg.TSLEEPUS < 10 {
		cfg.TSLEEPUS = 10
	}
	if cfg.ZINTERVAL > 1048576 {
		cfg.ZINTERVAL = 1048576
	} else if cfg.ZINTERVAL == 1 {
		cfg.ZINTERVAL = 4
	} else if cfg.ZINTERVAL == 2 {
		cfg.ZINTERVAL = 0
	} else if cfg.ZINTERVAL < 0 {
		cfg.ZINTERVAL = 0
	}
	if cfg.SLEEPSCALE > 1048576 {
		cfg.SLEEPSCALE = 1048576
	} else if cfg.SLEEPSCALE < 1 {
		cfg.SLEEPSCALE = 1
	}
	show()
	save()
}

func getTimeString() (string) {
	return time.Now().Format("15:04:05")
}

func show() {
	var time = getTimeString()
	var zintstr = ""
	if cfg.ZINTERVAL > 0 {
		zintstr = fmt.Sprintf("ZINTERVAL=%d, ", cfg.ZINTERVAL)
	}
	var long = fmt.Sprintf("[%s] H/L-VAL=%d/%d, TARGET=%d, H/L-SLEEPUS=%d/%d, TSLEEPUS=%d, SLEEPSCALE=%d, %sHZ=%d.", time, cfg.HIGHVAL, cfg.LOWVAL, cfg.TARGET, cfg.HSLEEPUS, cfg.LSLEEPUS, cfg.TSLEEPUS, cfg.SLEEPSCALE, zintstr, HZ)
	if len(long) > COLUMNS {
		if cfg.ZINTERVAL > 0 {
			zintstr = fmt.Sprintf("ZI=%d, ", cfg.ZINTERVAL)
		}
		fmt.Printf("\r[%s] H/Lv=%d/%d, TGTv=%d, H/LSus=%d/%d, TSus=%d, S=%d, %sHZ=%d.", time, cfg.HIGHVAL, cfg.LOWVAL, cfg.TARGET, cfg.HSLEEPUS, cfg.LSLEEPUS, cfg.TSLEEPUS, cfg.SLEEPSCALE, zintstr, HZ)
	} else {
		fmt.Printf("\r%s", long)
	}
}

func readKey() (byte) {
	var ch string
	select {
		case ch = <- KEYBUFFER:
			if len(ch) > 1 {
				fmt.Printf("Long key: %q\n", ch)
				time.Sleep(5)
				return '\000'
			} else {
				return ch[0]
			}
		default:
			return '\000'
	}
}

func decIfAbove(start, amount int) (int) {
	if start > amount {
		start -= amount
	}
	return start
}

func checkKey() {
	ch := readKey()
	for ch != 0 {
		switch(ch) {
			case 'q': RUN=0
			case 'A': cfg.HIGHVAL += 1
			case 'Z': cfg.HIGHVAL -= 1
			case 'a': cfg.LOWVAL += 1
			case 'z': cfg.LOWVAL -= 1
			case 's': cfg.TARGET  += 1
			case 'x': cfg.TARGET  -= 1
			case 'E': cfg.HSLEEPUS *= 2
			case 'D': cfg.HSLEEPUS += cfg.SLEEPSCALE
			case 'C': cfg.HSLEEPUS = decIfAbove(cfg.HSLEEPUS, cfg.SLEEPSCALE)
			case 'e': cfg.LSLEEPUS *= 2
			case 'd': cfg.LSLEEPUS += cfg.SLEEPSCALE
			case 'c': cfg.LSLEEPUS = decIfAbove(cfg.LSLEEPUS, cfg.SLEEPSCALE)
			case 'r': cfg.TSLEEPUS *= 2
			case 'f': cfg.TSLEEPUS += cfg.SLEEPSCALE
			case 'v': cfg.TSLEEPUS = decIfAbove(cfg.TSLEEPUS, cfg.SLEEPSCALE)
			case 'g': cfg.SLEEPSCALE *= 2
			case 'b': cfg.SLEEPSCALE /= 2
			case 'h': cfg.ZINTERVAL = ( cfg.ZINTERVAL * 2 ) + 1
			case 'n': cfg.ZINTERVAL /= 2
			default:
				fmt.Printf("Unknown key: %s\n", ch)
				time.Sleep(5)
				ch = readKey()
				continue
		}
		checkAndSave()
		ch = readKey()
	}
}

func zcheck() {
	if cfg.ZINTERVAL == 0 {
		ZCOUNTER=0
	} else if ZCOUNTER >= cfg.ZINTERVAL {
		setBrightness(0)
		time.Sleep(time.Millisecond)
		ZCOUNTER = 0
	} else {
		ZCOUNTER += 1
	}
}

func hzcheck() {
	HZ += 1
	now := time.Now().Unix()
	if now != LASTHZSEC {
		show()
		LASTHZSEC = time.Now().Unix()
		HZ = 0
	}
}

func testBrightness() (bool) {
	// test reading
	test, err := ReadFileInt(BrightnessFile)
	fmt.Printf("brightness=%d, err=%v\n", test, err)
	if test < 1 {
		return false
	}
	// test writing TARGET
	setBrightness(cfg.TARGET)
	fmt.Printf("TARGET=%d, brightness=%d, err=%v\n", cfg.TARGET, test, err)
	test, err = ReadFileInt(BrightnessFile)
	fmt.Printf("brightness=%d, err=%v\n", test, err)
	if test != cfg.TARGET {
		return false
	}
	// test writing HIGHVAL
	setBrightness(cfg.HIGHVAL)
	fmt.Printf("HIGHVAL=%d, brightness=%d, err=%v\n", cfg.HIGHVAL, test, err)
	test, err = ReadFileInt(BrightnessFile)
	fmt.Printf("brightness=%d, err=%v\n", test, err)
	if test != cfg.HIGHVAL {
		return false
	}
	// test writing LOWVAL
	setBrightness(cfg.LOWVAL)
	fmt.Printf("LOWVAL=%d, brightness=%d, err=%v\n", cfg.LOWVAL, test, err)
	test, err = ReadFileInt(BrightnessFile)
	fmt.Printf("brightness=%d, err=%v\n", test, err)
	if test != cfg.LOWVAL {
		return false
	}
	// all good!
	fmt.Printf("brightness ok!\n")
	return true
}

func main() {
	startTime := time.Now()
	if UID != 0 {
		fmt.Printf("WARNING: this process may require elevated privileges.\n")
	}
	
	LoadConfig()
	FindAndVerifyBacklightInterface()
	setupTerminal()
	defer shutdownTerminal()
	go kbThread()
	
	// check config values
	checkAndSave()

	// now loop around target brightness
	for RUN == 1 {
		hzcheck()
		checkKey()
		zcheck()
		setBrightness(cfg.HIGHVAL)
		time.Sleep(time.Duration(cfg.HSLEEPUS) * time.Microsecond)
		checkKey()
		zcheck()
		setBrightness(cfg.TARGET)
		time.Sleep(time.Duration(cfg.TSLEEPUS) * time.Microsecond)
		checkKey()
		zcheck()
		setBrightness(cfg.LOWVAL)
		time.Sleep(time.Duration(cfg.LSLEEPUS) * time.Microsecond)
	}

	// finalize by using exactly TARGET
	fmt.Printf("\n[%s] [TGT] %d.  Done.\n", getTimeString(), cfg.TARGET)
	setBrightness(cfg.TARGET)
	elapsedTime := time.Now().Sub(startTime)
	fmt.Printf("Ran for a total of %v seconds\n", elapsedTime)
}
