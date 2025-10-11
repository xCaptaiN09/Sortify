# 📁 Sortify
**Author:** [xCaptaiN09](https://github.com/xCaptaiN09)  
**Version:** 1.3  

Sortify is a Magisk / KernelSU module that automatically organizes files in your `/sdcard/Download/` folder into categorized folders — now smarter and safer than ever.

---

### 📦 Features
- 🕒 Runs automatically every 5 minutes  
- 📂 Organizes documents, images, videos, audio, archives, apps, and more  
- 🧠 Skips hidden and incomplete files (`.crdownload`, `.partial`, `.tmp`) to prevent corruption  
- 🪶 Lightweight and fully offline  
- 🧾 Keeps logs at `/sdcard/Sortify/sortify.log` (auto-trimmed)  

---

### 🧩 Installation
1. Flash `Sortify-v1.3.zip` through **Magisk** or **KernelSU**  
2. Reboot  
3. Sortify will automatically organize your downloads ✅  

---

### ⚙️ Configuration
You can customize:
- Organization interval → edit `sleep 300` in `service.sh`  
- Add/remove file extensions inside `service.sh`  

---

### 🧼 Uninstall
Simply remove **Sortify** from **Magisk Manager** and reboot.  
All created folders (`/sdcard/Sortify`) will remain unless you manually delete them.  

---

### 🧾 Changelog

#### v1.3 (2025-10-10)
- 🚫 Prevents moving hidden and incomplete download files (.crdownload, .partial, .tmp)  
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
