package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateHourlyMetadata_AcceptsValidHistogram(t *testing.T) {
	hourly := make([]any, 24)
	for index := range hourly {
		hourly[index] = float64(index % 3)
	}
	err := validateHourlyMetadata(map[string]any{"hourly": hourly}, 1000)
	require.NoError(t, err)
}

func TestValidateHourlyMetadata_RejectsTotalAboveCount(t *testing.T) {
	hourly := make([]any, 24)
	hourly[0] = float64(5)
	err := validateHourlyMetadata(map[string]any{"hourly": hourly}, 3)
	assert.Error(t, err)
}

func TestValidateHourlyMetadata_RejectsBadShape(t *testing.T) {
	assert.Error(t, validateHourlyMetadata(map[string]any{"hourly": []any{1, 2}}, 10))
	assert.Error(t, validateHourlyMetadata(map[string]any{"other": "x"}, 10))
	assert.Error(t, validateHourlyMetadata(map[string]any{"hourly": []any{"x", 2}, "count": 3}, 10))
	assert.NoError(t, validateHourlyMetadata(nil, 10))
}

func TestValidateHourlyMetadata_RejectsNegativeOrFractional(t *testing.T) {
	negative := make([]any, 24)
	negative[1] = float64(-1)
	assert.Error(t, validateHourlyMetadata(map[string]any{"hourly": negative}, 10))

	fractional := make([]any, 24)
	fractional[1] = float64(1.5)
	assert.Error(t, validateHourlyMetadata(map[string]any{"hourly": fractional}, 10))
}
