#!/system/bin/sh
# Define ui_print function
ui_print() {
    echo "$1"
}
# Sortify v4.0 - Manual Action

ui_print "▶ Sortify: Manual sort started"

DOWNLOADS="/sdcard/Download"
DEST_BASE="/sdcard/Sortify"

mkdir -p "$DEST_BASE/Documents" \
         "$DEST_BASE/Images" \
         "$DEST_BASE/Videos" \
         "$DEST_BASE/Audio" \
         "$DEST_BASE/Archives" \
         "$DEST_BASE/Apps" \
         "$DEST_BASE/Others" \
         "$DEST_BASE/Duplicates"

DOC_EXT="pdf doc docx txt xls xlsx ppt pptx csv"
IMG_EXT="jpg jpeg png gif bmp webp heic heif svg"
VID_EXT="mp4 mkv avi mov webm flv mpeg mpg 3gp"
AUD_EXT="mp3 m4a flac wav ogg opus aac wma"
ARC_EXT="zip rar 7z tar gz bz2 iso"
APP_EXT="apk exe"

move_files() {
    dest="$1"
    shift
    for ext in "$@"; do
        find "$DOWNLOADS" -maxdepth 1 -type f \
            ! -name ".*" \
            ! -name "*.crdownload" \
            ! -name "*.partial" \
            ! -name "*.tmp" \
            -iname "*.$ext" -print0 | while IFS= read -r -d '' file; do
                filename=$(basename "$file")
                if [ -e "$dest/$filename" ]; then
                    mv -f "$file" "$DEST_BASE/Duplicates/"
                else
                    mv -f "$file" "$dest/"
                fi
            done
    done
}

move_files "$DEST_BASE/Documents" $DOC_EXT
move_files "$DEST_BASE/Images" $IMG_EXT
move_files "$DEST_BASE/Videos" $VID_EXT
move_files "$DEST_BASE/Audio" $AUD_EXT
move_files "$DEST_BASE/Archives" $ARC_EXT
move_files "$DEST_BASE/Apps" $APP_EXT

find "$DOWNLOADS" -maxdepth 1 -type f \
    ! -name ".*" \
    ! -name "*.crdownload" \
    ! -name "*.partial" \
    ! -name "*.tmp" -print0 | while IFS= read -r -d '' file; do
        filename=$(basename "$file")
        if [ -e "$DEST_BASE/Others/$filename" ]; then
            mv -f "$file" "$DEST_BASE/Duplicates/"
        else
            mv -f "$file" "$DEST_BASE/Others/"
        fi
    done

date "+[%Y-%m-%d %H:%M:%S] Manual sort triggered" >> "$DEST_BASE/sortify.log"

ui_print "✔ Sortify: Manual sort completed"
