<p align="center">
  <img src="banner.png" alt="Sortify Banner" width="100%" />
</p>

# 📁 Sortify
**Author:** [xCaptaiN09](https://github.com/xCaptaiN09)  
**Version:** 2.0  

Sortify is a Magisk / KernelSU module that automatically organizes files in your `/sdcard/Download/` folder into categorized folders — now smarter, safer, and cleaner than ever.  

---

### 📦 Features
- 🕒 Runs automatically every 5 minutes  
- 📂 Organizes documents, images, videos, audio, archives, apps, and more  
- 🧠 Skips hidden and incomplete files (`.crdownload`, `.partial`, `.tmp`) to prevent corruption  
- 🪶 Lightweight and fully offline  
- 🧾 Keeps logs at `/sdcard/Sortify/sortify.log` (auto-trimmed)  
- 🗂️ **New in v2.0:** Detects duplicate files and moves them into `/sdcard/Sortify/Duplicates`  

---

### 🧩 Installation
1. Flash `Sortify-v2.0.zip` through **Magisk** or **KernelSU**  
2. Reboot  
3. Sortify will automatically organize your downloads ✅  

---

### ⚙️ Configuration
You can customize:
- Organization interval → edit `sleep 300` in `service.sh`  
- Add or remove file extensions inside `service.sh`  

---

### 🧼 Uninstall
Simply remove **Sortify** from **Magisk Manager** and reboot.  
All created folders (`/sdcard/Sortify`) will remain unless you manually delete them.  

---

### 🧾 Changelog

#### v2.0 (2025-10-18)
- 🆕 Added **duplicate detection** → duplicate files are now moved to `/sdcard/Sortify/Duplicates`  
- 🛠 Fixed uninstall script path (now points to `/sdcard/Sortify`)  
- ⚡ Centralized extension handling for easier customization  
- 🧾 Improved logging stability and trimming  
- 🔒 Safer file moves (no accidental overwrites)  

#### v1.3.1 (2025-10-17)
- 🧹 Removed unused `post-fs-data.sh` placeholder for a cleaner module structure  
- ⚙️ Internal cleanup and optimization  
- 🚫 No functional changes  

#### v1.3 (2025-10-10)
- 🚫 Prevented moving hidden and incomplete download files (`.crdownload`, `.partial`, `.tmp`)  
- 🗂️ Renamed main folder from `/sdcard/Organizer` → `/sdcard/Sortify`  
- 🧾 Updated logs to `sortify.log` with auto-trimming  
- ⚡ Improved background handling and reliability  

#### v1.2 (Initial Release)
- Initial public version  
- Automatic file categorization every 5 minutes  
- Base folder and logging system  

---

### 🧡 Credits
Developed by [xCaptaiN09](https://github.com/xCaptaiN09)  

---

[![GitHub release](https://img.shields.io/github/v/release/xCaptaiN09/Sortify)](https://github.com/xCaptaiN09/Sortify/releases)
