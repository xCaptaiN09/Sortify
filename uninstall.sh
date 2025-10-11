#!/system/bin/sh
# Sortify Uninstaller - by xCaptaiN09

BASE="/sdcard/Organizer"

echo "[Uninstall] $(date '+%Y-%m-%d %H:%M:%S') - Sortify uninstall initiated." >> "$BASE/organizer.log"

# Optional: Uncomment to delete folders on uninstall
# rm -rf "$BASE/Documents" "$BASE/Images" "$BASE/Videos" "$BASE/Audio" "$BASE/Archives" "$BASE/Apps" "$BASE/Others"
# rmdir "$BASE" 2>/dev/null

echo "[Uninstall] Sortify uninstalled successfully." >> "$BASE/organizer.log"
exit 0
