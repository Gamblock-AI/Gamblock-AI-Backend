package config

import "testing"

func TestIsProduction(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{"production", true},
		{"PRODUCTION", true},
		{"", true},      // safe default = production
		{"anything", true},
		{"development", false},
		{"staging", false},
		{"test", false},
		{"local", false},
	}
	for _, tc := range cases {
		c := Config{AppEnv: tc.env}
		assert := func(got bool) {
			if got != tc.want {
				t.Errorf("AppEnv=%q: got %v want %v", tc.env, got, tc.want)
			}
		}
		assert(c.IsProduction())
	}
}
