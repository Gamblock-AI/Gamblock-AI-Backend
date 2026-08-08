package seed

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/educationmedia"
)

//go:embed assets/learning-hub/logos/*.png
var learningHubLogoAssets embed.FS

//go:embed assets/learning-hub/thumbnails/*.jpg
var learningHubThumbnailAssets embed.FS

const learningHubThumbnailCount = 12

type learningSeedProvider struct {
	key      string
	name     string
	logoFile string
}

// Recognised external learning services with a bundled dummy logo. Items whose
// provider is not listed here still keep their provider name; the client falls
// back to an initials tile when no logo media is attached.
var learningSeedProviders = []learningSeedProvider{
	{"dicoding", "Dicoding", "dicoding.png"},
	{"freecodecamp", "freeCodeCamp", "freecodecamp.png"},
	{"coursera", "Coursera", "coursera.png"},
	{"hubspot", "HubSpot Academy", "hubspot.png"},
	{"britishcouncil", "British Council", "british-council.png"},
	{"openlearn", "OpenLearn", "openlearn.png"},
	{"microsoft", "Microsoft Learn", "microsoft-learn.png"},
	{"autodesk", "Autodesk Learning", "autodesk.png"},
}

func learningProviderLogoID(providerName string) string {
	normalized := strings.ToLower(providerName)
	for _, provider := range learningSeedProviders {
		if strings.Contains(normalized, strings.ToLower(provider.name)) ||
			strings.Contains(normalized, strings.ToLower(provider.key)) {
			return "med_seed_lh_logo_" + provider.key
		}
	}
	return ""
}

// thumbnailMediaIDFor deterministically maps an item slug onto one of the
// bundled dummy thumbnails so every course has a thumbnail without shipping a
// unique image per item.
func thumbnailMediaIDFor(slug string) string {
	sum := 0
	for _, r := range slug {
		sum = (sum*31 + int(r)) & 0x7fffffff
	}
	return fmt.Sprintf("med_seed_lh_thumb_%02d", sum%learningHubThumbnailCount+1)
}

// seedLearningHubAssets writes the bundled logo/thumbnail files into the media
// storage root and upserts their EducationMedia rows as published uploads so
// the public media route can serve them.
func seedLearningHubAssets(ctx context.Context, client *ent.Client, root string) error {
	if root == "" {
		root = "./var/media"
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return err
	}

	type asset struct {
		id        string
		file      string
		fs        embed.FS
		prefix    string
		mime      string
		extension string
		width     int
		height    int
	}
	assets := make([]asset, 0, len(learningSeedProviders)+learningHubThumbnailCount)
	for _, provider := range learningSeedProviders {
		assets = append(assets, asset{
			id: "med_seed_lh_logo_" + provider.key, file: provider.logoFile,
			fs: learningHubLogoAssets, prefix: "assets/learning-hub/logos/",
			mime: "image/png", extension: ".png", width: 128, height: 128,
		})
	}
	for i := 1; i <= learningHubThumbnailCount; i++ {
		assets = append(assets, asset{
			id: fmt.Sprintf("med_seed_lh_thumb_%02d", i), file: fmt.Sprintf("thumb-%02d.jpg", i),
			fs: learningHubThumbnailAssets, prefix: "assets/learning-hub/thumbnails/",
			mime: "image/jpeg", extension: ".jpg", width: 1600, height: 900,
		})
	}

	for _, item := range assets {
		data, err := item.fs.ReadFile(item.prefix + item.file)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		storageKey := item.id + item.extension
		target := filepath.Join(root, storageKey)
		if _, err := os.Stat(target); os.IsNotExist(err) {
			if err := os.WriteFile(target, data, 0o640); err != nil {
				return err
			}
		}
		row, err := client.EducationMedia.Query().Where(educationmedia.IDEQ(item.id)).Only(ctx)
		switch {
		case ent.IsNotFound(err):
			_, err = client.EducationMedia.Create().SetID(item.id).SetKind(educationmedia.KindUpload).
				SetPurpose(educationmedia.PurposeThumbnail).SetMediaType(educationmedia.MediaTypeImage).
				SetMimeType(item.mime).SetStorageKey(storageKey).SetOriginalName(item.file).
				SetSizeBytes(int64(len(data))).SetWidth(item.width).SetHeight(item.height).
				SetSha256(hex.EncodeToString(hash[:])).SetStatus(educationmedia.StatusPublished).SetCreatedBy("seed").Save(ctx)
		case err == nil:
			_, err = row.Update().SetStorageKey(storageKey).SetSizeBytes(int64(len(data))).
				SetWidth(item.width).SetHeight(item.height).SetSha256(hex.EncodeToString(hash[:])).
				SetStatus(educationmedia.StatusPublished).Save(ctx)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
