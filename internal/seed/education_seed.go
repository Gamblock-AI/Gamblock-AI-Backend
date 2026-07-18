package seed

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/educationmedia"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/psychoeducationmodule"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

//go:embed assets/education/*.webp
var educationAssets embed.FS

type seedSection struct {
	id, idTitle, enTitle, idBody, enBody, idQuestion, enQuestion string
	idChoices, enChoices                                         []string
	correct                                                      int
}

type seedModule struct {
	id, slug, category, idTitle, enTitle, idSummary, enSummary string
	idObjective, enObjective                                   string
	minutes                                                    int
	mediaIDs                                                   []string
	sections                                                   []seedSection
	sources                                                    []model.EducationSource
}

func richDocument(body string) model.RichTextDocument {
	return model.RichTextDocument{"type": "doc", "content": []any{
		map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": body}}},
	}}
}

func choices(prefix string, values []string) []model.EducationChoice {
	result := make([]model.EducationChoice, 0, len(values))
	for index, value := range values {
		result = append(result, model.EducationChoice{ID: prefix + string(rune('a'+index)), Text: value})
	}
	return result
}

func buildDocument(item seedModule) model.EducationDocument {
	sections := make([]model.EducationSection, 0, len(item.sections))
	for index, section := range item.sections {
		checkID := section.id + "-check"
		choicePrefix := checkID + "-"
		correctID := choicePrefix + string(rune('a'+section.correct))
		sections = append(sections, model.EducationSection{ID: section.id, SortOrder: index, Required: true,
			Translations: map[string]model.EducationSectionTranslation{
				"id": {Title: section.idTitle, Content: richDocument(section.idBody), KnowledgeCheck: &model.EducationKnowledgeCheck{ID: checkID, Question: section.idQuestion, Choices: choices(choicePrefix, section.idChoices), CorrectChoiceID: correctID, Explanation: "Jawaban ini membantu membedakan respons yang melindungi diri dari respons impulsif. Kamu dapat mencoba kembali tanpa penalti.", Required: true}},
				"en": {Title: section.enTitle, Content: richDocument(section.enBody), KnowledgeCheck: &model.EducationKnowledgeCheck{ID: checkID, Question: section.enQuestion, Choices: choices(choicePrefix, section.enChoices), CorrectChoiceID: correctID, Explanation: "This answer separates a protective response from an impulsive one. You can retry without penalty.", Required: true}},
			}})
	}
	thumbnails := make([]model.EducationThumbnail, 0, len(item.mediaIDs))
	for index, mediaID := range item.mediaIDs {
		thumbnails = append(thumbnails, model.EducationThumbnail{MediaID: mediaID, SortOrder: index,
			AltText: map[string]string{"id": "Ilustrasi pendukung modul " + item.idTitle, "en": "Supporting illustration for " + item.enTitle}})
	}
	return model.EducationDocument{
		Category: item.category, EstimatedMinutes: item.minutes,
		ReviewerName: "Tim Konten Gamblock-AI", ReviewerRole: "Tinjauan editorial berbasis sumber resmi",
		ReviewedAt: "2026-07-18",
		Translations: map[string]model.EducationTranslation{
			"id": {Title: item.idTitle, Summary: item.idSummary, LearningObjective: item.idObjective, Disclaimer: "Materi ini bersifat psikoedukasi, bukan diagnosis atau pengganti bantuan profesional.", ReviewerRole: "Tinjauan editorial berbasis sumber resmi"},
			"en": {Title: item.enTitle, Summary: item.enSummary, LearningObjective: item.enObjective, Disclaimer: "This psychoeducation material is not a diagnosis or a substitute for professional support.", ReviewerRole: "Editorial review based on official sources"},
		}, Sections: sections, Thumbnails: thumbnails, Sources: item.sources,
	}
}

func officialSource(title, publisher, rawURL, published string) model.EducationSource {
	return model.EducationSource{Title: title, Publisher: publisher, URL: rawURL, PublishedAt: published, AccessedAt: time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)}
}

