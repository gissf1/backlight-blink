#!/bin/bash
# reset backlight hardware by flashing between 2 brightness levels
#

if [ $UID != 0 ]; then
	sudo $0 $*
	exit $?
fi

# default values
RUN=1
HIGHVAL=255
TARGET=216
HSLEEPUS=100000
TSLEEPUS=100000
SLEEPSCALE=128
ZINTERVAL=0
ZCOUNTER=0
HZ=0
LASTHZSEC=0

# find and verify backlight interfaces
DIR="/sys/class/backlight/"
DIR=$( find "$DIR" -mindepth 1 -maxdepth 1 -type d,l | sed -r 's/^$// ; T ; d' | head -n1 )
if [ "$DIR" == "" ]; then
	echo "FATAL: cannot find any backlight devices in /sys/class/backlight/" >&2
	exit 1
fi
if [ ! -e "$DIR/max_brightness" ]; then
	echo "FATAL: cannot find file: $DIR/max_brightness" >&2
	exit 1
fi
if [ ! -r "$DIR/max_brightness" ]; then
	echo "FATAL: cannot read file: $DIR/max_brightness" >&2
	exit 1
fi
MAX=$( cat "$DIR/max_brightness" )
if [ ! -e "$DIR/brightness" ]; then
	echo "FATAL: cannot find file: $DIR/brightness" >&2
	exit 1
fi
if [ ! -w "$DIR/brightness" ]; then
	echo "FATAL: cannot write file: $DIR/brightness" >&2
	exit 1
fi
if [ -r "$DIR/actual_brightness" ]; then
	TARGET=$( cat "$DIR/actual_brightness" )
fi

# read last config
TMPFN="/tmp/$( basename "$0" ).tmp"
if [ -e "$TMPFN" ]; then
	. $TMPFN
fi

save() {
	echo -e "
HIGHVAL=$HIGHVAL
TARGET=$TARGET
HSLEEPUS=$HSLEEPUS
TSLEEPUS=$TSLEEPUS
ZINTERVAL=$ZINTERVAL
SLEEPSCALE=$SLEEPSCALE
" > "$TMPFN"
}

checkAndSave() {
	if [ $HIGHVAL -gt $MAX ]; then
		HIGHVAL=$MAX
	elif [ $HIGHVAL -lt 0 ]; then
		HIGHVAL=0
	elif [ $TARGET -lt 16 -a $HIGHVAL -lt 1 ]; then
		HIGHVAL=1
	fi
	if [ $TARGET -gt $MAX ]; then
		TARGET=$MAX
	elif [ $TARGET -lt 1 ]; then
		TARGET=1
	fi
	if [ $HSLEEPUS -gt 1048576 ]; then
		HSLEEPUS=1048576
	elif [ $HSLEEPUS -lt 10 ]; then
		HSLEEPUS=10
	fi
	if [ $TSLEEPUS -gt 1048576 ]; then
		TSLEEPUS=1048576
	elif [ $TSLEEPUS -lt 10 ]; then
		TSLEEPUS=10
	fi
	if [ $ZINTERVAL -gt 1048576 ]; then
		ZINTERVAL=1048576
	elif [ $ZINTERVAL -eq 1 ]; then
		ZINTERVAL=4
	elif [ $ZINTERVAL -eq 2 ]; then
		ZINTERVAL=0
	elif [ $ZINTERVAL -lt 0 ]; then
		ZINTERVAL=0
	fi
	if [ $SLEEPSCALE -gt 1048576 ]; then
		SLEEPSCALE=1048576
	elif [ $SLEEPSCALE -lt 1 ]; then
		SLEEPSCALE=1
	fi
	show
	save
}

