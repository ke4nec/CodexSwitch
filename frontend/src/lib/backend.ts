import type { APIProfileInput, AppState, UpdateSettingsInput } from '../types';
import { translate } from '../i18n';

interface BackendAPI {
  GetAppState(): Promise<AppState>;
  ImportCurrentProfile(): Promise<AppState>;
  ImportOfficialProfileFile(): Promise<AppState>;
  CreateApiProfile(input: APIProfileInput): Promise<AppState>;
  UpdateApiProfile(id: string, input: APIProfileInput): Promise<AppState>;
  GetApiProfileInput(id: string): Promise<APIProfileInput>;
  SwitchProfile(id: string): Promise<AppState>;
  DeleteProfile(id: string): Promise<AppState>;
  SetProfileDisabled(id: string, disabled: boolean): Promise<AppState>;
  RefreshRateLimits(ids: string[]): Promise<AppState>;
  RefreshApiLatencyTests(ids: string[]): Promise<AppState>;
  AutoRefreshApiLatencyTests(ids: string[]): Promise<AppState>;
  UpdateSettings(input: UpdateSettingsInput): Promise<AppState>;
}

declare global {
  interface Window {
    go?: {
      main?: {
        App?: BackendAPI;
      };
    };
  }
}

function appBridge(): BackendAPI {
  const bridge = window.go?.main?.App;
  if (!bridge) {
    throw new Error(translate('runtime.wailsNotReady'));
  }
  return bridge;
}

export const backend = {
  getAppState: () => appBridge().GetAppState(),
  importCurrentProfile: () => appBridge().ImportCurrentProfile(),
  importOfficialProfileFile: () => appBridge().ImportOfficialProfileFile(),
  createApiProfile: (input: APIProfileInput) => appBridge().CreateApiProfile(input),
  updateApiProfile: (id: string, input: APIProfileInput) => appBridge().UpdateApiProfile(id, input),
  getApiProfileInput: (id: string) => appBridge().GetApiProfileInput(id),
  switchProfile: (id: string) => appBridge().SwitchProfile(id),
  deleteProfile: (id: string) => appBridge().DeleteProfile(id),
  setProfileDisabled: (id: string, disabled: boolean) => appBridge().SetProfileDisabled(id, disabled),
  refreshRateLimits: (ids: string[]) => appBridge().RefreshRateLimits(ids),
  refreshApiLatencyTests: (ids: string[]) => appBridge().RefreshApiLatencyTests(ids),
  autoRefreshApiLatencyTests: (ids: string[]) => appBridge().AutoRefreshApiLatencyTests(ids),
  updateSettings: (input: UpdateSettingsInput) => appBridge().UpdateSettings(input),
};
