package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/academicprogram"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/institution"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningcluster"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningitem"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningrevision"
)

type learningSeedCluster struct {
	slug, titleID, titleEN, descriptionID, descriptionEN string
	programs                                             []struct{ slug, name, degree string }
	resources                                            []struct{ kind, slug, titleID, titleEN, provider, url string }
}

type LearningHubSeedReport struct {
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
}

// Keep the first published revision self-contained. This lets the student
// catalog continue serving the last published title/summary when an editor
// later changes the draft fields.
func learningRevisionSnapshot(document map[string]any, item learningSeedItem) map[string]any {
	return map[string]any{
		"_learning_item_snapshot": map[string]any{
			"document":   document,
			"slug":       item.slug,
			"kind":       item.kind,
			"title_id":   item.titleID,
			"title_en":   item.titleEN,
			"summary_id": item.summaryID,
			"summary_en": item.summaryEN,
		},
	}
}

var learningSeedClusters = []learningSeedCluster{
	{
		slug: "computing-data", titleID: "Komputasi & Data", titleEN: "Computing & Data",
		descriptionID: "Bangun fondasi digital, data, aplikasi, dan keamanan yang dapat dipakai lintas bidang.",
		descriptionEN: "Build practical foundations in digital work, data, applications, and security.",
		programs: []struct{ slug, name, degree string }{
			{"uty-d3-sistem-informasi", "Sistem Informasi", "D3"}, {"uty-informatika", "Informatika", "S1"},
			{"uty-informatika-medis", "Informatika Medis", "S1"}, {"uty-sistem-informasi", "Sistem Informasi", "S1"},
			{"uty-teknik-komputer", "Teknik Komputer", "S1"}, {"uty-sains-data", "Sains Data", "S1"},
		},
		resources: []struct{ kind, slug, titleID, titleEN, provider, url string }{
			{"course", "microsoft-power-platform-fundamentals", "Dasar Microsoft Power Platform", "Microsoft Power Platform Fundamentals", "Microsoft Learn", "https://learn.microsoft.com/en-us/training/paths/pl-900-power-platform-fundamentals/"},
			{"course", "microsoft-data-analytics", "Memulai Analitik Data Microsoft", "Get Started with Microsoft Data Analytics", "Microsoft Learn", "https://learn.microsoft.com/en-us/training/paths/data-analytics-microsoft/"},
			{"course", "cisco-cybersecurity-basics", "Pengantar Keamanan Siber", "Introduction to Cybersecurity", "Cisco Networking Academy", "https://www.cisco.com/site/us/en/learn/training-certifications/training/netacad/index.html"},
			{"course", "freecodecamp-web-foundations", "Fondasi Web Praktis", "Practical Web Foundations", "freeCodeCamp", "https://www.freecodecamp.org/learn/"},
			{"certification", "azure-data-fundamentals", "Persiapan Azure Data Fundamentals", "Azure Data Fundamentals Preparation", "Microsoft Credentials", "https://learn.microsoft.com/en-us/credentials/certifications/azure-data-fundamentals/"},
			{"toolkit", "computing-data-toolkit", "Toolkit Portofolio Data dan Aplikasi", "Data and Application Portfolio Toolkit", "Gamblock-AI", "https://learn.microsoft.com/en-us/training/"},
			{"career_snapshot", "computing-data-careers", "Peta Karier Komputasi dan Data", "Computing and Data Career Map", "Gamblock-AI", "https://learn.microsoft.com/en-us/training/career-paths/"},
			{"course", "dicoding-basic-web", "Dasar Pemrograman Web", "Web Programming Basics", "Dicoding", "https://www.dicoding.com/academies/261"},
			{"course", "dicoding-html-css", "Belajar Dasar HTML dan CSS", "HTML & CSS Basics", "Dicoding", "https://www.dicoding.com/academies/153"},
			{"course", "dicoding-javascript", "Belajar JavaScript Dasar", "JavaScript Fundamentals", "Dicoding", "https://www.dicoding.com/academies/256"},
			{"course", "dicoding-flutter", "Belajar Flutter untuk Pemula", "Flutter for Beginners", "Dicoding", "https://www.dicoding.com/academies/159"},
			{"course", "dicoding-android", "Membuat Aplikasi Android untuk Pemula", "Android Apps for Beginners", "Dicoding", "https://www.dicoding.com/academies/84"},
			{"course", "dicoding-machine-learning", "Memulai Machine Learning", "Getting Started with Machine Learning", "Dicoding", "https://www.dicoding.com/academies/257"},
			{"course", "dicoding-github", "Kolaborasi dengan Git dan GitHub", "Collaborating with Git & GitHub", "Dicoding", "https://www.dicoding.com/academies/278"},
			{"course", "dicoding-aws-cloud", "Belajar Dasar AWS Cloud", "AWS Cloud Fundamentals", "Dicoding", "https://www.dicoding.com/academies/173"},
		},
	},
	{
		slug: "engineering-built-environment", titleID: "Teknik, Desain & Lingkungan Binaan", titleEN: "Engineering, Design & Built Environment",
		descriptionID: "Hubungkan teori teknik dengan pemetaan, desain, dokumentasi, dan proyek yang dapat dipamerkan.",
		descriptionEN: "Connect engineering theory with mapping, design, documentation, and portfolio-ready projects.",
		programs: []struct{ slug, name, degree string }{
			{"uty-arsitektur", "Arsitektur", "S1"}, {"uty-teknik-sipil", "Teknik Sipil", "S1"},
			{"uty-teknik-elektro", "Teknik Elektro", "S1"}, {"uty-teknik-industri", "Teknik Industri", "S1"},
			{"uty-pwk", "Perencanaan Wilayah dan Kota", "S1"},
		},
		resources: []struct{ kind, slug, titleID, titleEN, provider, url string }{
			{"course", "autodesk-revit-foundations", "Fondasi Pemodelan Revit", "Revit Modeling Foundations", "Autodesk Learning", "https://www.autodesk.com/learning.autodesk"},
			{"course", "autodesk-autocad-design", "Menggambar Teknis dengan AutoCAD", "Technical Drawing with AutoCAD", "Autodesk Learning", "https://www.autodesk.com/learning.autodesk"},
			{"course", "qgis-basic-map", "Membuat Peta Dasar dengan QGIS", "Creating a Basic Map with QGIS", "QGIS Documentation", "https://docs.qgis.org/3.44/en/docs/training_manual/index.html"},
			{"course", "qgis-vector-data", "Mengolah Data Vektor", "Working with Vector Data", "QGIS Documentation", "https://docs.qgis.org/3.44/en/docs/training_manual/index.html"},
			{"certification", "engineering-digital-portfolio", "Portofolio Digital untuk Insinyur dan Desainer", "Digital Portfolio for Engineers and Designers", "Gamblock-AI", "https://www.autodesk.com/learning.autodesk"},
			{"toolkit", "built-environment-toolkit", "Toolkit Pemetaan dan Dokumentasi Proyek", "Mapping and Project Documentation Toolkit", "Gamblock-AI", "https://docs.qgis.org/3.44/en/docs/training_manual/index.html"},
			{"career_snapshot", "built-environment-careers", "Peta Karier Teknik dan Desain", "Engineering and Design Career Map", "Gamblock-AI", "https://www.autodesk.com/learning.autodesk"},
		},
	},
	{
		slug: "business-finance-tourism", titleID: "Bisnis, Keuangan & Pariwisata", titleEN: "Business, Finance & Tourism",
		descriptionID: "Ubah ide menjadi keputusan bisnis, pengelolaan uang, pemasaran, dan pengalaman wisata yang bertanggung jawab.",
		descriptionEN: "Turn ideas into practical decisions in business, money, marketing, and responsible tourism.",
		programs: []struct{ slug, name, degree string }{
			{"uty-d3-akuntansi", "Akuntansi", "D3"}, {"uty-s1-akuntansi", "Akuntansi", "S1"},
			{"uty-manajemen", "Manajemen", "S1"}, {"uty-destinasi-pariwisata", "Destinasi Pariwisata", "D4"},
		},
		resources: []struct{ kind, slug, titleID, titleEN, provider, url string }{
			{"course", "hubspot-digital-marketing", "Sertifikasi Digital Marketing", "Digital Marketing Certification", "HubSpot Academy", "https://academy.hubspot.com/courses/digital-marketing"},
			{"course", "hubspot-content-marketing", "Strategi Content Marketing", "Content Marketing Strategy", "HubSpot Academy", "https://academy.hubspot.com/courses/content-marketing"},
			{"course", "openlearn-bookkeeping", "Pengantar Bookkeeping dan Akuntansi", "Introduction to Bookkeeping and Accounting", "OpenLearn", "https://www.open.edu/openlearn/money-business/introduction-bookkeeping-and-accounting"},
			{"course", "openlearn-entrepreneurship", "Dari Ide ke Usaha", "Entrepreneurship: From Ideas to Reality", "OpenLearn", "https://www.open.edu/openlearn/money-business/entrepreneurship-ideas-reality"},
			{"certification", "hubspot-social-media", "Sertifikasi Social Media Marketing", "Social Media Marketing Certification", "HubSpot Academy", "https://academy.hubspot.com/courses/social-media"},
			{"toolkit", "business-tourism-toolkit", "Toolkit Riset Pasar dan Rencana Usaha", "Market Research and Business Plan Toolkit", "Gamblock-AI", "https://academy.hubspot.com/"},
			{"career_snapshot", "business-tourism-careers", "Peta Karier Bisnis dan Pariwisata", "Business and Tourism Career Map", "Gamblock-AI", "https://www.open.edu/openlearn/free-courses"},
		},
	},
	{
		slug: "psychology-education", titleID: "Psikologi & Pendidikan", titleEN: "Psychology & Education",
		descriptionID: "Pelajari perilaku, kesejahteraan, desain pembelajaran, dan komunikasi yang membantu orang berkembang.",
		descriptionEN: "Explore behaviour, wellbeing, learning design, and communication that help people grow.",
		programs: []struct{ slug, name, degree string }{
			{"uty-psikologi", "Psikologi", "S1"}, {"uty-bimbingan-konseling", "Bimbingan dan Konseling", "S1"},
			{"uty-pendidikan-bahasa-inggris", "Pendidikan Bahasa Inggris", "S1"}, {"uty-pendidikan-teknologi-informasi", "Pendidikan Teknologi Informasi", "S1"},
		},
		resources: []struct{ kind, slug, titleID, titleEN, provider, url string }{
			{"course", "openlearn-starting-psychology", "Memulai Studi Psikologi", "Starting with Psychology", "OpenLearn", "https://www.open.edu/openlearn/health-sports-psychology/psychology/starting-psychology/content-section-0"},
			{"course", "openlearn-adolescent-mental-health", "Pengantar Kesehatan Mental Remaja", "Introduction to Adolescent Mental Health", "OpenLearn", "https://www.open.edu/openlearn/health-sports-psychology/introduction-adolescent-mental-health"},
			{"course", "openlearn-learning-to-teach", "Belajar Menjadi Pengajar", "Learning to Teach", "OpenLearn", "https://www.open.edu/openlearn/education-development/learning-teach-making-sense-learning-teach/content-section-0"},
			{"course", "british-council-teachingenglish", "Sumber Daya TeachingEnglish", "TeachingEnglish Resources", "British Council", "https://www.teachingenglish.org.uk/"},
			{"certification", "openlearn-reflective-practice", "Menjadi Praktisi Reflektif", "Becoming a Reflective Practitioner", "OpenLearn", "https://www.open.edu/openlearn/education-development/learning-teach-becoming-reflective-practitioner/content-section-0"},
			{"toolkit", "psychology-education-toolkit", "Toolkit Observasi dan Refleksi", "Observation and Reflection Toolkit", "Gamblock-AI", "https://www.open.edu/openlearn/free-courses"},
			{"career_snapshot", "psychology-education-careers", "Peta Karier Psikologi dan Pendidikan", "Psychology and Education Career Map", "Gamblock-AI", "https://www.open.edu/openlearn/free-courses"},
		},
	},
	{
		slug: "communication-language-global", titleID: "Komunikasi, Bahasa & Global", titleEN: "Communication, Language & Global",
		descriptionID: "Bangun kemampuan menulis, berbicara, membuat konten, dan memahami konteks lintas budaya.",
		descriptionEN: "Build writing, speaking, content, and cross-cultural skills for public-facing work.",
		programs: []struct{ slug, name, degree string }{
			{"uty-ilmu-komunikasi", "Ilmu Komunikasi", "S1"}, {"uty-sastra-inggris", "Sastra Inggris", "S1"},
			{"uty-ilmu-hubungan-internasional", "Ilmu Hubungan Internasional", "S1"},
		},
		resources: []struct{ kind, slug, titleID, titleEN, provider, url string }{
			{"course", "british-council-business-english", "Business English", "Business English", "British Council", "https://learnenglish.britishcouncil.org/business-english"},
			{"course", "british-council-skills", "Keterampilan Bahasa Inggris", "English Skills", "British Council", "https://learnenglish.britishcouncil.org/skills"},
			{"course", "hubspot-video-marketing", "Strategi Video Marketing", "Video Marketing Strategy", "HubSpot Academy", "https://academy.hubspot.com/courses/video-marketing"},
			{"course", "hubspot-youtube-marketing", "Membangun Kanal YouTube", "Growing a YouTube Channel", "HubSpot Academy", "https://academy.hubspot.com/courses/youtube-marketing"},
			{"certification", "hubspot-content-creator", "Sertifikasi Content Marketing", "Content Marketing Certification", "HubSpot Academy", "https://academy.hubspot.com/courses/content-marketing"},
			{"toolkit", "communication-global-toolkit", "Toolkit Storytelling dan Presentasi", "Storytelling and Presentation Toolkit", "Gamblock-AI", "https://learnenglish.britishcouncil.org/"},
			{"career_snapshot", "communication-global-careers", "Peta Karier Komunikasi dan Global", "Communication and Global Career Map", "Gamblock-AI", "https://learnenglish.britishcouncil.org/"},
		},
	},
}

