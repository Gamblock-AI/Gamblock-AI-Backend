package seed

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/downloadapp"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

// DefaultDownloadApps is a checked-in snapshot of the latest stable official
// GitHub release metadata retrieved on 2026-09-14. It deliberately performs no
// network request: administrators control future updates through the API.
func DefaultDownloadApps(now time.Time) []model.DownloadApp {
	return []model.DownloadApp{
		{
			ID: "download_android", Platform: "android", Published: true, UpdatedBy: "seed", Version: "v1.6.16", CreatedAt: now, UpdatedAt: now,
			Eyebrow:      model.LocalizedText{ID: "Aplikasi Android · APK Sideload", EN: "Android App · APK Sideload"},
			Title:        model.LocalizedText{ID: "Gamblock-AI untuk Android", EN: "Gamblock-AI for Android"},
			Description:  model.LocalizedText{ID: "Perlindungan mobile dengan deteksi DOM on-device, animasi Pattern Interrupt, dan ruang pemulihan mandiri.", EN: "Mobile protection featuring on-device DOM detection, Pattern Interrupt animation, and self-recovery modules."},
			Requirements: model.LocalizedText{ID: "Android 8.0+", EN: "Android 8.0+"},
			Architecture: model.LocalizedText{ID: "Universal / ARM64", EN: "Universal / ARM64"},
			Features: []model.LocalizedText{
				{ID: "Deteksi konten halaman lokal (<5 MB model)", EN: "On-device page content analysis (<5 MB model)"},
				{ID: "Jeda intervensi impulsif 5–10 detik", EN: "5–10 second impulsive interruption pause"},
				{ID: "Tanpa perlu akses root pada perangkat", EN: "No root access required on your device"},
				{ID: "Tersedia mode offline untuk alur inti", EN: "Core features work completely offline"},
			},
			Assets: []model.DownloadAsset{{
				ID: "android-apk", Label: model.LocalizedText{ID: "Unduh APK Langsung", EN: "Direct APK Download"},
				FileName:  "gamblock-ai-research-1.6.16+38.apk",
				URL:       "https://github.com/Gamblock-AI/Gamblock-AI-Apps/releases/download/v1.6.16/gamblock-ai-research-1.6.16%2B38.apk",
				SizeBytes: 103620941, SHA256: "0e7faaa7abbfe1a0f6d7ad0ab47c6c535c663d4c37a6e6011a609c96c527227d", Primary: true,
			}},
		},
		{
			ID: "download_windows", Platform: "windows", Published: true, UpdatedBy: "seed", Version: "v1.6.16", CreatedAt: now, UpdatedAt: now,
			Eyebrow:      model.LocalizedText{ID: "Windows Service · Desktop Agent", EN: "Windows Service · Desktop Agent"},
			Title:        model.LocalizedText{ID: "Gamblock-AI untuk Windows", EN: "Gamblock-AI for Windows"},
			Description:  model.LocalizedText{ID: "Layanan latar belakang untuk memfilter akses berisiko di browser desktop tanpa membebani performa PC.", EN: "Background system service to filter risky access across desktop browsers without impacting PC performance."},
			Requirements: model.LocalizedText{ID: "Windows 10 / 11", EN: "Windows 10 / 11"},
			Architecture: model.LocalizedText{ID: "64-bit (x64)", EN: "64-bit (x64)"},
			Features: []model.LocalizedText{
				{ID: "Filter background untuk Chrome & Edge", EN: "Background filtering for Chrome & Edge"},
				{ID: "Penggunaan CPU sangat ringan (<1%)", EN: "Ultra-lightweight CPU footprint (<1%)"},
				{ID: "Status perlindungan di System Tray", EN: "Live protection status in System Tray"},
				{ID: "Bekerja mandiri tanpa ketergantungan server", EN: "Operates independently without cloud dependence"},
			},
			Assets: []model.DownloadAsset{{
				ID: "windows-msi", Label: model.LocalizedText{ID: "Unduh Installer MSI", EN: "Download MSI Installer"},
				FileName:  "GamblockAI-Pilot-1.6.16-x64.msi",
				URL:       "https://github.com/Gamblock-AI/Gamblock-AI-Apps/releases/download/v1.6.16/GamblockAI-Pilot-1.6.16-x64.msi",
				SizeBytes: 20078592, SHA256: "dcf06f7c5d7e7ec123fa9e58cecbd4f9d83578923733e25a0f29d3dac0738387", Primary: true,
			}},
		},
		{
			ID: "download_browser_extension", Platform: "browser_extension", Published: true, UpdatedBy: "seed", Version: "v1.0.1", CreatedAt: now, UpdatedAt: now,
			Eyebrow:      model.LocalizedText{ID: "Ekstensi Browser · MV3", EN: "Browser Extension · MV3"},
			Title:        model.LocalizedText{ID: "Ekstensi Browser Gamblock-AI", EN: "Gamblock-AI Browser Extension"},
			Description:  model.LocalizedText{ID: "Sensor browser pasif yang meneruskan teks halaman yang diizinkan ke Windows Service lokal. Ekstensi ini tidak mengklasifikasi, memblokir, atau mengalihkan halaman.", EN: "A passive browser sensor that relays permitted page text to the local Windows Service. It never classifies, blocks, or redirects pages."},
			Requirements: model.LocalizedText{ID: "Chrome / Edge / Firefox", EN: "Chrome / Edge / Firefox"},
			Architecture: model.LocalizedText{ID: "Manifest V3", EN: "Manifest V3"},
			Features: []model.LocalizedText{
				{ID: "Sensor teks DOM pasif", EN: "Passive DOM text sensing"},
				{ID: "Relay WebSocket lokal terautentikasi", EN: "Authenticated local WebSocket relay"},
				{ID: "Tanpa data penelusuran ke server", EN: "No remote browsing data"},
				{ID: "Terhubung dengan Windows Service", EN: "Works with the Windows Service"},
			},
			Assets: []model.DownloadAsset{
				{ID: "extension-chromium", Label: model.LocalizedText{ID: "ZIP Chrome / Edge", EN: "Chrome / Edge ZIP"}, FileName: "gamblock-ai-extension-chromium-v1.0.1.zip", URL: "https://github.com/Gamblock-AI/Gamblock-AI-Browser-Extention/releases/download/v1.0.1/gamblock-ai-extension-chromium-v1.0.1.zip", SizeBytes: 55638, SHA256: "ef7eb7fbc82a5dea20fd3242e9ce054966db94a04f1de0928316db27e9d0c0c1", Primary: true},
				{ID: "extension-firefox", Label: model.LocalizedText{ID: "ZIP Firefox", EN: "Firefox ZIP"}, FileName: "gamblock-ai-extension-firefox-v1.0.1.zip", URL: "https://github.com/Gamblock-AI/Gamblock-AI-Browser-Extention/releases/download/v1.0.1/gamblock-ai-extension-firefox-v1.0.1.zip", SizeBytes: 55797, SHA256: "7bba89f8f7c608144983f22c31f37d2eeeca4914f58397319180def6b402fdd3"},
			},
		},
	}
}

// SeedDownloadApps inserts only missing platform records, so deployment keeps
// any future release metadata that an administrator has already published.
func SeedDownloadApps(ctx context.Context, client *ent.Client) error {
	for _, item := range DefaultDownloadApps(time.Now().UTC()) {
		exists, err := client.DownloadApp.Query().Where(downloadapp.PlatformEQ(downloadapp.Platform(item.Platform))).Exist(ctx)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := client.DownloadApp.Create().SetID(item.ID).
			SetPlatform(downloadapp.Platform(item.Platform)).SetEyebrowJSON(item.Eyebrow).
			SetTitleJSON(item.Title).SetDescriptionJSON(item.Description).
			SetRequirementsJSON(item.Requirements).SetArchitectureJSON(item.Architecture).
			SetFeaturesJSON(item.Features).SetVersion(item.Version).SetAssetsJSON(item.Assets).
			SetPublished(item.Published).SetUpdatedBy(item.UpdatedBy).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}
