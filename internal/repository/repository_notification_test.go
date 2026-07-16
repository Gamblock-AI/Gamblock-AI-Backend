package repository

import (
	"context"
	"testing"
)

func TestNotification_Queue(t *testing.T) {
	repo, _ := newRepo(t)
	_ = repo.QueueNotification(context.Background(), "ntf_t", "APR-1", "", "email", "suci@gmail.com")
}

func TestNotification_QueueInMemory(t *testing.T) {
	repo, _ := newRepo(t)
	_ = repo.QueueNotification(context.Background(), "ntf_x", "APR-1", "sc-1", "whatsapp", "+62")
}
