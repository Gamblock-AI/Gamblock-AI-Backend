// Package i18n holds the end-user-facing message catalog for backend errors.
//
// Single source of truth for friendly Indonesian messages keyed by the stable
// error `code` returned in the API envelope `{ data, error: {code, message},
// request_id }`. Handlers resolve messages through [Friendly] so that:
//   - responses always show a friendly, non-leaking message;
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
	// shared request / authorization
	"auth_required":              "Sesi diperlukan. Silakan masuk terlebih dahulu.",
	"forbidden":                  "Anda tidak memiliki izin untuk tindakan ini.",
	"invalid_body":               "Data yang dikirim tidak dapat dibaca. Periksa isian lalu coba lagi.",
	"privacy_payload_rejected":   "Permintaan ditolak karena memuat data yang tidak boleh dikirim.",
	"err_validation":             "Periksa kembali isian yang belum sesuai.",
	"err_internal":               "Terjadi kendala pada layanan. Silakan coba beberapa saat lagi.",
	"create_admin_module_failed": "Modul admin belum dapat dibuat.",

	// auth
	"email_required":                     "Email wajib diisi.",
	"validation_failed":                  "Email dan nama wajib diisi.",
	"invalid_credentials":                "Email atau kata sandi salah. Silakan periksa kembali.",
	"registration_failed":                "Pendaftaran gagal. Email mungkin sudah terdaftar.",
	"dev_login_failed":                   "Gagal masuk sebagai pengguna demo.",
	"password_reset_invalid":             "Kode pemulihan tidak valid, sudah digunakan, atau telah kedaluwarsa.",
	"password_reset_failed":              "Pemulihan kata sandi belum dapat diproses. Silakan coba lagi.",
	"initial_password_change_invalid":    "Tautan penggantian kata sandi awal tidak valid atau telah kedaluwarsa.",
	"phone_verification_required":        "Verifikasi WhatsApp diperlukan untuk melanjutkan.",
	"refresh_token_required":             "Sesi telah berakhir. Silakan masuk kembali.",
	"invalid_refresh_token":              "Sesi tidak valid. Silakan masuk kembali.",
	"logout_failed":                      "Gagal keluar. Silakan coba lagi.",
	"phone_verification_failed":          "Nomor atau kode verifikasi WhatsApp tidak valid.",
	"email_verification_failed":          "Verifikasi kontak telah dipindahkan ke WhatsApp.",
	"email_verification_delivery_failed": "Pengiriman verifikasi telah dipindahkan ke WhatsApp.",
	"phone_required":                     "Nomor WhatsApp wajib diisi.",
	"recent_auth_required":               "Silakan masuk kembali sebelum menyelesaikan keputusan sensitif ini.",

	// devices
	"device_create_failed":        "Gagal mendaftarkan perangkat.",
	"device_id_required":          "Perangkat wajib dipilih.",
	"client_instance_required":    "Identitas instalasi aplikasi tidak tersedia.",
	"device_update_failed":        "Gagal memperbarui perangkat.",
	"heartbeat_failed":            "Gagal mengirim sinyal aktif perangkat.",
	"dashboard_summary_failed":    "Gagal memuat ringkasan dashboard.",
	"protection_status_failed":    "Gagal memuat status perlindungan.",
	"progress_snapshot_failed":    "Gagal memuat ringkasan progres.",
	"aggregate_event_rejected":    "Agregat perangkat tidak dapat diterima.",
	"profile_not_found":           "Profil tidak ditemukan.",
	"profile_update_failed":       "Gagal memperbarui profil.",
	"password_validation_failed":  "Kata sandi saat ini dan kata sandi baru minimal 8 karakter wajib diisi.",
	"current_password_invalid":    "Kata sandi saat ini tidak sesuai.",
	"password_reuse_not_allowed":  "Kata sandi baru harus berbeda dari kata sandi saat ini.",
	"password_update_failed":      "Gagal memperbarui kata sandi.",
	"analytics_period_invalid":    "Periode analitik tidak valid.",
	"protection_analytics_failed": "Gagal memuat analitik perlindungan.",

	// daily reminder & web push
	"reminder_preference_load_failed":   "Preferensi pengingat belum dapat dimuat.",
	"reminder_preference_update_failed": "Preferensi pengingat belum dapat disimpan.",
	"reminder_preference_invalid":       "Waktu atau zona waktu pengingat tidak valid.",
	"push_subscription_invalid":         "Langganan notifikasi tidak valid.",
	"push_subscription_update_failed":   "Langganan notifikasi belum dapat disimpan.",

	// partners / accountability
	"partner_email_required":               "Email pendamping wajib diisi.",
	"fetch_partners_failed":                "Gagal memuat data pendamping.",
	"partner_invite_failed":                "Gagal mengirim undangan pendamping.",
	"partner_accept_failed":                "Gagal menerima undangan pendamping.",
	"partner_revoke_failed":                "Gagal memutuskan hubungan pendamping.",
	"fetch_approval_requests_failed":       "Gagal memuat daftar permohonan.",
	"action_required":                      "Jenis tindakan wajib dipilih.",
	"approval_request_failed":              "Gagal mengajukan permohonan.",
	"approval_cancel_failed":               "Gagal membatalkan permohonan.",
	"approval_approve_failed":              "Gagal menyetujui permohonan.",
	"approval_deny_failed":                 "Gagal menolak permohonan.",
	"approval_apply_failed":                "Persetujuan tidak dapat diterapkan atau masa penerapannya telah berakhir.",
	"accountability_workspace_failed":      "Ruang akuntabilitas tidak dapat dimuat untuk akun ini.",
	"accountability_group_create_failed":   "Grup belum dapat dibuat. Pastikan email dan WhatsApp pendamping telah terverifikasi.",
	"accountability_code_invalid":          "Kode grup tidak valid atau sudah diganti.",
	"accountability_join_failed":           "Belum dapat bergabung ke grup. Periksa konfirmasi dan keanggotaan aktif Anda.",
	"accountability_code_rotate_failed":    "Kode grup belum dapat diganti.",
	"accountability_group_delete_failed":   "Grup hanya dapat dihapus setelah tidak memiliki anggota aktif.",
	"accountability_sharing_update_failed": "Preferensi berbagi belum dapat diperbarui.",
	"accountability_leave_failed":          "Permintaan keluar belum dapat diproses.",
	"accountability_leave_cancel_failed":   "Permintaan keluar tidak dapat dibatalkan. Muat ulang status lalu coba lagi.",
	"accountability_leave_resolve_failed":  "Keputusan keluar belum dapat disimpan.",
	"accountability_member_remove_failed":  "Anggota belum dapat dikeluarkan dari grup.",
	"partner_contact_create_failed":        "Permintaan menghubungi pendamping belum dapat dikirim.",
	"partner_contact_transition_failed":    "Status permintaan kontak belum dapat diperbarui.",

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
	"mission_fetch_failed":        "Gagal memuat misi harian.",
	"invalid_mission":             "Misi tidak valid.",
	"mission_update_failed":       "Gagal memperbarui misi harian.",
	"custom_mission_limit":        "Maksimal lima misi custom per hari.",
	"custom_mission_invalid":      "Isi misi custom belum valid.",
	"custom_mission_not_editable": "Misi custom yang sudah selesai tidak dapat diubah.",

	// learning hub
	"learning_hub_fetch_failed":            "Gagal memuat Learning Hub.",
	"learning_hub_progress_failed":         "Gagal memuat progres Learning Hub.",
	"learning_hub_state_invalid":           "Status belajar belum valid.",
	"learning_hub_checkpoint_invalid":      "Checkpoint belajar membutuhkan refleksi atau hasil proyek.",
	"learning_hub_item_not_found":          "Materi belajar tidak ditemukan.",
	"learning_hub_mutation_failed":         "Progres Learning Hub belum dapat disimpan.",
	"learning_hub_admin_failed":            "CMS Learning Hub belum dapat diproses.",
	"learning_hub_admin_not_found":         "Materi atau taxonomy Learning Hub tidak ditemukan.",
	"learning_hub_admin_validation_failed": "Periksa kembali metadata dan isian Learning Hub.",
	"learning_hub_admin_conflict":          "Draf Learning Hub berubah di sesi lain. Muat ulang sebelum menyimpan.",
	"learning_hub_taxonomy_conflict":       "Taxonomy masih digunakan oleh materi atau program lain.",

	// reflections / psychoeducation
	"fetch_reflections_failed":    "Gagal memuat jurnal refleksi.",
	"reflection_create_failed":    "Gagal menyimpan jurnal refleksi.",
	"reflection_update_failed":    "Jurnal refleksi belum dapat diperbarui.",
	"fetch_modules_failed":        "Gagal memuat modul psikoedukasi.",
	"module_not_found":            "Modul tidak ditemukan.",
	"education_conflict":          "Draf berubah di sesi lain. Muat ulang sebelum melanjutkan.",
	"education_validation_failed": "Modul belum memenuhi syarat untuk disimpan atau diterbitkan.",
	"education_media_invalid":     "Media tidak didukung, terlalu besar, atau sumbernya tidak diizinkan.",
	"education_media_not_found":   "Media pembelajaran tidak ditemukan.",
	"text_required":               "Teks refleksi wajib diisi.",

	// quick approval
	"token_required": "Token validasi wajib diisi.",
	"invalid_token":  "Token tidak valid atau sudah kadaluarsa.",
	"invalid_input":  "Token dan status (approved/denied) wajib diisi.",
	"resolve_failed": "Gagal memproses permohonan.",

	// support / data requests
	"fetch_support_cases_failed":     "Gagal memuat tiket bantuan.",
	"support_case_failed":            "Gagal mengirim tiket bantuan.",
	"summary_required":               "Ringkasan tiket wajib diisi.",
	"fetch_data_requests_failed":     "Gagal memuat permintaan data.",
	"data_request_failed":            "Gagal mengajukan permintaan data.",
	"type_required":                  "Jenis permintaan wajib dipilih.",
	"support_case_not_found":         "Tiket bantuan tidak ditemukan atau bukan milik Anda.",
	"support_reply_failed":           "Balasan belum dapat dikirim.",
	"support_transition_failed":      "Status tiket belum dapat diperbarui.",
	"support_claim_failed":           "Tiket belum dapat diambil. Muat ulang antrean dan coba lagi.",
	"support_release_failed":         "Tiket belum dapat dilepas dari antrean Anda.",
	"recovery_records_failed":        "Data pemulihan belum dapat dimuat.",
	"recovery_record_save_failed":    "Data pemulihan belum dapat disimpan.",
	"recovery_practice_fetch_failed": "Latihan pemulihan belum dapat dimuat.",
	"recovery_space_fetch_failed":    "Ruang pulih belum dapat dimuat.",
	"weekly_review_fetch_failed":     "Tinjauan mingguan belum dapat dimuat.",
	"weekly_review_save_failed":      "Tinjauan mingguan belum dapat disimpan.",
	"data_request_retry_failed":      "Permintaan data belum dapat diproses ulang.",
	"data_request_reject_failed":     "Permintaan data belum dapat ditolak.",
	"data_export_unavailable":        "Arsip data tidak tersedia atau masa unduhnya telah berakhir.",
	"account_deletion_failed":        "Penghapusan akun belum dapat dikonfirmasi.",

	// admin / emergency
	"fetch_admin_modules_failed":        "Gagal memuat modul admin.",
	"fetch_admin_support_cases_failed":  "Gagal memuat tiket admin.",
	"admin_overview_failed":             "Gagal memuat ringkasan operasional.",
	"site_social_links_failed":          "Pengaturan sosial media belum dapat diproses.",
	"audit_events_failed":               "Gagal memuat jejak audit.",
	"operators_fetch_failed":            "Gagal memuat akun operator.",
	"admin_accounts_fetch_failed":       "Gagal memuat daftar akun.",
	"admin_account_create_failed":       "Akun belum dapat dibuat.",
	"admin_account_update_failed":       "Status akun belum dapat diperbarui.",
	"operator_invitation_retired":       "Undangan operator sudah tidak digunakan. Hubungi admin untuk pembuatan akun.",
	"operator_invite_failed":            "Undangan operator belum dapat dikirim.",
	"operator_update_failed":            "Akun operator belum dapat diperbarui.",
	"operator_invitation_revoke_failed": "Undangan operator belum dapat dicabut.",
	"operator_invitation_invalid":       "Undangan operator tidak valid atau telah kedaluwarsa.",
	"operator_invitation_accept_failed": "Akun operator belum dapat diaktifkan.",
	"portal_overview_failed":            "Gagal memuat ringkasan operasional.",
	"generate_key_failed":               "Gagal membuat kunci darurat.",
	"emergency_request_failed":          "Gagal meminta pemulihan darurat.",
	"emergency_request_not_found":       "Permintaan pemulihan darurat tidak ditemukan.",
	"emergency_review_failed":           "Peninjauan pemulihan darurat gagal.",
	"emergency_key_required":            "Kunci darurat wajib diisi.",
	"invalid_key":                       "Kunci darurat tidak valid.",

	// translation (DeepSeek)
	"translation_failed":         "Gagal menerjemahkan konten. Silakan coba lagi.",
	"translation_invalid_input":  "Input translasi tidak valid.",
	"translation_unavailable":    "Layanan AI penerjemahan sedang tidak tersedia.",
	"translation_rate_limited":   "Penerjemahan AI sedang sibuk, coba lagi dalam beberapa saat.",

	// SPK decision support
	"spk_recommendation_failed":     "Rekomendasi harian belum dapat dimuat.",
	"spk_intervention_not_found":    "Rekomendasi tidak ditemukan atau bukan milik Anda.",
	"spk_intervention_complete_failed": "Rekomendasi belum dapat ditandai selesai.",
	"blocked_events_rejected":       "Data waktu blokir perangkat tidak dapat diterima.",
	"spk_preference_invalid":        "Preferensi belum valid.",
}

// Friendly returns the end-user-facing message for [code], or [Generic] when
// the code is unknown. Always non-empty and never leaks internal details.
func Friendly(code string) string {
	if msg, ok := messages[code]; ok && msg != "" {
		return msg
	}
	return Generic
}
