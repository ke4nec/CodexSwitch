export type ProfileType = 'official' | 'api' | 'unknown';
export type RateLimitFetchStatus = 'idle' | 'loading' | 'success' | 'stale' | 'error';
export type LatencyTestStatus = 'idle' | 'success' | 'error';
export type ProfileSortKey = 'usage5h' | 'usageWeekly' | 'latency' | 'updatedAt';
export type SortDirection = 'asc' | 'desc';

export interface ProfileSortState {
  key: ProfileSortKey;
  direction: SortDirection;
}

export interface AppSettings {
  codexHomePath: string;
  lastOpenedAt: string;
}

export interface RateLimitWindow {
  usedPercent: number;
  windowDurationMins?: number;
  resetsAt?: number;
}

export interface RateLimitState {
  primary?: RateLimitWindow;
  secondary?: RateLimitWindow;
  status: RateLimitFetchStatus;
  errorMessage?: string;
}

export interface LatencyTestState {
  status: LatencyTestStatus;
  available: boolean;
  latencyMs?: number;
  statusCode?: number;
  errorMessage?: string;
  errorType?: string;
  errorCode?: string;
  checkedAt?: string;
  history?: LatencyHistoryEntry[];
}

export interface LatencyHistoryEntry {
  status: LatencyTestStatus;
  available: boolean;
  latencyMs?: number;
  statusCode?: number;
  errorMessage?: string;
  errorType?: string;
  errorCode?: string;
  checkedAt?: string;
}

export interface ProfileMeta {
  id: string;
  type: ProfileType;
  displayName: string;
  stableKeyHash: string;
  disabled: boolean;
  email?: string;
  emailVerified: boolean;
  planType?: string;
  chatgptUserId?: string;
  chatgptAccountId?: string;
  clientId?: string;
  baseURL?: string;
  maskedApiKey?: string;
  model?: string;
  modelReasoningEffort?: string;
  source: string;
  isActive: boolean;
  isValid: boolean;
  contentHash: string;
  createdAt: string;
  updatedAt: string;
  lastRateLimitFetchAt?: string;
  rateLimits: RateLimitState;
  latencyTest: LatencyTestState;
}

export interface CurrentProfileState {
  path: string;
  available: boolean;
  managed: boolean;
  profileId?: string;
  type: ProfileType;
  displayName?: string;
  contentHash?: string;
  error?: string;
}

export interface AppState {
  settings: AppSettings;
  current: CurrentProfileState;
  profiles: ProfileMeta[];
}

export interface APIProfileInput {
  baseURL: string;
  model: string;
  modelReasoningEffort: string;
  modelContextWindow: string;
  apiKey: string;
}

export interface UpdateSettingsInput {
  codexHomePath: string;
}
