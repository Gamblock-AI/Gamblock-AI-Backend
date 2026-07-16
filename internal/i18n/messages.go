// Package i18n holds the end-user-facing message catalog for backend errors.
//
// Single source of truth for friendly Indonesian messages keyed by the stable
// error `code` returned in the API envelope `{ data, error: {code, message},
// request_id }`. Handlers resolve messages through [Friendly] so that:
//   - production responses always show a friendly, non-leaking message;
//   - development responses may surface technical detail for debugging;
//   - server faults retain technical detail, while expected 4xx rejections log
//     only safe request metadata (see Handler.respondErrorErr).
//
// The frontend mirrors these codes in its own catalog (Next.js lib/messages.ts,
// Flutter lib/core/messaging/app_messages.dart). Keep the codes in sync.
package i18n

// Generic fallback when a code has no specific friendly message.
const Generic = "Terjadi kendala, silakan coba beberapa saat lagi."

// messages maps error code -> friendly end-user message (Bahasa Indonesia).
var messages = map[string]string{
	// auth
	"email_required":             "Email wajib diisi.",
	"validation_failed":          "Email dan nama wajib diisi.",
	"invalid_credentials":        "Email atau kata sandi salah. Silakan periksa kembali.",
	"registration_failed":        "Pendaftaran gagal. Email mungkin sudah terdaftar.",
	"dev_login_failed":           "Gagal masuk sebagai pengguna demo.",
	"google_token_required":      "Token Google wajib diisi.",
	"google_verification_failed": "Verifikasi Google gagal. Silakan coba lagi.",
	"refresh_token_required":     "Sesi telah berakhir. Silakan masuk kembali.",
	"invalid_refresh_token":      "Sesi tidak valid. Silakan masuk kembali.",
	"logout_failed":              "Gagal keluar. Silakan coba lagi.",

	// devices
	"device_create_failed":     "Gagal mendaftarkan perangkat.",
	"device_update_failed":     "Gagal memperbarui perangkat.",
	"heartbeat_failed":         "Gagal mengirim sinyal aktif perangkat.",
	"dashboard_summary_failed": "Gagal memuat ringkasan dashboard.",
	"protection_status_failed": "Gagal memuat status perlindungan.",
	"progress_snapshot_failed": "Gagal memuat ringkasan progres.",
	"aggregate_event_rejected": "Agregat perangkat tidak dapat diterima.",
	"profile_not_found":        "Profil tidak ditemukan.",
	"profile_update_failed":    "Gagal memperbarui profil.",

	// partners / accountability
	"partner_email_required":         "Email pendamping wajib diisi.",
	"fetch_partners_failed":          "Gagal memuat data pendamping.",
	"partner_invite_failed":          "Gagal mengirim undangan pendamping.",
	"partner_accept_failed":          "Gagal menerima undangan pendamping.",
	"partner_revoke_failed":          "Gagal memutuskan hubungan pendamping.",
	"fetch_approval_requests_failed": "Gagal memuat daftar permohonan.",
	"action_required":                "Jenis tindakan wajib dipilih.",
	"approval_request_failed":        "Gagal mengajukan permohonan.",
	"approval_cancel_failed":         "Gagal membatalkan permohonan.",
	"approval_approve_failed":        "Gagal menyetujui permohonan.",
	"approval_deny_failed":           "Gagal menolak permohonan.",

	// organizations
	"name_required":        "Nama grup wajib diisi.",
	"create_org_failed":    "Gagal membuat grup.",
	"org_not_found":        "Grup tidak ditemukan.",
	"group_code_required":  "Kode grup wajib diisi.",
	"join_failed":          "Kode grup tidak valid. Coba lagi.",
	"no_org":               "Anda belum bergabung dengan grup mana pun.",
	"list_members_failed":  "Gagal memuat daftar anggota.",
	"analytics_failed":     "Gagal memuat analitik grup.",
	"remove_member_failed": "Gagal mengeluarkan anggota.",

	// missions
	"mission_fetch_failed":  "Gagal memuat misi harian.",
	"invalid_mission":       "Nomor misi harus 1-5.",
	"mission_update_failed": "Gagal memperbarui misi harian.",

	// reflections / psychoeducation
	"fetch_reflections_failed": "Gagal memuat jurnal refleksi.",
	"reflection_create_failed": "Gagal menyimpan jurnal refleksi.",
	"fetch_modules_failed":     "Gagal memuat modul psikoedukasi.",
	"module_not_found":         "Modul tidak ditemukan.",
	"text_required":            "Teks refleksi wajib diisi.",

	// quick approval
	"token_required": "Token validasi wajib diisi.",
	"invalid_token":  "Token tidak valid atau sudah kadaluarsa.",
	"invalid_input":  "Token dan status (approved/denied) wajib diisi.",
	"resolve_failed": "Gagal memproses permohonan.",

	// support / data requests
	"fetch_support_cases_failed": "Gagal memuat tiket bantuan.",
	"support_case_failed":        "Gagal mengirim tiket bantuan.",
	"summary_required":           "Ringkasan tiket wajib diisi.",
	"fetch_data_requests_failed": "Gagal memuat permintaan data.",
	"data_request_failed":        "Gagal mengajukan permintaan data.",
	"type_required":              "Jenis permintaan wajib dipilih.",

	// admin / releases / emergency
	"fetch_admin_modules_failed":        "Gagal memuat modul admin.",
	"fetch_admin_model_releases_failed": "Gagal memuat rilis model.",
	"fetch_admin_support_cases_failed":  "Gagal memuat tiket admin.",
	"create_model_release_failed":       "Gagal merilis model.",
	"create_ruleset_release_failed":     "Gagal merilis ruleset.",
	"create_network_release_failed":     "Gagal merilis ruleset jaringan.",
	"release_not_found":                 "Rilis tidak ditemukan.",
	"release_validation_failed":         "Artefak rilis belum lengkap atau checksum tidak cocok.",
	"artifact_unavailable":              "Artefak rilis belum tersedia atau gagal diverifikasi.",
	"portal_overview_failed":            "Gagal memuat ringkasan operasional.",
	"generate_key_failed":               "Gagal membuat kunci darurat.",
	"emergency_key_required":            "Kunci darurat wajib diisi.",
	"invalid_key":                       "Kunci darurat tidak valid.",
}

// Friendly returns the end-user-facing message for [code], or [Generic] when
// the code is unknown. Always non-empty and never leaks internal details.
func Friendly(code string) string {
	if msg, ok := messages[code]; ok && msg != "" {
		return msg
	}
	return Generic
}