func educationFixtures() []seedModule {
	who := officialSource("Gambling", "World Health Organization", "https://www.who.int/news-room/fact-sheets/detail/gambling", "2024-12-02")
	kemkes := officialSource("Tingkat Candu Judi Online Seperti Zat Adiktif", "Kementerian Kesehatan RI", "https://kemkes.go.id/id/tingkat-candu-judi-online-seperti-zat-adiktif", "2024-07-10")
	ppatk := officialSource("Analisis Strategis - Dampak Sosial Judi Online", "PPATK", "https://www.ppatk.go.id/publikasi/read/287/analisis-strategis---dampak-sosial-judi-online.html", "2026-05-04")
	komdigi := officialSource("Edukasi dan Penegakan Hukum: Strategi Pemerintah Berantas Judi Online", "Kementerian Komunikasi dan Digital RI", "https://www.komdigi.go.id/berita/artikel/detail/edukasi-dan-penegakan-hukum-strategi-pemerintah-berantas-judi-online", "2024-06-27")
	ojk := officialSource("OJK, Komdigi, dan Perbankan Perkuat Pemberantasan Scam dan Judol", "Otoritas Jasa Keuangan", "https://www.ojk.go.id/id/berita-dan-kegiatan/siaran-pers/Pages/OJK-Komdigi-dan-Perbankan-Sepakat-Perkuat-Upaya-Pemberantasan-Scam-dan-Judol.aspx", "2026-07-14")
	healing := officialSource("Kenali Layanan Healing119.id", "Kementerian Kesehatan RI", "https://kesprimkom.kemkes.go.id/konten/145/151/0/cegah-bunuh-diri-dukung-kesehatan-jiwa-kenali-layanan-healing119-id", "2025-08-07")
	return []seedModule{
		{id: "mod_impulse_cycle", slug: "memahami-siklus-dorongan", category: "impulse-awareness", idTitle: "Memahami Siklus Dorongan", enTitle: "Understanding the Urge Cycle", idSummary: "Kenali urutan pemicu, pikiran, dorongan, tindakan, dan konsekuensi agar kamu punya ruang untuk memilih.", enSummary: "Recognize triggers, thoughts, urges, actions, and consequences so you have room to choose.", idObjective: "Mengidentifikasi satu titik dalam siklus yang dapat dijeda.", enObjective: "Identify one point in the cycle where you can pause.", minutes: 7, mediaIDs: []string{"med_seed_impulse", "med_seed_pause"}, sources: []model.EducationSource{who, kemkes}, sections: []seedSection{
			{id: "cycle-map", idTitle: "Petakan siklus tanpa menghakimi", enTitle: "Map the cycle without judgment", idBody: "Dorongan tidak muncul dari kelemahan pribadi. Catat pemicu yang kamu sadari sendiri, pikiran yang mengikuti, perubahan tubuh, tindakan, lalu konsekuensinya. Pemetaan ini tidak boleh memakai riwayat browsing otomatis.", enBody: "An urge is not a personal weakness. Note a trigger you recognize yourself, the thought that follows, body changes, action, and consequence. This map must never use automatically captured browsing history.", idQuestion: "Manakah langkah awal yang paling membantu?", enQuestion: "Which first step is most helpful?", idChoices: []string{"Menghakimi diri", "Menamai pemicu dan dorongan", "Mengejar hasil sebelumnya"}, enChoices: []string{"Judge yourself", "Name the trigger and urge", "Chase the previous outcome"}, correct: 1},
			{id: "choice-point", idTitle: "Temukan titik pilihan", enTitle: "Find a choice point", idBody: "Titik pilihan bisa muncul sebelum membuka aplikasi, sebelum melakukan pembayaran, atau ketika tubuh mulai tegang. Pilih satu tindakan kecil seperti berdiri, bernapas, atau berpindah ruangan.", enBody: "A choice point can appear before opening an app, before making a payment, or when your body becomes tense. Choose one small action such as standing up, breathing, or changing rooms.", idQuestion: "Respons mana yang menciptakan ruang untuk memilih?", enQuestion: "Which response creates room to choose?", idChoices: []string{"Jeda dan berpindah tempat", "Menaikkan nominal", "Menyembunyikan masalah"}, enChoices: []string{"Pause and change location", "Increase the amount", "Hide the problem"}, correct: 0},
		}},
		{id: "mod_design_traps", slug: "mengenali-perangkap-desain", category: "digital-literacy", idTitle: "Mengenali Perangkap Desain", enTitle: "Recognizing Design Traps", idSummary: "Pahami bagaimana permainan cepat, notifikasi, dan dorongan mengejar kerugian dapat mempersempit keputusan.", enSummary: "Understand how rapid play, notifications, and loss chasing can narrow decisions.", idObjective: "Mengenali dua pola desain dan memilih batas yang konkret.", enObjective: "Recognize two design patterns and choose a concrete boundary.", minutes: 8, mediaIDs: []string{"med_seed_design", "med_seed_environment"}, sources: []model.EducationSource{who, komdigi}, sections: []seedSection{
			{id: "rapid-loop", idTitle: "Loop cepat dan perhatian", enTitle: "Rapid loops and attention", idBody: "Putaran yang cepat, efek visual, dan notifikasi dapat mengurangi waktu untuk menilai konsekuensi. Beri jeda nyata antara dorongan dan tindakan, bukan sekadar mengandalkan niat.", enBody: "Rapid cycles, visual effects, and notifications can reduce time to evaluate consequences. Create a real delay between urge and action rather than relying on intention alone.", idQuestion: "Apa tujuan jeda pada loop cepat?", enQuestion: "What is the purpose of a pause in a rapid loop?", idChoices: []string{"Menambah taruhan", "Memberi waktu menilai konsekuensi", "Mengejar promosi"}, enChoices: []string{"Increase a stake", "Create time to evaluate consequences", "Chase a promotion"}, correct: 1},
			{id: "loss-chasing", idTitle: "Kerugian bukan utang yang harus dimenangkan kembali", enTitle: "A loss is not a debt to win back", idBody: "Keinginan mengembalikan kerugian dengan permainan berikutnya disebut mengejar kerugian. Hasil acak berikutnya tidak diwajibkan menutup hasil sebelumnya. Hentikan aliran pembayaran dan bicarakan situasinya dengan orang tepercaya.", enBody: "Trying to recover a loss through the next round is called loss chasing. A future random outcome does not owe you a recovery. Stop the payment path and discuss the situation with someone you trust.", idQuestion: "Respons mana yang memutus kejar kerugian?", enQuestion: "Which response interrupts loss chasing?", idChoices: []string{"Menambah deposit", "Menghentikan pembayaran dan meminta dukungan", "Meminjam uang"}, enChoices: []string{"Add another deposit", "Stop payments and ask for support", "Borrow money"}, correct: 1},
		}},
		{id: "mod_ten_minute_pause", slug: "jeda-sepuluh-menit", category: "coping-skill", idTitle: "Jeda Sepuluh Menit", enTitle: "The Ten-Minute Pause", idSummary: "Gunakan napas, perubahan tempat, dan aktivitas singkat untuk melewati puncak dorongan.", enSummary: "Use breathing, a change of place, and a brief activity to move through an urge peak.", idObjective: "Menyusun rutinitas jeda yang bisa dilakukan saat dorongan muncul.", enObjective: "Build a pause routine you can use when an urge appears.", minutes: 6, mediaIDs: []string{"med_seed_pause"}, sources: []model.EducationSource{who, kemkes}, sections: []seedSection{
			{id: "pause-routine", idTitle: "Buat jeda yang terlihat", enTitle: "Make the pause visible", idBody: "Letakkan ponsel, pindah ke tempat yang lebih terbuka, dan atur timer sepuluh menit. Tarik napas dengan ritme nyaman. Tujuannya bukan memaksa perasaan hilang, tetapi menunda tindakan sampai pilihan lain terlihat.", enBody: "Put down the phone, move to a more open space, and set a ten-minute timer. Breathe at a comfortable pace. The goal is not to force the feeling away, but to delay action until other choices become visible.", idQuestion: "Apa tujuan utama jeda?", enQuestion: "What is the main purpose of the pause?", idChoices: []string{"Memaksa emosi hilang", "Menunda tindakan dan membuka pilihan", "Mengisolasi diri"}, enChoices: []string{"Force emotions away", "Delay action and open choices", "Isolate yourself"}, correct: 1},
			{id: "next-action", idTitle: "Pilih tindakan setelah jeda", enTitle: "Choose an action after the pause", idBody: "Sesudah jeda, pilih satu kegiatan yang dapat dimulai dalam dua menit: minum air, berjalan singkat, menghubungi teman, atau membuka tugas kuliah yang kecil.", enBody: "After the pause, choose one activity you can start within two minutes: drink water, take a short walk, contact a friend, or open a small study task.", idQuestion: "Tindakan mana yang paling mudah dimulai?", enQuestion: "Which action is easiest to start?", idChoices: []string{"Tindakan kecil dan konkret", "Rencana sempurna satu bulan", "Kembali ke pemicu"}, enChoices: []string{"A small concrete action", "A perfect one-month plan", "Return to the trigger"}, correct: 0},
		}},
		{id: "mod_protect_essentials", slug: "melindungi-kebutuhan-pokok", category: "financial-literacy", idTitle: "Melindungi Uang Kuliah dan Kebutuhan Pokok", enTitle: "Protecting Tuition and Essentials", idSummary: "Pisahkan kebutuhan pokok dari akses pembayaran berisiko dan lihat kerugian secara realistis.", enSummary: "Separate essential expenses from risky payment access and view losses realistically.", idObjective: "Membuat dua penghalang pembayaran yang dapat diterapkan hari ini.", enObjective: "Create two payment barriers you can apply today.", minutes: 9, mediaIDs: []string{"med_seed_finance"}, sources: []model.EducationSource{ppatk, ojk, who}, sections: []seedSection{
			{id: "essential-first", idTitle: "Kebutuhan pokok dibayar lebih dulu", enTitle: "Pay essentials first", idBody: "Pisahkan biaya kuliah, makan, tempat tinggal, kesehatan, dan transportasi sebelum uang tersedia untuk keputusan impulsif. Gunakan rekening atau jadwal pembayaran yang berbeda bila memungkinkan.", enBody: "Separate tuition, food, housing, health, and transport before money is available for an impulsive decision. Use separate accounts or scheduled payments where possible.", idQuestion: "Dana mana yang diprioritaskan?", enQuestion: "Which funds come first?", idChoices: []string{"Kebutuhan pokok", "Promosi acak", "Deposit berikutnya"}, enChoices: []string{"Essential expenses", "A random promotion", "The next deposit"}, correct: 0},
			{id: "payment-friction", idTitle: "Tambahkan hambatan pembayaran", enTitle: "Add payment friction", idBody: "Hapus metode pembayaran tersimpan, turunkan limit, dan hindari pinjaman untuk menutup kerugian. Jika ada utang, catat jumlahnya dan cari bantuan keuangan yang sah tanpa menambah taruhan.", enBody: "Remove saved payment methods, lower limits, and avoid borrowing to cover losses. If debt exists, record the amount and seek legitimate financial help without adding another bet.", idQuestion: "Hambatan mana yang paling langsung?", enQuestion: "Which barrier is most direct?", idChoices: []string{"Menyimpan semua kartu", "Menghapus metode pembayaran tersimpan", "Meminjam lebih banyak"}, enChoices: []string{"Keep every card saved", "Remove saved payment methods", "Borrow more"}, correct: 1},
		}},
		{id: "mod_change_environment", slug: "mengubah-lingkungan-digital", category: "environment-change", idTitle: "Mengubah Lingkungan Digital", enTitle: "Changing Your Digital Environment", idSummary: "Kurangi notifikasi, jalur pembayaran, dan waktu sendirian dengan pemicu; siapkan aktivitas pengganti.", enSummary: "Reduce notifications, payment paths, and isolated trigger time; prepare replacement activities.", idObjective: "Memilih satu perubahan perangkat dan satu aktivitas pengganti.", enObjective: "Choose one device change and one replacement activity.", minutes: 7, mediaIDs: []string{"med_seed_environment", "med_seed_support"}, sources: []model.EducationSource{komdigi, who}, sections: []seedSection{
			{id: "device-boundary", idTitle: "Kurangi pintu masuk", enTitle: "Reduce entry points", idBody: "Matikan notifikasi yang memicu, hapus bookmark, gunakan perlindungan perangkat, dan jangan menyimpan akses pembayaran. Pemblokiran membantu menciptakan waktu, tetapi tetap perlu dikombinasikan dengan pilihan perilaku.", enBody: "Disable triggering notifications, remove bookmarks, use device protection, and avoid saved payment access. Blocking creates time, but it still needs to be combined with behavioral choices.", idQuestion: "Apa batas perangkat yang konkret?", enQuestion: "What is a concrete device boundary?", idChoices: []string{"Mematikan notifikasi pemicu", "Menambah akun baru", "Menyimpan akses pembayaran"}, enChoices: []string{"Disable triggering notifications", "Create another account", "Save payment access"}, correct: 0},
			{id: "replacement", idTitle: "Siapkan pengganti sebelum dibutuhkan", enTitle: "Prepare an alternative before it is needed", idBody: "Pilih kegiatan rendah biaya yang cocok dengan kondisimu: belajar bersama, berjalan, olahraga ringan, kegiatan kreatif, atau menghubungi teman. Letakkan pengingatnya di tempat yang mudah terlihat.", enBody: "Choose a low-cost activity that fits your situation: study with others, walk, exercise, make something creative, or contact a friend. Keep the reminder where it is easy to see.", idQuestion: "Aktivitas pengganti yang baik bersifat...", enQuestion: "A useful replacement activity is...", idChoices: []string{"Sulit dan mahal", "Mudah dimulai dan tersedia", "Bergantung pada keberuntungan"}, enChoices: []string{"Difficult and expensive", "Easy to start and available", "Dependent on chance"}, correct: 1},
		}},
		{id: "mod_seek_support", slug: "meminta-dukungan", category: "help-seeking", idTitle: "Meminta Dukungan", enTitle: "Asking for Support", idSummary: "Sampaikan kebutuhan dengan konkret, kurangi rasa malu, dan kenali kapan bantuan profesional diperlukan.", enSummary: "State what you need clearly, reduce shame, and recognize when professional support is needed.", idObjective: "Menyiapkan satu pesan bantuan dan satu jalur profesional.", enObjective: "Prepare one support message and one professional pathway.", minutes: 8, mediaIDs: []string{"med_seed_support"}, sources: []model.EducationSource{kemkes, healing, who}, sections: []seedSection{
			{id: "support-message", idTitle: "Gunakan pesan yang singkat dan spesifik", enTitle: "Use a short, specific message", idBody: "Kamu dapat mengatakan: saya sedang mengalami dorongan dan butuh ditemani sepuluh menit. Sebutkan tindakan yang membantu, seperti menelepon, berjalan bersama, atau menjaga akses pembayaran sementara dengan persetujuanmu.", enBody: "You can say: I am experiencing an urge and need company for ten minutes. Name an action that helps, such as calling, walking together, or temporarily guarding payment access with your consent.", idQuestion: "Pesan bantuan yang efektif berisi...", enQuestion: "An effective support message includes...", idChoices: []string{"Kebutuhan yang spesifik", "Penghakiman", "Rahasia tambahan"}, enChoices: []string{"A specific need", "Judgment", "More secrecy"}, correct: 0},
			{id: "professional-help", idTitle: "Bantuan profesional bukan kegagalan", enTitle: "Professional help is not failure", idBody: "Cari bantuan profesional ketika judi mengganggu kuliah, keuangan, relasi, tidur, atau kesehatan mental; ketika upaya berhenti berulang kali tidak berhasil; atau saat kamu merasa tidak aman. Dalam krisis psikologis di Indonesia, layanan resmi Healing119 dapat menjadi salah satu jalur dukungan awal.", enBody: "Seek professional support when gambling disrupts study, finances, relationships, sleep, or mental health; when repeated attempts to stop do not work; or when you feel unsafe. In a psychological crisis in Indonesia, the official Healing119 service can be one initial support pathway.", idQuestion: "Kapan bantuan profesional layak dicari?", enQuestion: "When is professional help appropriate?", idChoices: []string{"Hanya setelah semuanya runtuh", "Saat dampak mulai mengganggu hidup atau keselamatan", "Tidak pernah"}, enChoices: []string{"Only after everything collapses", "When harms begin disrupting life or safety", "Never"}, correct: 1},
		}},
	}
}

