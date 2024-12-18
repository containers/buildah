# !/bin/sh

function output {
    for mask in /proc/acpi /proc/kcore /proc/keys /proc/latency_stats /proc/sched_debug /proc/scsi /proc/timer_list /proc/timer_stats /sys/dev/block /sys/devices/virtual/powercap /sys/firmware /sys/fs/selinux; do
	test -e $mask || continue
	test -f $mask && cat $mask 2> /dev/null
	test -d $mask && ls $mask
    done
}
output=$(output | wc -c )
if [ $output == 0 ]; then
    echo "masked"
else
    echo "unmasked"
fi
