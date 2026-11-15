#!/system/bin/sh
MODDIR=${0%/*}
STORAGE=${SORTIFY_STORAGE_ROOT:-/sdcard}
STATE=${SORTIFY_STATE_DIR:-/data/adb/sortify}
# Do not canonicalize Android's emulated-storage aliases before storage mounts.
(
    while [ ! -d "$STORAGE" ] || [ ! -r "$STORAGE" ]; do
        [ -d "$MODDIR" ] || exit 0
        [ ! -f "$MODDIR/disable" ] && [ ! -f "$MODDIR/remove" ] || exit 0
        sleep 5
    done
    if command -v getprop >/dev/null 2>&1; then
        while [ "$(getprop sys.boot_completed)" != "1" ]; do
            [ -d "$MODDIR" ] || exit 0
            [ ! -f "$MODDIR/disable" ] && [ ! -f "$MODDIR/remove" ] || exit 0
            sleep 5
        done
    fi
    [ -d "$MODDIR" ] || exit 0
    [ ! -f "$MODDIR/disable" ] && [ ! -f "$MODDIR/remove" ] || exit 0
    mkdir -p "$STATE" || exit 1
    chmod 0700 "$STATE" || exit 1
    # Truncate on startup; the singleton helper handles its persistent status.
    sh "$MODDIR/sortify.sh" serve >"$STATE/service.log" 2>&1
) &
