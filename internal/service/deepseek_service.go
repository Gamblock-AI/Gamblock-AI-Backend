package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/deepseek"
)

var (
	ErrTranslationMaxTexts  = errors.New("too many texts in a single translation request")
	ErrTranslationEmptyText = errors.New("empty source text")
	ErrSameLanguage         = errors.New("source and target languages must be different")
	ErrInvalidLanguage      = errors.New("unsupported language code")
	ErrTextTooLong          = errors.New("source text exceeds maximum length")
)

const (
	maxTextsPerRequest = 50
	maxCharsPerText    = 2000
	translationBatchSep = "\n---ITEM---\n"
)

type DeepSeekService struct {
	client *deepseek.Client
	logger *zap.Logger
}

func NewDeepSeekService(cfg config.Config, logger *zap.Logger) *DeepSeekService {
	if strings.TrimSpace(cfg.DeepSeekAPIKey) == "" {
		logger.Warn("DEEPSEEK_API_KEY is not configured; translation will be unavailable")
	}
	return &DeepSeekService{
		client: deepseek.NewClient(cfg.DeepSeekBaseURL, cfg.DeepSeekAPIKey, cfg.DeepSeekModel),
		logger: logger,
	}
}

func (s *DeepSeekService) BatchTranslate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
	if err := validateTranslateInput(texts, sourceLang, targetLang); err != nil {
		return nil, err
	}

	stripped := make([]string, len(texts))
	for i, t := range texts {
		stripped[i] = strings.TrimSpace(t)
	}

	systemPrompt := buildBatchSystemPrompt(sourceLang, targetLang)
	userMessage := buildBatchUserMessage(stripped)

	result, err := s.client.Chat(ctx, systemPrompt, userMessage)
	if err != nil {
		s.logger.Error("deepseek translation failed",
			zap.Error(err),
			zap.String("source_lang", sourceLang),
			zap.String("target_lang", targetLang),
			zap.Int("text_count", len(texts)),
		)
		return nil, err
	}

	translations, err := parseBatchResponse(result, len(texts))
	if err != nil {
		s.logger.Error("deepseek response parsing failed",
			zap.Error(err),
			zap.String("raw_response", result),
		)
		return nil, fmt.Errorf("unexpected translation response format")
	}

	for i := range translations {
		translations[i] = strings.TrimSpace(translations[i])
	}

	if s.logger != nil {
		s.logger.Debug("translation completed",
			zap.String("source_lang", sourceLang),
			zap.String("target_lang", targetLang),
			zap.Int("text_count", len(translations)),
		)
	}

	return translations, nil
}

func validateTranslateInput(texts []string, sourceLang, targetLang string) error {
	if sourceLang != "id" && sourceLang != "en" {
		return ErrInvalidLanguage
	}
	if targetLang != "id" && targetLang != "en" {
		return ErrInvalidLanguage
	}
	if sourceLang == targetLang {
		return ErrSameLanguage
	}
	if len(texts) == 0 {
		return ErrTranslationEmptyText
	}
	if len(texts) > maxTextsPerRequest {
		return ErrTranslationMaxTexts
	}
	for _, t := range texts {
		if strings.TrimSpace(t) == "" {
			continue
		}
		if utf8.RuneCountInString(t) > maxCharsPerText {
			return ErrTextTooLong
		}
	}
	return nil
}

func buildBatchSystemPrompt(sourceLang, targetLang string) string {
	return fmt.Sprintf(
		"You are a professional translator. Translate each item from %s to %s. "+
			"Items are separated by the delimiter '---ITEM---'. "+
			"Return ONLY the translations in the same order, separated by the same delimiter '---ITEM---'. "+
			"Do not include commentary, numbering, quotation marks, or any other text. "+
			"Preserve the original tone, register, and meaning exactly.",
		languageNameCode(sourceLang), languageNameCode(targetLang),
	)
}

func buildBatchUserMessage(texts []string) string {
	return strings.Join(texts, translationBatchSep)
}

func parseBatchResponse(raw string, expectedCount int) ([]string, error) {
	parts := strings.Split(raw, translationBatchSep)
	if len(parts) != expectedCount {
		return nil, fmt.Errorf("expected %d items, got %d", expectedCount, len(parts))
	}
	return parts, nil
}

func languageNameCode(code string) string {
	switch code {
	case "id":
		return "Bahasa Indonesia"
	case "en":
		return "English"
	default:
		return code
	}
}
