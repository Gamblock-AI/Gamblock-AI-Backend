package service

import (
	"net/url"
	"testing"
)

func TestAllowedSocialQueryFacebookProfileID(t *testing.T) {
	parsed, err := url.Parse("https://web.facebook.com/profile.php?id=61591544143202")
	if err != nil {
		t.Fatal(err)
	}
	if !allowedSocialQuery("facebook", parsed) {
		t.Fatal("expected numeric Facebook profile ID query to be allowed")
	}
}

func TestAllowedSocialQueryRejectsOtherQueries(t *testing.T) {
	parsed, err := url.Parse("https://instagram.com/gamblockai?utm_source=share")
	if err != nil {
		t.Fatal(err)
	}
	if allowedSocialQuery("instagram", parsed) {
		t.Fatal("expected non-Facebook social query to be rejected")
	}
}
