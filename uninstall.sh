#!/system/bin/sh
MODDIR=${0%/*}
# The daemon checks remove/disable flags and its module path, then exits cleanly.
# Do not kill an in-progress transfer or delete organized files/history/settings.
printf '%s\n' 'Sortify removed. Organized files and /data/adb/sortify settings/history are preserved.'
