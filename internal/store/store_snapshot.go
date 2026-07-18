package store

import "strings"

func (s *Store) UserByEmail(email string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, user := range s.Users {
		if strings.EqualFold(user.Email, email) {
			return user, true
		}
	}
	return User{}, false
}

func (s *Store) DefaultUser() User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Users[0]
}

func (s *Store) Snapshot() Store {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Store{
		Users:                append([]User(nil), s.Users...),
		Devices:              append([]Device(nil), s.Devices...),
		Partners:             append([]Partner(nil), s.Partners...),
		Approvals:            append([]ApprovalRequest(nil), s.Approvals...),
		Modules:              append([]EducationModule(nil), s.Modules...),
		EducationMedia:       append([]EducationMedia(nil), s.EducationMedia...),
		EducationProgress:    append([]EducationProgress(nil), s.EducationProgress...),
		SupportCases:         append([]SupportCase(nil), s.SupportCases...),
		DataRequests:         append([]DataRequest(nil), s.DataRequests...),
		Organizations:        append([]Organization(nil), s.Organizations...),
		ModelReleases:        append([]Release(nil), s.ModelReleases...),
		RulesetReleases:      append([]Release(nil), s.RulesetReleases...),
		NetworkRulesets:      append([]Release(nil), s.NetworkRulesets...),
		AuditEvents:          append([]AuditEvent(nil), s.AuditEvents...),
		NotificationEvents:   append([]NotificationItem(nil), s.NotificationEvents...),
		JournalEntries:       append([]JournalEntry(nil), s.JournalEntries...),
		Missions:             append([]DailyMission(nil), s.Missions...),
		Intentions:           append([]Intention(nil), s.Intentions...),
		CheckIns:             append([]CheckIn(nil), s.CheckIns...),
		AggregateEvents:      append([]AggregateEvent(nil), s.AggregateEvents...),
		EmergencyKeyRequests: append([]EmergencyKeyRequest(nil), s.EmergencyKeyRequests...),
	}
}

func (s *Store) Lock() {
	s.mu.Lock()
}

func (s *Store) Unlock() {
	s.mu.Unlock()
}

func (s *Store) RLock() {
	s.mu.RLock()
}

func (s *Store) RUnlock() {
	s.mu.RUnlock()
}