var seedMedia = []struct{ id, file string }{
	{"med_seed_impulse", "impulse-cycle.webp"}, {"med_seed_design", "design-traps.webp"},
	{"med_seed_pause", "ten-minute-pause.webp"}, {"med_seed_finance", "protect-essentials.webp"},
	{"med_seed_environment", "change-environment.webp"}, {"med_seed_support", "seek-support.webp"},
}

func InstallEducationAssets(root string) error {
	if root == "" {
		root = "./var/media"
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return err
	}
	for _, item := range seedMedia {
		data, err := educationAssets.ReadFile("assets/education/" + item.file)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(root, item.id+".webp"), data, 0o640); err != nil {
			return err
		}
	}
	return nil
}

func DemoEducationModules(now time.Time) []model.EducationModule {
	items := educationFixtures()
	modules := make([]model.EducationModule, 0, len(items))
	for _, item := range items {
		document := buildDocument(item)
		published := document
		modules = append(modules, model.EducationModule{ID: item.id, Slug: item.slug, Title: item.idTitle,
			Summary: item.idSummary, EstimatedMinutes: item.minutes, Status: "published",
			DraftDocument: document, PublishedDocument: &published, DraftRevision: 1,
			PublishedRevision: 1, PublishedAt: &now, CreatedBy: "seed", UpdatedBy: "seed",
			CreatedAt: now, UpdatedAt: now})
	}
	return modules
}

