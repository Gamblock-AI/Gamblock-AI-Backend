package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/checkin"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/intention"
	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) GetIntention(ctx context.Context, userID string) (model.Intention, bool) {
	if r.db != nil {
		item, err := r.db.Intention.Query().
			Where(intention.UserID(userID), intention.StatusEQ(intention.StatusActive)).
			Order(ent.Desc(intention.FieldUpdatedAt)).
			First(ctx)
		if err != nil {
			return model.Intention{}, false
		}
		return intentionToModel(item), true
	}
	r.store.RLock()
	defer r.store.RUnlock()
	for _, intn := range r.store.Intentions {
		if intn.UserID == userID && intn.Status == "active" {
			return intn, true
		}
	}
	return model.Intention{}, false
}

func intentionToModel(item *ent.Intention) model.Intention {
	return model.Intention{
		ID:             item.ID,
		UserID:         item.UserID,
		Text:           item.IntentionText,
		Status:         item.Status.String(),
		SchoolImpact:   schoolImpactStr(item.SchoolImpact),
		MoneySpent:     moneySpentStr(item.MoneySpent),
		ScreenTime:     screenTimeStr(item.ScreenTime),
		QuitAttempts:   quitAttemptsStr(item.QuitAttempts),
		QuitMotivation: quitMotivationStr(item.QuitMotivation),
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func (r *Repository) SaveIntention(ctx context.Context, userID, text, status string, quiz model.Intention) (model.Intention, error) {
	now := time.Now().UTC()
	if r.db != nil {
		item, err := r.db.Intention.Query().
			Where(intention.UserID(userID), intention.StatusEQ(intention.StatusActive)).
			Order(ent.Desc(intention.FieldUpdatedAt)).
			First(ctx)
		if err == nil {
			qb := item.Update().SetIntentionText(text).SetStatus(intention.Status(status))
			qb = setQuizFields(qb, quiz)
			item, err = qb.Save(ctx)
			if err != nil {
				return model.Intention{}, err
			}
			r.RefreshStore(ctx)
			return intentionToModel(item), nil
		}
		cb := r.db.Intention.Create().
			SetID("int_" + uuid.NewString()[:8]).
			SetUserID(userID).
			SetIntentionText(text).
			SetStatus(intention.Status(status))
		cb = setQuizCreateFields(cb, quiz)
		item, err = cb.Save(ctx)
		if err != nil {
			return model.Intention{}, err
		}
		r.RefreshStore(ctx)
		return intentionToModel(item), nil
	}
	r.store.Lock()
	defer r.store.Unlock()

	for i, intn := range r.store.Intentions {
		if intn.UserID == userID {
			r.store.Intentions[i].Text = text
			r.store.Intentions[i].Status = status
			r.store.Intentions[i].SchoolImpact = quiz.SchoolImpact
			r.store.Intentions[i].MoneySpent = quiz.MoneySpent
			r.store.Intentions[i].ScreenTime = quiz.ScreenTime
			r.store.Intentions[i].QuitAttempts = quiz.QuitAttempts
			r.store.Intentions[i].QuitMotivation = quiz.QuitMotivation
			r.store.Intentions[i].UpdatedAt = now
			return r.store.Intentions[i], nil
		}
	}

	newEntry := store.Intention{
		ID:             "int_" + uuid.NewString()[:8],
		UserID:         userID,
		Text:           text,
		Status:         status,
		SchoolImpact:   quiz.SchoolImpact,
		MoneySpent:     quiz.MoneySpent,
		ScreenTime:     quiz.ScreenTime,
		QuitAttempts:   quiz.QuitAttempts,
		QuitMotivation: quiz.QuitMotivation,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	r.store.Intentions = append(r.store.Intentions, newEntry)
	return newEntry, nil
}

func (r *Repository) GetCheckIns(ctx context.Context, userID string) ([]model.CheckIn, error) {
	if r.db != nil {
		rows, err := r.db.CheckIn.Query().
			Where(checkin.UserID(userID)).
			Order(ent.Desc(checkin.FieldCreatedAt)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		list := make([]model.CheckIn, 0, len(rows))
		for _, item := range rows {
			list = append(list, model.CheckIn{ID: item.ID, UserID: item.UserID, Mood: item.MoodScore, Urge: item.UrgeScore, Context: value(item.ContextText), CreatedAt: item.CreatedAt})
		}
		return list, nil
	}
	r.store.RLock()
	defer r.store.RUnlock()
	var list []model.CheckIn
	for _, chk := range r.store.CheckIns {
		if chk.UserID == userID {
			list = append(list, chk)
		}
	}
	return list, nil
}

func (r *Repository) SaveCheckIn(ctx context.Context, userID string, mood, urge int, contextText string) (model.CheckIn, error) {
	now := time.Now().UTC()
	jakarta := time.FixedZone("Asia/Jakarta", 7*60*60)
	localNow := now.In(jakarta)
	dayStartLocal := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, jakarta)
	dayStart := dayStartLocal.UTC()
	dayEnd := dayStartLocal.Add(24 * time.Hour).UTC()
	if r.db != nil {
		existing, existingErr := r.db.CheckIn.Query().Where(
			checkin.UserIDEQ(userID), checkin.CreatedAtGTE(dayStart), checkin.CreatedAtLT(dayEnd),
		).Only(ctx)
		if existingErr == nil {
			item, err := existing.Update().SetMoodScore(mood).SetUrgeScore(urge).
				SetNillableContextText(optional(contextText)).Save(ctx)
			if err != nil {
				return model.CheckIn{}, err
			}
			r.RefreshStore(ctx)
			return model.CheckIn{ID: item.ID, UserID: item.UserID, Mood: item.MoodScore, Urge: item.UrgeScore, Context: value(item.ContextText), CreatedAt: item.CreatedAt}, nil
		}
		item, err := r.db.CheckIn.Create().
			SetID("chk_" + uuid.NewString()[:8]).
			SetUserID(userID).
			SetMoodScore(mood).
			SetUrgeScore(urge).
			SetNillableContextText(optional(contextText)).
			Save(ctx)
		if err != nil {
			return model.CheckIn{}, err
		}
		r.RefreshStore(ctx)
		return model.CheckIn{ID: item.ID, UserID: item.UserID, Mood: item.MoodScore, Urge: item.UrgeScore, Context: value(item.ContextText), CreatedAt: item.CreatedAt}, nil
	}
	r.store.Lock()
	for i := range r.store.CheckIns {
		item := &r.store.CheckIns[i]
		if item.UserID == userID && !item.CreatedAt.Before(dayStart) && item.CreatedAt.Before(dayEnd) {
			item.Mood = mood
			item.Urge = urge
			item.Context = contextText
			updated := *item
			r.store.Unlock()
			return updated, nil
		}
	}
	newEntry := store.CheckIn{
		ID:        "chk_" + uuid.NewString()[:8],
		UserID:    userID,
		Mood:      mood,
		Urge:      urge,
		Context:   contextText,
		CreatedAt: now,
	}
	r.store.CheckIns = append(r.store.CheckIns, newEntry)
	r.store.Unlock()
	return newEntry, nil
}

func schoolImpactStr(v *intention.SchoolImpact) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func moneySpentStr(v *intention.MoneySpent) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func screenTimeStr(v *intention.ScreenTime) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func quitAttemptsStr(v *intention.QuitAttempts) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func quitMotivationStr(v *intention.QuitMotivation) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func schoolImpactVal(v string) *intention.SchoolImpact {
	if v == "" {
		return nil
	}
	s := intention.SchoolImpact(v)
	return &s
}

func moneySpentVal(v string) *intention.MoneySpent {
	if v == "" {
		return nil
	}
	s := intention.MoneySpent(v)
	return &s
}

func screenTimeVal(v string) *intention.ScreenTime {
	if v == "" {
		return nil
	}
	s := intention.ScreenTime(v)
	return &s
}

func quitAttemptsVal(v string) *intention.QuitAttempts {
	if v == "" {
		return nil
	}
	s := intention.QuitAttempts(v)
	return &s
}

func quitMotivationVal(v string) *intention.QuitMotivation {
	if v == "" {
		return nil
	}
	s := intention.QuitMotivation(v)
	return &s
}

func setQuizFields(qb *ent.IntentionUpdateOne, quiz model.Intention) *ent.IntentionUpdateOne {
	if quiz.SchoolImpact != "" {
		qb = qb.SetSchoolImpact(intention.SchoolImpact(quiz.SchoolImpact))
	} else {
		qb = qb.ClearSchoolImpact()
	}
	if quiz.MoneySpent != "" {
		qb = qb.SetMoneySpent(intention.MoneySpent(quiz.MoneySpent))
	} else {
		qb = qb.ClearMoneySpent()
	}
	if quiz.ScreenTime != "" {
		qb = qb.SetScreenTime(intention.ScreenTime(quiz.ScreenTime))
	} else {
		qb = qb.ClearScreenTime()
	}
	if quiz.QuitAttempts != "" {
		qb = qb.SetQuitAttempts(intention.QuitAttempts(quiz.QuitAttempts))
	} else {
		qb = qb.ClearQuitAttempts()
	}
	if quiz.QuitMotivation != "" {
		qb = qb.SetQuitMotivation(intention.QuitMotivation(quiz.QuitMotivation))
	} else {
		qb = qb.ClearQuitMotivation()
	}
	return qb
}

func setQuizCreateFields(cb *ent.IntentionCreate, quiz model.Intention) *ent.IntentionCreate {
	cb = cb.SetNillableSchoolImpact(schoolImpactVal(quiz.SchoolImpact))
	cb = cb.SetNillableMoneySpent(moneySpentVal(quiz.MoneySpent))
	cb = cb.SetNillableScreenTime(screenTimeVal(quiz.ScreenTime))
	cb = cb.SetNillableQuitAttempts(quitAttemptsVal(quiz.QuitAttempts))
	cb = cb.SetNillableQuitMotivation(quitMotivationVal(quiz.QuitMotivation))
	return cb
}
