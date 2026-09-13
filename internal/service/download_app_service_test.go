package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAdminServiceUpdateDownloadAppPublishesOfficialRelease(t *testing.T) {
	repo, st := newRepo(t)
	svc := NewAdminService(repo, testCfg(), NewWhatsAppService(testCfg(), zap.NewNop()), zap.NewNop())
	app := st.Snapshot().DownloadApps[0]

	updated, err := svc.UpdateDownloadApp(context.Background(), "usr_nasywa", app.Platform, "publish stable release", app)
	require.NoError(t, err)
	assert.Equal(t, app.Platform, updated.Platform)
	assert.Equal(t, "v1.6.16", updated.Version)
	assert.Equal(t, "gamblock-ai-research-1.6.16+38.apk", updated.Assets[0].FileName)

	publicApps, err := svc.PublicDownloadApps(context.Background())
	require.NoError(t, err)
	assert.Len(t, publicApps, 3)
	var audited bool
	for _, event := range st.Snapshot().AuditEvents {
		if event.Action == "download_app_updated" && event.TargetType == "download_app" && event.Target == app.Platform {
			audited = true
			break
		}
	}
	assert.True(t, audited)
}

func TestAdminServiceUpdateDownloadAppRejectsUnofficialOrUnstableAssets(t *testing.T) {
	repo, st := newRepo(t)
	svc := NewAdminService(repo, testCfg(), NewWhatsAppService(testCfg(), zap.NewNop()), zap.NewNop())
	app := st.Snapshot().DownloadApps[0]

	app.Assets = append(app.Assets[:0:0], app.Assets...)
	app.Assets[0].URL = "https://example.com/releases/download/v1.6.16/gamblock-ai-research-1.6.16%2B38.apk"
	_, err := svc.UpdateDownloadApp(context.Background(), "usr_nasywa", app.Platform, "reject third-party asset", app)
	require.Error(t, err)

	app = st.Snapshot().DownloadApps[0]
	app.Version = "v1.6.16-beta.1"
	_, err = svc.UpdateDownloadApp(context.Background(), "usr_nasywa", app.Platform, "reject prerelease", app)
	require.Error(t, err)
}
