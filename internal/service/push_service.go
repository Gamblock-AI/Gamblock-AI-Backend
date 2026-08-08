package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

var errWebPushRejected = errors.New("web push endpoint rejected the message")

// PushPayload is the notification body rendered by the website service worker.
// URL is the authenticated route the notification click should open.
type PushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon"`
	URL   string `json:"url"`
}

// PushService delivers opt-in Web Push messages to stored subscriptions and
// prunes endpoints the push service declares invalid.
type PushService struct {
	repo   *repository.Repository
	cfg    config.Config
	logger *zap.Logger
}

func NewPushService(repo *repository.Repository, cfg config.Config, logger *zap.Logger) *PushService {
	return &PushService{repo: repo, cfg: cfg, logger: logger}
}

// UpsertSubscription records a browser push subscription for the user.
func (s *PushService) UpsertSubscription(ctx context.Context, userID, endpoint, p256dh, authKey string, userAgent *string) (model.PushSubscription, error) {
	return s.repo.UpsertPushSubscription(ctx, userID, endpoint, p256dh, authKey, userAgent)
}

// DeleteSubscription removes a browser push subscription owned by the user.
func (s *PushService) DeleteSubscription(ctx context.Context, userID, endpoint string) error {
	return s.repo.DeletePushSubscription(ctx, userID, endpoint)
}

// SendToUser pushes payload to every subscription owned by userID and reports
// the number of endpoints reached. Invalid endpoints are removed from storage.
func (s *PushService) SendToUser(ctx context.Context, userID string, payload PushPayload) (int, error) {
	subscriptions, err := s.repo.PushSubscriptionsForUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, subscription := range subscriptions {
		if err := s.sendOne(ctx, subscription, payloadBytes); err != nil {
			s.logger.Info("web push delivery failed",
				zap.String("subscription_id", subscription.ID),
				zap.Error(err))
			continue
		}
		sent++
	}
	return sent, nil
}

func (s *PushService) sendOne(ctx context.Context, subscription model.PushSubscription, payload []byte) error {
	if !s.configured() {
		return nil
	}
	sub := &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			P256dh: subscription.P256dh,
			Auth:   subscription.AuthKey,
		},
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := webpush.SendNotification(payload, sub, &webpush.Options{
		HTTPClient:      client,
		VAPIDPublicKey:  s.cfg.VAPIDPublicKey,
		VAPIDPrivateKey: s.cfg.VAPIDPrivateKey,
		Subscriber:      s.cfg.VAPIDSubject,
		TTL:             12 * 3600,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return s.repo.RemovePushSubscriptionByID(ctx, subscription.ID)
	}
	if resp.StatusCode >= 400 {
		return errWebPushRejected
	}
	return nil
}

func (s *PushService) configured() bool {
	return s.cfg.VAPIDPublicKey != "" && s.cfg.VAPIDPrivateKey != "" && s.cfg.VAPIDSubject != ""
}
