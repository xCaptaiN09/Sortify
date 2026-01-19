#!/system/bin/sh
# Sortify v3.0 Uninstaller - by xCaptaiN09

BASE="/sdcard/Sortify"
LOG="$BASE/sortify.log"

echo "[Uninstall] $(date '+%Y-%m-%d %H:%M:%S') - Sortify uninstall initiated." >> "$LOG"

# Optional: Uncomment the lines below if you want to delete all Sortify folders on uninstall
# rm -rf "$BASE/Documents" \
#        "$BASE/Images" \
#        "$BASE/Videos" \
#        "$BASE/Audio" \
#        "$BASE/Archives" \
#        "$BASE/Apps" \
#        "$BASE/Others" \
#        "$BASE/Duplicates"
# rmdir "$BASE" 2>/dev/null

echo "[Uninstall] Sortify uninstalled successfully." >> "$LOG"
exit 0