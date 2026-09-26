package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

// Struktur data untuk mapping shortlink
type LinkTarget struct {
	Slug      string
	SafeURL   string // Tujuan jika dari Indonesia / Bot (Misal: vidio.com)
	TargetURL string // Tujuan asli (Landing Page untuk luar negeri)
}

// Database shortlink (bisa disesuaikan atau dihubungkan ke database nanti)
var linksDatabase = map[string]LinkTarget{
	"7SF8h3": {
		Slug:      "7SF8h3",
		SafeURL:   "https://www.vidio.com",
		TargetURL: "https://lead-flow-finder-automation.lovable.app", // Ganti dengan landing page asli Anda
	},
}

func cloakingHandler(w http.ResponseWriter, r *http.Request) {
	// Ambil slug dari URL (contoh: /7SF8h3)
	slug := strings.TrimPrefix(r.URL.Path, "/")
	
	// Cek apakah slug terdaftar
	target, exists := linksDatabase[slug]
	if !exists {
		http.NotFound(w, r)
		return
	}

	// 1. Ambil IP Klien
	clientIP := getClientIP(r)

	// 2. Deteksi Negara (Prioritas ambil dari Cloudflare Header jika nanti di-deploy)
	countryCode := r.Header.Get("CF-IPCountry")

	// Jika tidak pakai Cloudflare (misal testing lokal), baca dari database .mmdb
	if countryCode == "" {
		detectedCountry, err := lookupGeoIP(clientIP)
		if err == nil {
			countryCode = detectedCountry
		} else {
			countryCode = "ID" // Default jika gagal baca IP (dianggap lokal)
		}
	}

	// 3. Logika Cloaking / Redirect
	finalRedirectURL := target.SafeURL // Default aman (ke vidio.com)

	// Jika pengunjung BUKAN dari Indonesia (!= "ID") dan negara terdeteksi
	if countryCode != "" && countryCode != "ID" {
		finalRedirectURL = target.TargetURL
	}

	// Log aktivitas di terminal server
	fmt.Printf("[CLOAKING] IP: %s | Negara: %s | Arah ke: %s\n", clientIP, countryCode, finalRedirectURL)

	// 4. Eksekusi Redirect (HTTP 302 Found)
	http.Redirect(w, r, finalRedirectURL, http.StatusFound)
}

// Helper: Mendapatkan IP asli klien
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

// Helper: Pengecekan GeoIP menggunakan file .mmdb lokal
// Helper: Pengecekan GeoIP menggunakan file .mmdb lokal
func lookupGeoIP(ipStr string) (string, error) {
	// Buka database MaxMind (.mmdb)
	db, err := geoip2.Open("GeoLite2-Country.mmdb")
	if err != nil {
		return "", err
	}
	defer db.Close()

	// (BAGIAN MOCK IP LOKAL DIHAPUS, karena di server nanti IP sudah berupa IP publik asli)

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", fmt.Errorf("invalid IP")
	}

	record, err := db.Country(ip)
	if err != nil {
		return "", err
	}

	return record.Country.IsoCode, nil
}

func main() {
	http.HandleFunc("/", cloakingHandler)

	port := ":8080"
	fmt.Printf("🚀 Cloaking Server (Local MMDB) berjalan di http://localhost:8080...\n")
	log.Fatal(http.ListenAndServe(port, nil))
}