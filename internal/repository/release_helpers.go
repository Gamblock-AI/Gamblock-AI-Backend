package repository

import "github.com/gamblock-ai/gamblock-ai-backend/internal/model"

func copyReleases(source []model.Release) []model.Release {
	var releases []model.Release
	for _, release := range source {
		releases = append(releases, release)
	}
	return releases
}
