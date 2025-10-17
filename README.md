<p align="center">
  <img src="https://raw.githubusercontent.com/xCaptaiN09/Sortify/main/banner.png" width="800" alt="Sortify Banner">
</p>

# 📁 Sortify
**Author:** [xCaptaiN09](https://github.com/xCaptaiN09)  
**Version:** 1.4  

Sortify is a Magisk / KernelSU module that automatically organizes files in your `/sdcard/Download/` folder into categorized folders — now even smarter with **large file detection**.

---

### 📦 Features
- 🕒 Runs automatically every 5 minutes  
- 📂 Organizes documents, images, videos, audio, archives, apps, and other files  
- 🧠 Skips hidden and incomplete files (`.crdownload`, `.partial`, `.tmp`)  
- 💾 Moves large files (≥1GB) to a special **`/sdcard/Sortify/LargeFiles`** folder  
- 🧾 Keeps logs at `/sdcard/Sortify/sortify.log` (auto-trimmed to 200 lines)  
- 🪶 Lightweight and fully offline  
- 🚫 Safe — never touches hidden or system data  

---

### 🧩 Installation
1. Flash `Sortify-v1.4.zip` through **Magisk** or **KernelSU**  
2. Reboot  
3. Sortify automatically organizes your downloads ✅  

---

### ⚙️ Configuration
You can customize:
- Organization interval → edit `sleep 300` in `service.sh`  
- Add or remove file extensions inside `service.sh`  
- Large file threshold (default: ≥1GB) → adjust `-size +1073741824c` in `move_large_files()`  

---

### 🧼 Uninstall
Simply remove **Sortify** from **Magisk Manager** and reboot.  
All created folders (`/sdcard/Sortify`) remain unless you manually delete them.  

---

### 🧾 Changelog

#### v1.4 (2025-10-17)
- 💾 Added automatic **LargeFiles** folder for ≥1GB files  
- ⚙️ Improved file safety with `-readable` and zero-byte skip  
- 🧹 Updated uninstall paths and cleanup improvements  
- 🪶 Cleaned up logs and UI messages 

#### v1.3.1 (2025-10-17)
- 🧹 Removed unused `post-fs-data.sh` placeholder  
- ⚙️ Internal cleanup and optimization  

#### v1.3 (2025-10-10)
- 🚫 Prevented moving hidden and incomplete files (`.crdownload`, `.partial`, `.tmp`)  
- 🗂️ Renamed main folder to `/sdcard/Sortify`  
- 🧾 Updated logging system  
- ⚡ Improved background handling  

#### v1.2 (Initial Release)
- Automatic categorization every 5 minutes  
- Base folder and logging system  

---

### 🧡 Credits
Developed by [xCaptaiN09](https://github.com/xCaptaiN09)

---

[![GitHub release](https://img.shields.io/github/v/release/xCaptaiN09/Sortify)](https://github.com/xCaptaiN09/Sortify/releases)
