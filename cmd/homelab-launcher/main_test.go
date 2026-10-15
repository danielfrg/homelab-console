package main

import "testing"

func TestListenAddr(t *testing.T) {
	for _, tc := range []struct {
		name, launcher, dashboard, want string
	}{
		{"default", "", "", "127.0.0.1:8080"},
		{"launcher", ":8081", "", ":8081"},
		{"legacy fallback", "", ":8082", ":8082"},
		{"launcher takes precedence", ":8081", ":8082", ":8081"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOMELAB_LAUNCHER_ADDR", tc.launcher)
			t.Setenv("HOMELAB_DASHBOARD_ADDR", tc.dashboard)
			if got := listenAddr(); got != tc.want {
				t.Fatalf("listenAddr() = %q, want %q", got, tc.want)
			}
		})
	}
}
