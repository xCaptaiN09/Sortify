#!/system/bin/sh
# Sortify v1.4 by xCaptaiN09
# Smart Auto Organizer for Android Downloads
# Now with Large Files handling (≥1GB)

DOWNLOADS="/sdcard/Download"
DEST_BASE="/sdcard/Sortify"

sleep 30  # Wait for storage to mount

# Create all destination folders
mkdir -p "$DEST_BASE/Documents" \
         "$DEST_BASE/Images" \
         "$DEST_BASE/Videos" \
         "$DEST_BASE/Audio" \
         "$DEST_BASE/Archives" \
         "$DEST_BASE/Apps" \
         "$DEST_BASE/Others" \
         "$DEST_BASE/LargeFiles"

# Function to move files based on extensions
move_files() {
    dest="$1"
    shift
    exts="$@"
    for ext in $exts; do
        find "$DOWNLOADS" -maxdepth 1 -type f -readable -size +0c \
             ! -name ".*" ! -name "*.crdownload" ! -name "*.partial" ! -name "*.tmp" \
             -iname "*.$ext" -exec mv -f "{}" "$dest/" \; 2>/dev/null
    done
}

# Function to move large files (≥1GB)
move_large_files() {
    find "$DOWNLOADS" -maxdepth 1 -type f -readable -size +1073741824c \
         ! -name ".*" ! -name "*.crdownload" ! -name "*.partial" ! -name "*.tmp" \
         -exec mv -f "{}" "$DEST_BASE/LargeFiles/" \; 2>/dev/null
}

# Function to organize remaining downloads
organize_downloads() {
    move_files "$DEST_BASE/Documents" pdf doc docx txt xls xlsx ppt pptx csv
    move_files "$DEST_BASE/Images" jpg jpeg png gif bmp webp heic heif svg
    move_files "$DEST_BASE/Videos" mp4 mkv avi mov webm flv mpeg mpg 3gp
    move_files "$DEST_BASE/Audio" mp3 m4a flac wav ogg opus aac wma
    move_files "$DEST_BASE/Archives" zip rar 7z tar gz bz2 iso
    move_files "$DEST_BASE/Apps" apk exe
    # Move remaining eligible files to "Others"
    find "$DOWNLOADS" -maxdepth 1 -type f -readable -size +0c \
         ! -name ".*" ! -name "*.crdownload" ! -name "*.partial" ! -name "*.tmp" \
         -exec mv -f "{}" "$DEST_BASE/Others/" \; 2>/dev/null
}

# Main loop — runs every 5 minutes (300s)
while true; do
    move_large_files
    organize_downloads
    date "+[%Y-%m-%d %H:%M:%S] Organized downloads" >> "$DEST_BASE/sortify.log"
    tail -n 200 "$DEST_BASE/sortify.log" > "$DEST_BASE/sortify.tmp" && mv "$DEST_BASE/sortify.tmp" "$DEST_BASE/sortify.log"
    sleep 300
done &
