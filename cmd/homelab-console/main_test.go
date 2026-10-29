package main

import "testing"

func TestListenAddr(t *testing.T) {
	for _, tc := range []struct {
		name, launcher, want string
	}{
		{"default", "", "127.0.0.1:8080"},
		{"launcher", ":8081", ":8081"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOMELAB_CONSOLE_ADDR", tc.launcher)
			if got := listenAddr(); got != tc.want {
				t.Fatalf("listenAddr() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestartEnabledParsing(t *testing.T) {
	for value, want := range map[string]bool{"": true, "FALSE ": false, "false": false, "0": true, "true": true} {
		if got := restartEnabled(value); got != want {
			t.Fatalf("restartEnabled(%q) = %v", value, got)
		}
	}
}
