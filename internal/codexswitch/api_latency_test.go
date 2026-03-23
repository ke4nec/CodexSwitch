package codexswitch

import "testing"

func TestAPILatencyProbeURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "root host keeps homepage",
			baseURL: "https://api.openai.com",
			want:    "https://api.openai.com/",
		},
		{
			name:    "trailing v1 uses homepage",
			baseURL: "https://api.openai.com/v1",
			want:    "https://api.openai.com/",
		},
		{
			name:    "custom nested v1 removes only final segment",
			baseURL: "https://example.com/openai/v1/",
			want:    "https://example.com/openai",
		},
		{
			name:    "custom path without v1 stays unchanged",
			baseURL: "https://example.com/gateway",
			want:    "https://example.com/gateway",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := apiLatencyProbeURL(test.baseURL)
			if err != nil {
				t.Fatalf("apiLatencyProbeURL(%q) returned error: %v", test.baseURL, err)
			}
			if got != test.want {
				t.Fatalf("apiLatencyProbeURL(%q) = %q, want %q", test.baseURL, got, test.want)
			}
		})
	}
}

func TestOfficialLatencyModelsURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "empty uses api openai models endpoint",
			baseURL: "",
			want:    "https://api.openai.com/v1/models",
		},
		{
			name:    "backend api rewrites to v1 models",
			baseURL: "https://example.com/backend-api",
			want:    "https://example.com/v1/models",
		},
		{
			name:    "existing v1 keeps models endpoint",
			baseURL: "https://api.openai.com/v1",
			want:    "https://api.openai.com/v1/models",
		},
		{
			name:    "custom nested path appends v1 models",
			baseURL: "https://example.com/openai",
			want:    "https://example.com/openai/v1/models",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := officialLatencyModelsURL(test.baseURL)
			if err != nil {
				t.Fatalf("officialLatencyModelsURL(%q) returned error: %v", test.baseURL, err)
			}
			if got != test.want {
				t.Fatalf("officialLatencyModelsURL(%q) = %q, want %q", test.baseURL, got, test.want)
			}
		})
	}
}
