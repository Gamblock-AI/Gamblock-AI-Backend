package model

import "testing"

func TestPaginationHelpersCoverNormalizationAndSlices(t *testing.T) {
	query := PaginationQuery{Page: 0, PageSize: 0, Limit: 250}
	page, limit, offset := query.Normalize(25)
	if page != 1 || limit != 100 || offset != 0 {
		t.Fatalf("Normalize() = (%d, %d, %d), want (1, 100, 0)", page, limit, offset)
	}

	query = PaginationQuery{Page: 3, PageSize: 2}
	page, limit, offset = query.Normalize(25)
	if page != 3 || limit != 2 || offset != 4 {
		t.Fatalf("Normalize() = (%d, %d, %d), want (3, 2, 4)", page, limit, offset)
	}

	list := PaginateSlice([]string{"a", "b", "c"}, PaginationQuery{Page: 2, Limit: 2}, 10)
	if len(list.Items) != 1 || list.Items[0] != "c" || list.TotalCount != 3 || list.HasMore {
		t.Fatalf("PaginateSlice() = %#v, want page two with one item", list)
	}

	empty := PaginateSlice([]string{"a"}, PaginationQuery{Page: 4, Limit: 2}, 10)
	if len(empty.Items) != 0 || empty.TotalCount != 1 {
		t.Fatalf("PaginateSlice() out of range = %#v, want empty page", empty)
	}

	defaulted := NewPaginatedList[string](nil, 0, 1, 0)
	if defaulted.Items == nil || defaulted.TotalPages != 1 || defaulted.PageSize != 10 {
		t.Fatalf("NewPaginatedList() defaults = %#v", defaulted)
	}
}

func TestRoleAndMissionHelpers(t *testing.T) {
	for _, role := range []string{RoleUser, RolePartner, RoleAdmin} {
		if !IsAccountRole(role) {
			t.Fatalf("IsAccountRole(%q) = false", role)
		}
	}
	if IsAccountRole("unknown") {
		t.Fatal("IsAccountRole(unknown) = true")
	}

	legacy := DailyMission{Mission1: true, Mission3: true, Mission6: true}
	if got := legacy.CompletedTaskCount(); got != 3 {
		t.Fatalf("legacy CompletedTaskCount() = %d, want 3", got)
	}
	if got := legacy.SystemCompletedTaskCount(); got != 3 {
		t.Fatalf("legacy SystemCompletedTaskCount() = %d, want 3", got)
	}

	mission := DailyMission{
		Mission1: true,
		Mission2: true,
		TaskRecords: []MissionRecord{
			{Key: "mission_1", Source: "system", Status: "completed"},
			{Key: "custom_1", Source: "custom", Status: "completed"},
			{Key: "mission_4", Source: "system", Status: "pending"},
		},
	}
	if got := mission.CompletedTaskCount(); got != 2 {
		t.Fatalf("recorded CompletedTaskCount() = %d, want 2", got)
	}
	if got := mission.SystemCompletedTaskCount(); got != 2 {
		t.Fatalf("recorded SystemCompletedTaskCount() = %d, want 2", got)
	}
}