show() {
	if [ -t 1 ]; then
		local T="${#HIGHVAL} + ${#TARGET} + ${#HSLEEPUS} + ${#TSLEEPUS} + ${#ZINTERVAL} + ${#SLEEPSCALE}"
		T=$(( T ))
		if [ $T -gt 16 ]; then
			echo -en "\r[$( date +%T )] HIv=$HIGHVAL, TGTv=$TARGET, HSus=$HSLEEPUS, TSus=$TSLEEPUS, ZI=$ZINTERVAL, S=$SLEEPSCALE, HZ=$HZ."
		else
			echo -en "\r[$( date +%T )] HIGHVAL=$HIGHVAL, TARGET=$TARGET, HSLEEPUS=$HSLEEPUS, TSLEEPUS=$TSLEEPUS, ZINTERVAL=$ZINTERVAL, SLEEPSCALE=$SLEEPSCALE, HZ=$HZ."
		fi
	fi
}

readKey() {
	read -N 1 -r -s -t 0.0001
	local RET=$?
	if [ $RET -gt 128 ]; then
		# TODO defined as -ge, but we're trying eq instead
		return 128
	elif [ $RET -eq 0 ]; then
		return 0
	else
		echo "Unknown read errorcode: $RET"
		sleep 5
		return 254
	fi
}

checkKey() {
	while readKey ; do
		case "$REPLY" in
			q) RUN=0 ;;
			a) (( ++HIGHVAL )) ; checkAndSave ;;
			z) (( --HIGHVAL )) ; checkAndSave ;;
			s) (( ++TARGET ))  ; checkAndSave ;;
			x) (( --TARGET ))  ; checkAndSave ;;
			e) HSLEEPUS=$(( HSLEEPUS * 2 )) ; checkAndSave ;;
			d) HSLEEPUS=$(( HSLEEPUS + SLEEPSCALE )) ; checkAndSave ;;
			c) [ $HSLEEPUS -gt $SLEEPSCALE ] && HSLEEPUS=$(( HSLEEPUS - SLEEPSCALE )) ; checkAndSave ;;
			r) TSLEEPUS=$(( TSLEEPUS * 2 )) ; checkAndSave ;;
			f) TSLEEPUS=$(( TSLEEPUS + SLEEPSCALE )) ; checkAndSave ;;
			v) [ $TSLEEPUS -gt $SLEEPSCALE ] && TSLEEPUS=$(( TSLEEPUS - SLEEPSCALE )) ; checkAndSave ;;
			g) ZINTERVAL=$(( ( ZINTERVAL * 2 ) + 1 )) ; checkAndSave ;;
			b) ZINTERVAL=$(( ZINTERVAL / 2 )) ; checkAndSave ;;
			h) SLEEPSCALE=$(( SLEEPSCALE * 2 )) ; checkAndSave ;;
			n) SLEEPSCALE=$(( SLEEPSCALE / 2 )) ; checkAndSave ;;
			*) echo "Unknown key: $REPLY" ; sleep 5 ;;
		esac
	done
}

zcheck() {
	if [ $ZINTERVAL -eq 0 ]; then
		ZCOUNTER=0
	elif [ $ZCOUNTER -ge $ZINTERVAL ]; then
		echo 0 > "$DIR/brightness"
		#echo -n "0"
		sleep 0.001
		ZCOUNTER=0
	else
		ZCOUNTER=$(( ZCOUNTER + 1 ))
	fi
}

hzcheck() {
	HZ=$(( HZ + 1 ))
	if [ $SECONDS != $LASTHZSEC ]; then
		HZ=$(( HZ / 2 ))
		show
		LASTHZSEC=$SECONDS
		HZ=0
	fi
}

# check config values
checkAndSave

# now loop around target brightness
while [ "$RUN" == "1" ] ; do
	hzcheck
	checkKey
	zcheck
	echo $TARGET > "$DIR/brightness"
	#echo -n "."
	S=$( printf "%0.8f" $( bc -l <<< "scale=8 ; $TSLEEPUS / 1000000" ) )
	sleep $S
	checkKey
	zcheck
	echo $HIGHVAL > "$DIR/brightness"
	#echo -n ":"
	S=$( printf "%0.8f" $( bc -l <<< "scale=8 ; $HSLEEPUS / 1000000" ) )
	sleep $S
done

# finalize by using exactly TARGET
PL=$TARGET
if [ -t 1 ]; then
	echo -e "\r[$( date +%T )] [TGT] $PL.  Done."
fi
echo $PL > "$DIR/brightness"
echo "Ran for a total of $SECONDS seconds"
