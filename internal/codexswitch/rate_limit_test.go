package codexswitch

import "testing"

func TestOfficialUsageURLNormalizesChatGPTHosts(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "empty uses default chatgpt backend api",
			baseURL: "",
			want:    "https://chatgpt.com/backend-api/wham/usage",
		},
		{
			name:    "chatgpt root adds backend-api",
			baseURL: "https://chatgpt.com",
			want:    "https://chatgpt.com/backend-api/wham/usage",
		},
		{
			name:    "chat.openai root adds backend-api",
			baseURL: "https://chat.openai.com/",
			want:    "https://chat.openai.com/backend-api/wham/usage",
		},
		{
			name:    "codex api host keeps codex path",
			baseURL: "https://api.openai.com",
			want:    "https://api.openai.com/api/codex/usage",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := officialUsageURL(test.baseURL)
			if got != test.want {
				t.Fatalf("officialUsageURL(%q) = %q, want %q", test.baseURL, got, test.want)
			}
		})
	}
}
