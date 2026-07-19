package seed

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/sitesociallink"
)

// SeedSiteSocialLinks creates the public links only for a fresh social-link
// configuration. Administrator-managed settings are never overwritten.
func SeedSiteSocialLinks(ctx context.Context, client *ent.Client) error {
	count, err := client.SiteSocialLink.Query().Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	links := []struct {
		platform sitesociallink.Platform
		label    string
		url      string
	}{
		{
			platform: sitesociallink.PlatformInstagram,
			label:    "Instagram",
			url:      "https://www.instagram.com/gamblockai.pkmkc/",
		},
		{
			platform: sitesociallink.PlatformFacebook,
			label:    "Facebook",
			url:      "https://web.facebook.com/profile.php?id=61591544143202",
		},
		{
			platform: sitesociallink.PlatformTiktok,
			label:    "TikTok",
			url:      "https://www.tiktok.com/@gamblockai.pkmkc",
		},
	}

	for index, link := range links {
		if _, err := client.SiteSocialLink.Create().
			SetID("social_" + link.platform.String()).
			SetPlatform(link.platform).
			SetLabel(link.label).
			SetURL(link.url).
			SetEnabled(true).
			SetSortOrder(index).
			SetUpdatedBy("seed").
			Save(ctx); err != nil {
			return err
		}
	}

	return nil
}
