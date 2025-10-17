#!/system/bin/sh
# Sortify v1.3.1 by xCaptaiN09
# Automatically organizes files from Downloads into categorized folders.

BASE="/sdcard/Sortify"

mkdir -p "$BASE/Documents"
mkdir -p "$BASE/Images"
mkdir -p "$BASE/Videos"
mkdir -p "$BASE/Audio"
mkdir -p "$BASE/Archives"
mkdir -p "$BASE/Apps"
mkdir -p "$BASE/Others"

ui_print "✔ Sortify folders created at $BASE"
ui_print "✔ Module by xCaptaiN09"
