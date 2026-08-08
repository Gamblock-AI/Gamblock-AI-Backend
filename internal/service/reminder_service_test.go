package service

import (
	"context"
	"testing"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReminderService_UpdatePreferenceValidation(t *testing.T) {
	svc := NewReminderService(repository.New(nil, store.New()))

	_, err := svc.UpdatePreference(context.Background(), "usr_1", true, "25:00", "Asia/Jakarta", "id")
	require.ErrorIs(t, err, ErrReminderPreferenceInvalid)

	_, err = svc.UpdatePreference(context.Background(), "usr_1", true, "19:00", "Not/AZone", "id")
	require.ErrorIs(t, err, ErrReminderPreferenceInvalid)

	pref, err := svc.UpdatePreference(context.Background(), "usr_1", true, "06:30", "", "fr")
	require.NoError(t, err)
	assert.True(t, pref.Enabled)
	assert.Equal(t, "06:30", pref.LocalTime)
	assert.Equal(t, "Asia/Jakarta", pref.Timezone)
	assert.Equal(t, "id", pref.Locale)

	loaded, err := svc.GetPreference(context.Background(), "usr_1")
	require.NoError(t, err)
	assert.True(t, loaded.Enabled)
}

func TestReminderService_UpdatePreferenceDisableKeepsDefaults(t *testing.T) {
	svc := NewReminderService(repository.New(nil, store.New()))
	pref, err := svc.UpdatePreference(context.Background(), "usr_2", false, "21:15", "Asia/Jakarta", "en")
	require.NoError(t, err)
	assert.False(t, pref.Enabled)
	assert.Equal(t, "21:15", pref.LocalTime)
}
