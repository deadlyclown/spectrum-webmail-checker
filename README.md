🌐 SPECTRUM WEBMAIL CHECKER BY GOLANG 💻
==============================================================================================================

	🚀 Features
	1. Ultra-Optimized Performance: Developed in Go for superior speed and resource efficiency.
	2. Fresh Browser Per Account: Utilizes chromedp to launch a new, clean, and isolated headless browser instance for every account check, minimizing state interference and maximizing reliability.
	3. Ultra-Fast Detection: Implements aggressive detection logic to quickly determine login success or failure.
	4. Real-time Results: Outputs validation results directly to a single file (spectrum_results_<timestamp>.txt) as soon as each account is processed, ensuring data persistence even if the tool is interrupted.
	5. Minimal Overhead: Optimized browser settings (e.g., no image/CSS loading) for faster page interactions.
	6. Clean Output: Suppresses verbose Chromedp debug logs for a cleaner console experience.
	7. Interactive CLI: Prompts for input file, headless mode, and delay settings, making it user-friendly.
	8. Comprehensive Summary: Generates a summary file (spectrum_summary_<timestamp>.txt) at the end of the process, providing a quick overview.


🛠️ Installation Guide
This tool requires Go (Golang) and a Chromium-based browser (Chromium or Google Chrome) installed on your system.
The installation steps vary slightly depending on your operating system.


	1. Install Go (Golang) 🎯
Choose your operating system:

	Arch Linux:
	sudo pacman -S go

==============================================================================================================

	Kali Linux (Debian-based):
	sudo apt update
	sudo apt install golang-go

Note: Kali Linux might have an older Go version. For the latest, consider installing from Go's official website.

	Windows:
	Download the official Go installer (.msi file) from Go's official website.
	Run the installer and follow the instructions. It will typically set up the necessary environment variables for you.
	Verify installation in Command Prompt/PowerShell: go version

==============================================================================================================

	Termux (Android):
	pkg update && pkg upgrade
	pkg install golang

Note: Termux might have an older Go version due to platform limitations.

	2. Install Chromium Browser 🌐
A Chromium-based browser is required for chromedp to function.

	Arch Linux:
	sudo pacman -S chromium
	# Alternatively for Google Chrome (if using AUR helper like yay):
	# yay -S google-chrome-stable

Make sure the chromium binary is accessible in your PATH.

	Kali Linux (Debian-based):
	sudo apt update
	sudo apt install chromium
	# Alternatively for Google Chrome:
	# sudo apt install google-chrome-stable # Requires adding Google Chrome repository first.
	# See: https://www.google.com/linux/chrome/deb/

==============================================================================================================

	Windows:
	Download and install Google Chrome or Chromium. Ensure it's installed to its default location.
	chromedp will typically find Chrome automatically.

==============================================================================================================

	Termux (Android):
	Currently, running chromedp (which requires a full browser like Chromium) directly within Termux on Android is not feasible due to the lack of a full desktop browser environment that chromedp can control.
	This tool is primarily designed for desktop Linux/Windows environments.

==============================================================================================================

	3. Clone the Repository (All OS except Termux) 🧑‍💻
	Navigate to your desired directory in your terminal/command prompt and clone this repository:

	git clone https://github.com/deadlyclown/spectrum-webmail-checker.git
	cd spectrum-webmail-checker

==============================================================================================================

	4. Initialize Go Modules and Install Dependencies 📦
	Once inside the spectrum-webmail-checker directory, run:

	go mod tidy

This command will read the go.mod file, download all necessary Go modules (including chromedp and its cdproto dependencies), and set up your project.

	🚀 How to Use
	Prepare Your Account List 📝:		
	Create a text file (e.g., accounts.txt) in the same directory as the main.go file. Each line should contain an email and password separated by a colon (:).
 
==============================================================================================================

	Example empas.txt:

	user1@cfl.rr.com:password123
	test@tampabay.rr.com:wrongpass
	anotheruser@rr.com:correctpass

 ==============================================================================================================

Run the Tool ▶️:

Execute the tool from your terminal/command prompt. You can run it directly or build an executable.

Run Directly (for testing/development):

	go run main.go

Build an Executable (recommended for repeated use):

	go build -o spectrum_checker main.go
	./spectrum_checker
 
==============================================================================================================

	Input file name: (e.g., empas.txt)
	Run in headless mode? (y/n): (Recommended y for no visible browser window)
	Delay between accounts (seconds): (e.g., 1 to 5 seconds to avoid rate limits)

Example interaction:

	🚀 SPECTRUM EMAIL ACCOUNT CHECKER - ULTRA OPTIMIZED VERSION (GOLANG)
	=================================================================
	🆕 Strategi Instance Browser Baru Per Akun
	⚡ Deteksi Ultra Cepat dengan Penutupan Browser Instan
	🐧 Dioptimalkan untuk Arch Linux
	=================================================================
	🔍 Memeriksa dependensi...
	✅ Chromium ditemukan.
	✅ Koneksi internet terdeteksi.

	✅ Semua dependensi ditemukan!

	📁 Nama file input (default: list.txt): accounts.txt
	👁️  Jalankan dalam mode headless? (y/n, default: y): y
	⏱️  Delay antar akun (detik, default: 1): 2
	📝 Hasil akan disimpan secara real-time ke: spectrum_results_20250622_HHMMSS.txt

	⚙️  KONFIGURASI ULTRA OPTIMIZED:
	   📁 File Input: accounts.txt
	   👁️  Mode Headless: Aktif
	   ⏱️  Delay antar akun: 2 detik
	   🆕 Strategi: Instance browser baru per akun
	   ⚡ Deteksi: Ultra cepat dengan penutupan instan
	   🌐 Target: webmail.spectrum.net

	🚀 Mulai pengecekan ultra optimized? (y/n): y

	🚀 Memulai verifikasi akun ultra optimized...
	🆕 Setiap akun akan mendapatkan instance browser baru yang bersih

	📋 [1/3] Memproses: user1@cfl.rr.com
	🔍 Memeriksa: user1@cfl.rr.com
	🔐 Menutup instance browser...
	❌ INVALID: user1@cfl.rr.com (⏱️ 15.3s)
	   🚫 Alasan: Timeout - Tidak ada respons jelas
	... (dan seterusnya untuk setiap akun)

	📄 Output Files
	The tool generates two primary output files:

	spectrum_results_<timestamp>.txt:

This file contains the real-time results for each account checked, including its status (VALID/INVALID), the reason/message, and individual processing time. This file is updated continuously, ensuring no data loss upon unexpected termination.

	spectrum_summary_<timestamp>.txt:

Provides a summary of the entire checking process, including total accounts processed, number of valid/invalid accounts, success rate, and total/average processing times.

🤝 Contribution
Contributions are welcome! If you have suggestions for improvements, bug fixes, or new features, please open an issue or submit a pull request.

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
