package codexswitch

type ProfileType string

const (
	ProfileTypeOfficial ProfileType = "official"
	ProfileTypeAPI      ProfileType = "api"
	ProfileTypeUnknown  ProfileType = "unknown"
)

type RateLimitFetchStatus string

const (
	RateLimitStatusIdle    RateLimitFetchStatus = "idle"
	RateLimitStatusLoading RateLimitFetchStatus = "loading"
	RateLimitStatusSuccess RateLimitFetchStatus = "success"
	RateLimitStatusStale   RateLimitFetchStatus = "stale"
	RateLimitStatusError   RateLimitFetchStatus = "error"
)

type LatencyTestStatus string

const (
	LatencyTestStatusIdle    LatencyTestStatus = "idle"
	LatencyTestStatusSuccess LatencyTestStatus = "success"
	LatencyTestStatusError   LatencyTestStatus = "error"
)

type LatencyHistoryEntry struct {
	Status       LatencyTestStatus `json:"status"`
	Available    bool              `json:"available"`
	LatencyMs    *int64            `json:"latencyMs,omitempty"`
	StatusCode   *int              `json:"statusCode,omitempty"`
	ErrorMessage string            `json:"errorMessage,omitempty"`
	ErrorType    string            `json:"errorType,omitempty"`
	ErrorCode    string            `json:"errorCode,omitempty"`
	CheckedAt    string            `json:"checkedAt,omitempty"`
}

type AppSettings struct {
	CodexHomePath string `json:"codexHomePath"`
	LastOpenedAt  string `json:"lastOpenedAt"`
}

type RateLimitWindow struct {
	UsedPercent        int    `json:"usedPercent"`
	WindowDurationMins *int64 `json:"windowDurationMins,omitempty"`
	ResetsAt           *int64 `json:"resetsAt,omitempty"`
}

type RateLimitState struct {
	Primary      *RateLimitWindow     `json:"primary,omitempty"`
	Secondary    *RateLimitWindow     `json:"secondary,omitempty"`
	Status       RateLimitFetchStatus `json:"status"`
	ErrorMessage string               `json:"errorMessage,omitempty"`
}

type LatencyTestState struct {
	Status       LatencyTestStatus     `json:"status"`
	Available    bool                  `json:"available"`
	LatencyMs    *int64                `json:"latencyMs,omitempty"`
	StatusCode   *int                  `json:"statusCode,omitempty"`
	ErrorMessage string                `json:"errorMessage,omitempty"`
	ErrorType    string                `json:"errorType,omitempty"`
	ErrorCode    string                `json:"errorCode,omitempty"`
	CheckedAt    string                `json:"checkedAt,omitempty"`
	History      []LatencyHistoryEntry `json:"history,omitempty"`
}

type ProfileMeta struct {
	ID                   string           `json:"id"`
	Type                 ProfileType      `json:"type"`
	DisplayName          string           `json:"displayName"`
	StableKeyHash        string           `json:"stableKeyHash"`
	Email                string           `json:"email,omitempty"`
	EmailVerified        bool             `json:"emailVerified"`
	PlanType             string           `json:"planType,omitempty"`
	ChatGPTUserID        string           `json:"chatgptUserId,omitempty"`
	ChatGPTAccountID     string           `json:"chatgptAccountId,omitempty"`
	ClientID             string           `json:"clientId,omitempty"`
	BaseURL              string           `json:"baseURL,omitempty"`
	MaskedAPIKey         string           `json:"maskedApiKey,omitempty"`
	Model                string           `json:"model,omitempty"`
	ModelReasoningEffort string           `json:"modelReasoningEffort,omitempty"`
	Source               string           `json:"source"`
	IsActive             bool             `json:"isActive"`
	IsValid              bool             `json:"isValid"`
	ContentHash          string           `json:"contentHash"`
	CreatedAt            string           `json:"createdAt"`
	UpdatedAt            string           `json:"updatedAt"`
	LastRateLimitFetchAt string           `json:"lastRateLimitFetchAt,omitempty"`
	RateLimits           RateLimitState   `json:"rateLimits"`
	LatencyTest          LatencyTestState `json:"latencyTest"`
}

type CurrentProfileState struct {
	Path        string      `json:"path"`
	Available   bool        `json:"available"`
	Managed     bool        `json:"managed"`
	ProfileID   string      `json:"profileId,omitempty"`
	Type        ProfileType `json:"type"`
	DisplayName string      `json:"displayName,omitempty"`
	ContentHash string      `json:"contentHash,omitempty"`
	Error       string      `json:"error,omitempty"`
}

type AppState struct {
	Settings AppSettings         `json:"settings"`
	Current  CurrentProfileState `json:"current"`
	Profiles []ProfileMeta       `json:"profiles"`
}

type APIProfileInput struct {
	BaseURL              string `json:"baseURL"`
	Model                string `json:"model"`
	ModelReasoningEffort string `json:"modelReasoningEffort"`
	ModelContextWindow   string `json:"modelContextWindow"`
	APIKey               string `json:"apiKey"`
}

type UpdateSettingsInput struct {
	CodexHomePath string `json:"codexHomePath"`
}

type storedProfile struct {
	Meta      ProfileMeta
	AuthRaw   string
	ConfigRaw string
}

type profileSnapshot struct {
	Meta      ProfileMeta
	AuthRaw   string
	ConfigRaw string
}