func DemoEducationMedia(now time.Time) []model.EducationMedia {
	media := make([]model.EducationMedia, 0, len(seedMedia))
	for _, item := range seedMedia {
		data, _ := educationAssets.ReadFile("assets/education/" + item.file)
		hash := sha256.Sum256(data)
		media = append(media, model.EducationMedia{ID: item.id, Kind: "upload", Purpose: "thumbnail",
			MediaType: "image", MIMEType: "image/webp", StorageKey: item.id + ".webp",
			OriginalName: item.file, SizeBytes: int64(len(data)), Width: 1600, Height: 900,
			SHA256: hex.EncodeToString(hash[:]), Status: "published", CreatedBy: "seed", CreatedAt: now, UpdatedAt: now})
	}
	return media
}

func SeedEducationModules(ctx context.Context, client *ent.Client, mediaPath ...string) error {
	root := "./var/media"
	if len(mediaPath) > 0 && mediaPath[0] != "" {
		root = mediaPath[0]
	}
	if err := InstallEducationAssets(root); err != nil {
		return err
	}
	for _, item := range seedMedia {
		data, err := educationAssets.ReadFile("assets/education/" + item.file)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		row, err := client.EducationMedia.Query().Where(educationmedia.IDEQ(item.id)).Only(ctx)
		if ent.IsNotFound(err) {
			_, err = client.EducationMedia.Create().SetID(item.id).SetKind(educationmedia.KindUpload).
				SetPurpose(educationmedia.PurposeThumbnail).SetMediaType(educationmedia.MediaTypeImage).
				SetMimeType("image/webp").SetStorageKey(item.id + ".webp").SetOriginalName(item.file).
				SetSizeBytes(int64(len(data))).SetWidth(1600).SetHeight(900).
				SetSha256(hex.EncodeToString(hash[:])).SetStatus(educationmedia.StatusPublished).SetCreatedBy("seed").Save(ctx)
		} else if err == nil {
			_, err = row.Update().SetStorageKey(item.id + ".webp").SetSizeBytes(int64(len(data))).SetWidth(1600).SetHeight(900).
				SetSha256(hex.EncodeToString(hash[:])).SetStatus(educationmedia.StatusPublished).Save(ctx)
		}
		if err != nil {
			return err
		}
	}
	for _, item := range educationFixtures() {
		document := buildDocument(item)
		row, err := client.PsychoeducationModule.Query().Where(psychoeducationmodule.SlugEQ(item.slug)).Only(ctx)
		if ent.IsNotFound(err) {
			_, err = client.PsychoeducationModule.Create().SetID(item.id).SetSlug(item.slug).
				SetTitle(item.idTitle).SetSummary(item.idSummary).SetBodyMarkdown("").SetEstimatedMinutes(item.minutes).
				SetStatus(psychoeducationmodule.StatusPublished).SetDraftDocumentJSON(document).SetPublishedDocumentJSON(document).
				SetDraftRevision(1).SetPublishedRevision(1).SetPublishedAt(time.Now().UTC()).SetCreatedBy("seed").SetUpdatedBy("seed").Save(ctx)
		} else if err == nil {
			_, err = row.Update().SetTitle(item.idTitle).SetSummary(item.idSummary).SetEstimatedMinutes(item.minutes).
				SetDraftDocumentJSON(document).SetPublishedDocumentJSON(document).SetStatus(psychoeducationmodule.StatusPublished).
				SetPublishedRevision(row.DraftRevision).SetPublishedAt(time.Now().UTC()).SetUpdatedBy("seed").Save(ctx)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
