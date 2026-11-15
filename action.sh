#!/system/bin/sh
# Manual runs use the same lock, eligibility and history as scheduled runs.
MODDIR=${0%/*}
exec sh "$MODDIR/sortify.sh" run
