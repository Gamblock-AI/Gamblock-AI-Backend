package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

const maxAvatarBytes int64 = 2 << 20

func (s *ClientService) UploadAvatar(ctx context.Context, userID string, source io.Reader) (model.User, error) {
	data, err := io.ReadAll(io.LimitReader(source, maxAvatarBytes+1))
	if err != nil {
		return model.User{}, err
	}
	if len(data) == 0 || int64(len(data)) > maxAvatarBytes || http.DetectContentType(data) != "image/webp" {
		return model.User{}, errors.New("avatar must be a WebP image no larger than 2 MiB")
	}
	if err := os.MkdirAll(s.avatarStoragePath, 0o750); err != nil {
		return model.User{}, err
	}

	oldKey, _ := s.repo.UserAvatarStorageKey(ctx, userID)
	storageKey := "avatar/" + uuid.NewString() + ".webp"
	target := filepath.Join(s.avatarStoragePath, filepath.Base(storageKey))
	temp, err := os.CreateTemp(s.avatarStoragePath, ".avatar-upload-*")
	if err != nil {
		return model.User{}, err
	}
	tempName := temp.Name()
	defer os.Remove(tempName) //nolint:errcheck
	if _, err = temp.Write(data); err != nil {
		_ = temp.Close()
		return model.User{}, err
	}
	if err = temp.Chmod(0o640); err != nil {
		_ = temp.Close()
		return model.User{}, err
	}
	if err = temp.Close(); err != nil {
		return model.User{}, err
	}
	if err = os.Rename(tempName, target); err != nil {
		return model.User{}, err
	}

	user, err := s.repo.UpdateUserAvatar(ctx, userID, &storageKey)
	if err != nil {
		_ = os.Remove(target)
		return model.User{}, err
	}
	if oldKey != "" && oldKey != storageKey {
		_ = os.Remove(filepath.Join(s.avatarStoragePath, filepath.Base(oldKey)))
	}
	return user, nil
}

func (s *ClientService) DeleteAvatar(ctx context.Context, userID string) (model.User, error) {
	oldKey, _ := s.repo.UserAvatarStorageKey(ctx, userID)
	user, err := s.repo.UpdateUserAvatar(ctx, userID, nil)
	if err != nil {
		return model.User{}, err
	}
	if oldKey != "" {
		if err := os.Remove(filepath.Join(s.avatarStoragePath, filepath.Base(oldKey))); err != nil && !errors.Is(err, os.ErrNotExist) {
			return model.User{}, err
		}
	}
	return user, nil
}

func (s *ClientService) AvatarFile(ctx context.Context, userID string) (*os.File, int64, error) {
	storageKey, ok := s.repo.UserAvatarStorageKey(ctx, userID)
	if !ok || !strings.HasPrefix(storageKey, "avatar/") || filepath.Base(storageKey) == "." {
		return nil, 0, errors.New("avatar not found")
	}
	path := filepath.Join(s.avatarStoragePath, filepath.Base(storageKey))
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("open avatar: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, err
	}
	return file, info.Size(), nil
}
