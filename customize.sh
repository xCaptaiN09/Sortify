#!/system/bin/sh
# Sortify v1.4 by xCaptaiN09
# Automatically organizes files from Downloads into categorized folders.

BASE="/sdcard/Sortify"

# Create base folders
mkdir -p "$BASE/Documents"
mkdir -p "$BASE/Images"
mkdir -p "$BASE/Videos"
mkdir -p "$BASE/Audio"
mkdir -p "$BASE/Archives"
mkdir -p "$BASE/Apps"
mkdir -p "$BASE/Others"
mkdir -p "$BASE/LargeFiles"

ui_print "📂 Sortify folders created at $BASE"
ui_print "💾 Added LargeFiles folder for ≥1GB files"
ui_print "✨ Module by xCaptaiN09"
