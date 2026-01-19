<p align="center">
  <img src="banner.png" alt="Sortify Banner" width="100%" />
</p>

# 📁 Sortify

**Author:** [xCaptaiN09](https://github.com/xCaptaiN09)  
**Version:** 4.0  

Sortify is a powerful **Magisk / KernelSU** module that automatically organizes files in your `/sdcard/Download/` folder into categorized subfolders. Now smarter, safer, and equipped with a **Native WebUI** for easy configuration.

---

## 📦 Features

* **⚡ Automatic Sorting:** Runs automatically in the background (Default: every 5 mins).
* **🌐 Native WebUI:** **NEW!** Configure sorting intervals directly inside the KernelSU Manager (No browser needed).
* **▶ Manual Trigger:** Supports Magisk/KSU Action Button to sort instantly.
* **📂 Smart Categories:** Sorts Documents, Images, Videos, Audio, Archives, Apps, and more.
* **🛡️ Integrity Protection:** Skips hidden/incomplete files (`.crdownload`, `.partial`, `.tmp`) to prevent corruption.
* **🗂️ Duplicate Detection:** Automatically moves duplicate files to `/sdcard/Sortify/Duplicates`.
* **🪶 Lightweight:** 100% offline, uses native system resources.

---

## 🧩 Installation

1.  Download `Sortify-v4.0.zip` from Releases.
2.  Flash through **Magisk** or **KernelSU**.
3.  Reboot your device.
4.  **Done!** Sortify is now running.

---

## ⚙️ Configuration (WebUI)

**Sortify v4.0** introduces a native dashboard to change settings without editing code.

### How to access:
1.  Open **KernelSU Manager**.
2.  Go to the **Modules** tab.
3.  Find **Sortify**.
4.  Tap the **Settings / Globe Icon** on the module card.

### Settings available:
* **Sort Interval:** Change how often the background service checks for new files (in seconds).
    * *Default:* 300 seconds (5 minutes).
    * *Changes apply immediately.*

---

## ▶ Manual Trigger

You can force a sort immediately without waiting for the timer:
* **Magisk/KSU App:** Tap the **Action** button on the module card.
* **Terminal:** Run `su -c sh /data/adb/modules/sortify/action.sh`

---

## 🧼 Uninstall

1.  Remove Sortify from your Module Manager.
2.  Reboot.
3.  *(Optional)* Delete the `/sdcard/Sortify` folder if you no longer need the organized files.

---

## 🧾 Changelog

### v4.0 (2026-01-19)
* **🌐 Native WebUI:** Added KernelSU `webroot` support for internal configuration (Replaces port 8081 server).
* **⚡ Optimized Service:** Removed BusyBox HTTPD dependency for lighter footprint.
* **🚀 Performance:** Improved background service logic.
* **🔧 Stability:** Fixed permission handling for scripts and config files.

### v3.0 (2026-01-19)
* ▶ Added manual one-tap sorting via Magisk Action button.
* ♻️ Refined automatic background sorting.

### v2.0 (2025-10-18)
* 🆕 Added duplicate detection (moves to `/sdcard/Sortify/Duplicates`).
* 🛠 Fixed uninstall script path.
* ⚡ Centralized extension handling.
* 🔒 Safer file moves (no accidental overwrites).

### v1.3 (2025-10-10)
* 🚫 Prevented moving hidden/incomplete downloads (`.crdownload`, etc.).
* 🗂️ Renamed main folder to `/sdcard/Sortify`.
* 🧾 Updated logs to `sortify.log` with auto-trimming.

---

## 🧡 Credits

* Developed by **[xCaptaiN09](https://github.com/xCaptaiN09)**
* Powered by BusyBox & Android Shell

[![GitHub release](https://img.shields.io/github/v/release/xCaptaiN09/Sortify)](https://github.com/xCaptaiN09/Sortify/releases)
