#!/system/bin/sh
# Sortify v2.0 by xCaptaiN09
# Automatically organizes files from Downloads into categorized folders.
# Now with duplicate detection and improved stability.

BASE="/sdcard/Sortify"

# Create all category folders
mkdir -p "$BASE/Documents"
mkdir -p "$BASE/Images"
mkdir -p "$BASE/Videos"
mkdir -p "$BASE/Audio"
mkdir -p "$BASE/Archives"
mkdir -p "$BASE/Apps"
mkdir -p "$BASE/Others"
mkdir -p "$BASE/Duplicates"   # New in v2.0

ui_print "✔ Sortify v2.0 folders created at $BASE"
ui_print "✔ Module by xCaptaiN09"
ui_print "✔ New feature: Duplicate files are now moved to $BASE/Duplicates"
