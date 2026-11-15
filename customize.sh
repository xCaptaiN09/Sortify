#!/system/bin/sh
# Select a bundled executable; do not create or touch user storage at install time.
STATE=/data/adb/sortify
ui_print "- Sortify 5.0"
case "$ARCH" in
    arm64|arm|x64|x86) SORTIFY_ARCH=$ARCH ;;
    *) abort "Unsupported architecture: $ARCH" ;;
esac
HELPER="$MODPATH/bin/$SORTIFY_ARCH/sortify"
[ -f "$HELPER" ] || abort "Missing native helper. Install the built Sortify release ZIP."
cp "$HELPER" "$MODPATH/bin/sortify" || abort "Could not install Sortify helper."
set_perm "$MODPATH/bin/sortify" 0 0 0755
for script in action.sh service.sh sortify.sh uninstall.sh; do
    set_perm "$MODPATH/$script" 0 0 0755
done
# The universal ZIP carries four builds; an installed module needs only one.
for architecture in arm64 arm x64 x86; do
    rm -rf "$MODPATH/bin/$architecture" || abort "Could not remove unused helpers."
done
mkdir -p "$STATE" || abort "Could not create Sortify state directory."
chmod 0700 "$STATE" || abort "Could not protect Sortify state directory."
# Read a single numeric v4 setting as data, never execute the legacy file.
LEGACY_CONF=/data/adb/modules/sortify/sortify.conf
if [ ! -f "$STATE/config.json" ] && [ -f "$LEGACY_CONF" ]; then
    INTERVAL=$(awk -F= '/^INTERVAL=[0-9]+$/ {value=$2; count++} END {if(count==1) print value}' "$LEGACY_CONF")
    case "$INTERVAL" in
        ''|*[!0-9]*) abort "Invalid legacy interval. Fix the v4 interval before upgrading." ;;
    esac
    "$MODPATH/bin/sortify" --state "$STATE" --storage /sdcard --module "$MODPATH" init --legacy-interval "$INTERVAL" >"$STATE/install.log" 2>&1 || abort "Settings migration failed. See $STATE/install.log."
    ui_print "- Previous settings migrated."
else
    "$MODPATH/bin/sortify" --state "$STATE" --storage /sdcard --module "$MODPATH" init >"$STATE/install.log" 2>&1 || abort "Settings setup failed. See $STATE/install.log."
    ui_print "- Settings ready."
fi
rm -f "$STATE/install.log"
ui_print "- Ready. Reboot, then open the WebUI."
