#!/system/bin/sh
# Sortify v4.0 by xCaptaiN09

BASE="/sdcard/Sortify"

# 1. Create User Directories
ui_print "- Creating folder structure..."
mkdir -p "$BASE/Documents"
mkdir -p "$BASE/Images"
mkdir -p "$BASE/Videos"
mkdir -p "$BASE/Audio"
mkdir -p "$BASE/Archives"
mkdir -p "$BASE/Apps"
mkdir -p "$BASE/Others"
mkdir -p "$BASE/Duplicates"   

ui_print "✔ Sortify folders ready at $BASE"

# 2. Set Module Permissions
ui_print "- Setting permissions..."

# A. Default Rule: Set ALL folders to 755 and ALL files to 644
# This automatically handles 'webroot/index.html' correctly (0644)
set_perm_recursive "$MODPATH" 0 0 0755 0644

# B. Override Scripts: Make them executable (755)
set_perm "$MODPATH/service.sh" 0 0 0755
set_perm "$MODPATH/action.sh" 0 0 0755
set_perm "$MODPATH/uninstall.sh" 0 0 0755

# C. Config File: Ensure writable
if [ -f "$MODPATH/sortify.conf" ]; then
    set_perm "$MODPATH/sortify.conf" 0 0 0644
fi

ui_print "✔ Permissions applied"
