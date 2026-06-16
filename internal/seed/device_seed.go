package seed

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
)

func SeedDevices(ctx context.Context, client *ent.Client, now time.Time) error {
	items := []struct{ id, platform, label, os, status string }{
		{"dev_android", "android", "Gading Android", "Android 15", "active"},
		{"dev_windows", "windows", "Gading Windows", "Windows 11", "degraded"},
	}
	for _, item := range items {
		if _, err := client.Device.Create().SetID(item.id).SetUserID("usr_gading").SetPlatform(device.Platform(item.platform)).SetLabel(item.label).SetAppVersion("1.0.0").SetOsVersion(item.os).SetModelVersion("artifact-v0.3.1").SetRulesetVersion("ruleset-2026.05.1").SetProtectionStatus(device.ProtectionStatus(item.status)).SetLastSeenAt(now.Add(-2 * time.Minute)).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}
