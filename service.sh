#!/system/bin/sh
# Sortify v4.0 Service (Native WebUI Version)

MODDIR=${0%/*}
CONF="$MODDIR/sortify.conf"
LOG="$MODDIR/sortify.log"

# 1. Load Defaults if config missing
if [ ! -f "$CONF" ]; then
    echo "INTERVAL=300" > "$CONF"
fi

# (WebUI is now handled natively by KernelSU via 'webroot' folder. No httpd needed.)

# 2. Wait for Storage
wait_until_storage() {
    until [ -d "/sdcard/Download" ]; do
        sleep 10
    done
}
wait_until_storage

# 3. Main Loop (Backgrounded)
(
    while true; do
        # Re-read config every cycle to get new INTERVAL
        if [ -f "$CONF" ]; then
            source "$CONF"
        fi
        
        # Run the action script
        # We redirect stdout/stderr to log to capture any 'echo' from action.sh
        sh "$MODDIR/action.sh" >> "$LOG" 2>&1
        
        # Log the service heartbeat
        echo "[Service] $(date '+%Y-%m-%d %H:%M:%S') - Cycle complete. Sleeping ${INTERVAL}s" >> "$LOG"
        
        # Prune log (Keep last 200 lines)
        tail -n 200 "$LOG" > "$LOG.tmp" && mv "$LOG.tmp" "$LOG"
        
        sleep "${INTERVAL:-300}"
    done
) &  # <--- CRITICAL: Run entire loop in background
