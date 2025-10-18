#!/system/bin/sh
# Sortify v2.0 by xCaptaiN09
# Smart Auto Organizer for Android Downloads
# Now with duplicate detection, safer filename handling, and improved stability

DOWNLOADS="/sdcard/Download"
DEST_BASE="/sdcard/Sortify"
INTERVAL=300   # Interval in seconds (default: 5 minutes)

sleep 30  # Wait for storage to mount

# Create all category folders
mkdir -p "$DEST_BASE/Documents" \
         "$DEST_BASE/Images" \
         "$DEST_BASE/Videos" \
         "$DEST_BASE/Audio" \
         "$DEST_BASE/Archives" \
         "$DEST_BASE/Apps" \
         "$DEST_BASE/Others" \
         "$DEST_BASE/Duplicates"

# Define extensions
DOC_EXT="pdf doc docx txt xls xlsx ppt pptx csv"
IMG_EXT="jpg jpeg png gif bmp webp heic heif svg"
VID_EXT="mp4 mkv avi mov webm flv mpeg mpg 3gp"
AUD_EXT="mp3 m4a flac wav ogg opus aac wma"
ARC_EXT="zip rar 7z tar gz bz2 iso"
APP_EXT="apk exe"

# Move files with duplicate detection
move_files() {
    dest="$1"
    shift
    exts="$@"
    for ext in $exts; do
        find "$DOWNLOADS" -maxdepth 1 -type f \
            ! -name ".*" \
            ! -name "*.crdownload" \
            ! -name "*.partial" \
            ! -name "*.tmp" \
            -iname "*.$ext" -print0 | while IFS= read -r -d '' file; do
                filename=$(basename "$file")
                if [ -e "$dest/$filename" ]; then
                    # Duplicate found → move to Duplicates
                    mv -f "$file" "$DEST_BASE/Duplicates/"
                    echo "[Duplicate] $filename moved to Duplicates" >> "$DEST_BASE/sortify.log"
                else
                    mv -f "$file" "$dest/"
                fi
            done
    done
}

# Organize downloads into categories
organize_downloads() {
    move_files "$DEST_BASE/Documents" $DOC_EXT
    move_files "$DEST_BASE/Images" $IMG_EXT
    move_files "$DEST_BASE/Videos" $VID_EXT
    move_files "$DEST_BASE/Audio" $AUD_EXT
    move_files "$DEST_BASE/Archives" $ARC_EXT
    move_files "$DEST_BASE/Apps" $APP_EXT

    # Everything else → Others
    find "$DOWNLOADS" -maxdepth 1 -type f \
        ! -name ".*" \
        ! -name "*.crdownload" \
        ! -name "*.partial" \
        ! -name "*.tmp" -print0 | while IFS= read -r -d '' file; do
            filename=$(basename "$file")
            if [ -e "$DEST_BASE/Others/$filename" ]; then
                mv -f "$file" "$DEST_BASE/Duplicates/"
                echo "[Duplicate] $filename moved to Duplicates" >> "$DEST_BASE/sortify.log"
            else
                mv -f "$file" "$DEST_BASE/Others/"
            fi
        done
}

# Main loop
while true; do
    organize_downloads
    date "+[%Y-%m-%d %H:%M:%S] Organized downloads" >> "$DEST_BASE/sortify.log"
    # Trim log to last 200 lines
    tail -n 200 "$DEST_BASE/sortify.log" > "$DEST_BASE/sortify.tmp" && mv "$DEST_BASE/sortify.tmp" "$DEST_BASE/sortify.log"
    sleep $INTERVAL
done &