func SeedLearningHubDefaults(ctx context.Context, client *ent.Client) error {
	_, err := SeedLearningHubDefaultsWithReport(ctx, client)
	return err
}

func SeedLearningHubDefaultsWithReport(ctx context.Context, client *ent.Client, mediaPath ...string) (LearningHubSeedReport, error) {
	root := "./var/media"
	if len(mediaPath) > 0 && mediaPath[0] != "" {
		root = mediaPath[0]
	}
	before, err := learningHubRecordCount(ctx, client)
	if err != nil {
		return LearningHubSeedReport{}, err
	}
	tx, err := client.Tx(ctx)
	if err != nil {
		return LearningHubSeedReport{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if err := seedLearningHubWithClient(ctx, tx.Client(), root); err != nil {
		return LearningHubSeedReport{}, err
	}
	after, err := learningHubRecordCount(ctx, tx.Client())
	if err != nil {
		return LearningHubSeedReport{}, err
	}
	if err := tx.Commit(); err != nil {
		return LearningHubSeedReport{}, err
	}
	committed = true
	inserted := after - before
	if inserted < 0 {
		inserted = 0
	}
	expected := learningHubExpectedRecordCount()
	skipped := expected - inserted
	if skipped < 0 {
		skipped = 0
	}
	return LearningHubSeedReport{Inserted: inserted, Skipped: skipped}, nil
}

func learningHubRecordCount(ctx context.Context, client *ent.Client) (int, error) {
	counts := []func(context.Context) (int, error){
		client.Institution.Query().Count,
		client.LearningCluster.Query().Count,
		client.AcademicProgram.Query().Count,
		client.LearningItem.Query().Count,
		client.LearningRevision.Query().Count,
	}
	total := 0
	for _, count := range counts {
		value, err := count(ctx)
		if err != nil {
			return 0, err
		}
		total += value
	}
	return total, nil
}

func learningHubExpectedRecordCount() int {
	programs, items := 0, 0
	for _, cluster := range learningSeedClusters {
		programs += len(cluster.programs)
		items += len(cluster.resources) + 3 // one path and two mini-projects
	}
	return 1 + len(learningSeedClusters) + programs + items + items
}

func seedLearningHubWithClient(ctx context.Context, client *ent.Client, mediaPath string) error {
	if err := seedLearningHubAssets(ctx, client, mediaPath); err != nil {
		return err
	}
	uty, err := client.Institution.Query().Where(institution.SlugEQ("uty")).Only(ctx)
	if ent.IsNotFound(err) {
		uty, err = client.Institution.Create().SetID("inst_uty").SetSlug("uty").SetName("Universitas Teknologi Yogyakarta").SetStatus(institution.StatusActive).Save(ctx)
	}
	if err != nil {
		return err
	}
	reviewedAt := time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC)
	for clusterIndex, cluster := range learningSeedClusters {
		if err := seedLearningCluster(ctx, client, cluster, clusterIndex); err != nil {
			return err
		}
		for programIndex, program := range cluster.programs {
			if err := seedLearningProgram(ctx, client, uty.ID, program, cluster.slug, clusterIndex*10+programIndex); err != nil {
				return err
			}
		}
		for resourceIndex, resource := range cluster.resources {
			if err := seedLearningItem(ctx, client, learningSeedItem{resource.kind, resource.slug, resource.titleID, resource.titleEN, resource.titleID, resource.titleEN, resource.provider, resource.url, cluster.slug, resourceIndex, nil}, reviewedAt, mediaPath); err != nil {
				return err
			}
		}
		path := learningSeedItem{"learning_path", cluster.slug + "-path", "Jalur Belajar " + cluster.titleID, cluster.titleEN + " Learning Path", "Susunan langkah pendek untuk mencoba bidang ini dengan proyek nyata.", "A short sequence for trying this field through a practical project.", "Gamblock-AI", "https://uty.ac.id/", cluster.slug, 90, nil}
		path.stepSlugs = []string{cluster.resources[0].slug, cluster.resources[1].slug, cluster.slug + "-mini-project-1"}
		if err := seedLearningItem(ctx, client, path, reviewedAt, mediaPath); err != nil {
			return err
		}
		for projectIndex := 1; projectIndex <= 2; projectIndex++ {
			project := learningSeedItem{"mini_project", fmt.Sprintf("%s-mini-project-%d", cluster.slug, projectIndex), fmt.Sprintf("Mini-project %s %d", cluster.titleID, projectIndex), fmt.Sprintf("%s Mini-project %d", cluster.titleEN, projectIndex), "Hasil kecil yang dapat dipamerkan, bukan tugas yang harus sempurna.", "A small portfolio-ready outcome, not a perfection test.", "Gamblock-AI", "https://uty.ac.id/", cluster.slug, 100 + projectIndex, nil}
			if err := seedLearningItem(ctx, client, project, reviewedAt, mediaPath); err != nil {
				return err
			}
		}
	}
	return fillLearningHubMedia(ctx, client)
}

type learningSeedItem struct {
	kind, slug, titleID, titleEN, summaryID, summaryEN, provider, url, cluster string
	order                                                                      int
	stepSlugs                                                                  []string
}

func seedLearningCluster(ctx context.Context, client *ent.Client, cluster learningSeedCluster, sortOrder int) error {
	if _, err := client.LearningCluster.Query().Where(learningcluster.SlugEQ(cluster.slug)).Only(ctx); err == nil {
		return nil
	} else if !ent.IsNotFound(err) {
		return err
	}
	_, err := client.LearningCluster.Create().SetID("cluster_" + cluster.slug).SetSlug(cluster.slug).SetTitleID(cluster.titleID).SetTitleEn(cluster.titleEN).SetDescriptionID(cluster.descriptionID).SetDescriptionEn(cluster.descriptionEN).SetSortOrder(sortOrder).SetActive(true).Save(ctx)
	return err
}

func seedLearningProgram(ctx context.Context, client *ent.Client, institutionID string, program struct{ slug, name, degree string }, cluster string, sortOrder int) error {
	if _, err := client.AcademicProgram.Query().Where(academicprogram.SlugEQ(program.slug)).Only(ctx); err == nil {
		return nil
	} else if !ent.IsNotFound(err) {
		return err
	}
	_, err := client.AcademicProgram.Create().SetID("program_" + program.slug).SetInstitutionID(institutionID).SetSlug(program.slug).SetName(program.name).SetDegree(program.degree).SetPrimaryClusterSlug(cluster).SetSortOrder(sortOrder).SetActive(true).Save(ctx)
	return err
}

func seedLearningItem(ctx context.Context, client *ent.Client, item learningSeedItem, reviewedAt time.Time, mediaPath string) error {
	existing, err := client.LearningItem.Query().Where(learningitem.SlugEQ(item.slug)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	if existing != nil {
		return nil
	}
	cost := "free_or_audit"
	if item.kind == "certification" {
		cost = "learning_free_exam_may_cost"
	}
	document := map[string]any{
		"provider": item.provider, "url": item.url, "cost": cost,
		"certificate": "provider_dependent", "language": []string{"id", "en"},
		"difficulty": "beginner", "duration_minutes": learningDuration(item.kind, item.order),
		"clusters": []string{item.cluster}, "programs": learningProgramSlugs(item.cluster),
		"outcomes_id": []string{"Memahami konsep dasar dan menentukan satu langkah praktik berikutnya."},
		"outcomes_en": []string{"Understand the foundation and choose one practical next step."},
		"reviewed_at": reviewedAt.Format("2006-01-02"), "reviewer_name": "Gamblock-AI baseline curation",
		"duration_basis": "suggested_starter_session",
	}
	if mediaPath != "" {
		if logoID := learningProviderLogoID(item.provider); logoID != "" {
			document["provider_logo_media_id"] = logoID
		}
		document["thumbnail_media_id"] = thumbnailMediaIDFor(item.slug)
	}
	if descID, descEN := learningProviderDescription(item.provider); descID != "" || descEN != "" {
		document["provider_description_id"] = descID
		document["provider_description_en"] = descEN
	}
	if item.kind == "mini_project" {
		document["outcomes_id"] = []string{"Menghasilkan artefak kecil yang dapat dijelaskan dalam portofolio."}
		document["outcomes_en"] = []string{"Create a small artifact you can explain in a portfolio."}
		document["reflection_prompt"] = "Apa yang kamu buat, pelajari, dan ingin perbaiki?"
	}
	if item.kind == "learning_path" {
		document["steps"] = append([]string(nil), item.stepSlugs...)
	}
	created, err := client.LearningItem.Create().SetID("item_" + item.slug).SetSlug(item.slug).SetKind(learningitem.Kind(item.kind)).SetTitleID(item.titleID).SetTitleEn(item.titleEN).SetSummaryID(item.summaryID).SetSummaryEn(item.summaryEN).SetDocumentJSON(document).SetStatus(learningitem.StatusPublished).SetDraftRevision(1).SetPublishedRevision(1).SetPublishedAt(reviewedAt).SetCreatedBy("seed").SetUpdatedBy("seed").Save(ctx)
	if err != nil {
		return err
	}
	_, err = client.LearningRevision.Create().SetID("learnrev_" + item.slug).SetItemID(created.ID).SetRevision(1).SetDocumentJSON(learningRevisionSnapshot(document, item)).SetKind(learningrevision.KindPublished).SetCreatedBy("seed").Save(ctx)
	return err
}

// fillLearningHubMedia retrofits provider logos and course thumbnails onto items
// that were seeded before media support existed (skip-if-exists keeps them out
// of the create path), so existing databases converge to the same presentation.
func fillLearningHubMedia(ctx context.Context, client *ent.Client) error {
	items, err := client.LearningItem.Query().Where(
		learningitem.StatusNEQ(learningitem.StatusArchived),
		learningitem.CreatedByEQ("seed"),
		learningitem.UpdatedByEQ("seed"),
	).All(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		document := cloneSeedDocument(item.DocumentJSON)
		if fillLearningHubMediaDocument(document, item.Slug) {
			if _, err := client.LearningItem.UpdateOneID(item.ID).SetDocumentJSON(document).Save(ctx); err != nil {
				return err
			}
		}
		if err := fillPublishedLearningHubMedia(ctx, client, item, document); err != nil {
			return err
		}
	}
	return nil
}

// fillPublishedLearningHubMedia keeps the immutable public snapshot of an
// older seed-created item aligned with media added by a later seeder version.
// Admin-published revisions are deliberately excluded: the seeder must never
// overwrite editorial content managed through the control plane.
func fillPublishedLearningHubMedia(ctx context.Context, client *ent.Client, item *ent.LearningItem, draftDocument map[string]any) error {
	revision, err := client.LearningRevision.Query().Where(
		learningrevision.ItemIDEQ(item.ID),
		learningrevision.KindEQ(learningrevision.KindPublished),
	).Order(ent.Desc(learningrevision.FieldRevision)).First(ctx)
	if ent.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if revision.CreatedBy != "seed" {
		return nil
	}

	revisionDocument := cloneSeedDocument(revision.DocumentJSON)
	publicDocument := revisionDocument
	if rawSnapshot, ok := revisionDocument["_learning_item_snapshot"].(map[string]any); ok {
		snapshot := cloneSeedDocument(rawSnapshot)
		if rawDocument, ok := snapshot["document"].(map[string]any); ok {
			publicDocument = cloneSeedDocument(rawDocument)
			snapshot["document"] = publicDocument
			revisionDocument["_learning_item_snapshot"] = snapshot
		}
	}

	changed := false
	for _, key := range []string{"provider_logo_media_id", "thumbnail_media_id", "provider_description_id", "provider_description_en"} {
		if _, exists := publicDocument[key]; exists {
			continue
		}
		if value, exists := draftDocument[key]; exists {
			publicDocument[key] = value
			changed = true
		}
	}
	if !changed {
		return nil
	}
	_, err = client.LearningRevision.UpdateOneID(revision.ID).SetDocumentJSON(revisionDocument).Save(ctx)
	return err
}

func fillLearningHubMediaDocument(document map[string]any, slug string) bool {
	changed := false
	logoID := learningProviderLogoID(seedDocumentString(document, "provider"))
	if logoID != "" {
		if _, exists := document["provider_logo_media_id"]; !exists {
			document["provider_logo_media_id"] = logoID
			changed = true
		}
	}
	if _, exists := document["thumbnail_media_id"]; !exists {
		document["thumbnail_media_id"] = thumbnailMediaIDFor(slug)
		changed = true
	}
	if _, exists := document["provider_description_id"]; !exists {
		if descID, descEN := learningProviderDescription(seedDocumentString(document, "provider")); descID != "" || descEN != "" {
			document["provider_description_id"] = descID
			document["provider_description_en"] = descEN
			changed = true
		}
	}
	return changed
}

func cloneSeedDocument(document map[string]any) map[string]any {
	clone := make(map[string]any, len(document))
	for key, value := range document {
		clone[key] = value
	}
	return clone
}

func seedDocumentString(document map[string]any, key string) string {
	if value, ok := document[key].(string); ok {
		return value
	}
	return ""
}

func learningProgramSlugs(clusterSlug string) []string {
	for _, cluster := range learningSeedClusters {
		if cluster.slug != clusterSlug {
			continue
		}
		programs := make([]string, 0, len(cluster.programs))
		for _, program := range cluster.programs {
			programs = append(programs, program.slug)
		}
		return programs
	}
	return nil
}

func learningDuration(kind string, order int) int {
	switch kind {
	case "learning_path":
		return 180
	case "mini_project":
		return 90
	case "certification":
		return 120
	case "career_snapshot":
		return 20
	case "toolkit":
		return 30
	default:
		return 45 + (order%3)*15
	}
}
