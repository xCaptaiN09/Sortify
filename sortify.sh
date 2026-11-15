#!/system/bin/sh
# Only the native helper parses user data. No configuration is sourced here.
MODDIR=${0%/*}
STATE=${SORTIFY_STATE_DIR:-/data/adb/sortify}
STORAGE=${SORTIFY_STORAGE_ROOT:-/sdcard}
HELPER="$MODDIR/bin/sortify"

if [ ! -x "$HELPER" ]; then
    printf '%s\n' '{"ok":false,"error":"Sortify native helper is missing. Install the built release ZIP."}'
    exit 1
fi
if [ ! -f "$STATE/config.json" ]; then
    "$HELPER" --state "$STATE" --storage "$STORAGE" --module "$MODDIR" init >/dev/null || exit 1
fi
exec "$HELPER" --state "$STATE" --storage "$STORAGE" --module "$MODDIR" "$@"
