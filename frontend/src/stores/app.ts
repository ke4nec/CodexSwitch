import { defineStore } from 'pinia';

import { runtimeMessageMarkers, translate } from '../i18n';
import { backend } from '../lib/backend';
import type {
  APIProfileInput,
  AppState,
  ProfileMeta,
  ProfileSortKey,
  ProfileSortState,
  RateLimitWindow,
  SortDirection,
} from '../types';

type ApiDialogMode = 'create' | 'edit';
type ConfirmAction = 'switch' | 'delete';

const emptyAppState: AppState = {
  settings: {
    codexHomePath: '',
    lastOpenedAt: '',
  },
  current: {
    path: '',
    available: false,
    managed: false,
    type: 'unknown',
  },
  profiles: [],
};

const defaultApiForm = (): APIProfileInput => ({
  baseURL: 'https://api.openai.com/v1',
  model: 'gpt-5.4',
  modelReasoningEffort: 'xhigh',
  apiKey: '',
});

const activeOfficialProfileRefreshIntervalMs = 5 * 60 * 1000;
const defaultProfileSort: ProfileSortState = {
  key: 'updatedAt',
  direction: 'desc',
};

const defaultSortDirectionByKey: Record<ProfileSortKey, SortDirection> = {
  usage5h: 'desc',
  usageWeekly: 'desc',
  latency: 'asc',
  updatedAt: 'desc',
};

let activeOfficialProfileRefreshTimer: ReturnType<typeof setInterval> | null = null;

function getRemainingUsagePercent(window?: RateLimitWindow) {
  if (!window || typeof window.usedPercent !== 'number' || !Number.isFinite(window.usedPercent)) {
    return null;
  }

  return Math.max(0, 100 - window.usedPercent);
}

function getProfileLatencyMs(profile: ProfileMeta) {
  const latencyMs = profile.latencyTest.latencyMs;
  if (profile.latencyTest.status !== 'success' || typeof latencyMs !== 'number' || latencyMs <= 0) {
    return null;
  }

  return latencyMs;
}

function getProfileUpdatedAtTimestamp(profile: ProfileMeta) {
  const timestamp = new Date(profile.updatedAt).getTime();
  return Number.isNaN(timestamp) ? null : timestamp;
}

function getProfileSortValue(profile: ProfileMeta, key: ProfileSortKey) {
  switch (key) {
    case 'usage5h':
      return getRemainingUsagePercent(profile.rateLimits.primary);
    case 'usageWeekly':
      return getRemainingUsagePercent(profile.rateLimits.secondary);
    case 'latency':
      return getProfileLatencyMs(profile);
    case 'updatedAt':
      return getProfileUpdatedAtTimestamp(profile);
    default:
      return null;
  }
}

function compareNullableNumbers(left: number | null, right: number | null, direction: SortDirection) {
  const leftMissing = left == null || Number.isNaN(left);
  const rightMissing = right == null || Number.isNaN(right);

  if (leftMissing || rightMissing) {
    if (leftMissing && rightMissing) {
      return 0;
    }

    return leftMissing ? 1 : -1;
  }

  return direction === 'asc' ? left - right : right - left;
}

function sortProfilesForDisplay(profiles: ProfileMeta[], sortState: ProfileSortState) {
  return profiles
    .map((profile, index) => ({ profile, index }))
    .sort((left, right) => {
      const leftValue = getProfileSortValue(left.profile, sortState.key);
      const rightValue = getProfileSortValue(right.profile, sortState.key);
      const comparison = compareNullableNumbers(leftValue, rightValue, sortState.direction);
      if (comparison !== 0) {
        return comparison;
      }

      return left.index - right.index;
    })
    .map(({ profile }) => profile);
}

export const useAppStore = defineStore('app', {
  state: () => ({
    appState: emptyAppState as AppState,
    loading: false,
    acting: false,
    importingOfficialFile: false,
    testingAllLatency: false,
    refreshingProfileIds: [] as string[],
    testingLatencyProfileIds: [] as string[],
    profileSort: { ...defaultProfileSort } as ProfileSortState,
    snackbar: {
      show: false,
      text: '',
      color: 'success' as 'success' | 'error' | 'warning',
    },
    apiDialog: {
      open: false,
      mode: 'create' as ApiDialogMode,
      profileId: '',
      form: defaultApiForm(),
    },
    settingsDialog: {
      open: false,
      codexHomePath: '',
    },
    confirmDialog: {
      open: false,
      action: 'switch' as ConfirmAction,
      profileId: '',
      title: '',
      text: '',
      confirmText: '',
      color: 'primary' as 'primary' | 'error',
    },
  }),

  getters: {
    profiles: (state) => sortProfilesForDisplay(state.appState.profiles, state.profileSort),
    current: (state) => state.appState.current,
    settings: (state) => state.appState.settings,
    officialProfileIds: (state) =>
      state.appState.profiles.filter((profile) => profile.type === 'official').map((profile) => profile.id),
    apiProfileIds: (state) =>
      state.appState.profiles.filter((profile) => profile.type === 'api').map((profile) => profile.id),
    latencyProfileIds: (state) =>
      state.appState.profiles
        .filter((profile) => profile.type === 'official' || profile.type === 'api')
        .map((profile) => profile.id),
  },

  actions: {
    toggleProfileSort(key: ProfileSortKey) {
      if (this.profileSort.key === key) {
        this.profileSort = {
          key,
          direction: this.profileSort.direction === 'asc' ? 'desc' : 'asc',
        };
        return;
      }

      this.profileSort = {
        key,
        direction: defaultSortDirectionByKey[key],
      };
    },

    notify(text: string, color: 'success' | 'error' | 'warning' = 'success') {
      this.snackbar = {
        show: true,
        text,
        color,
      };
    },

    async bootstrap() {
      if (!activeOfficialProfileRefreshTimer) {
        activeOfficialProfileRefreshTimer = setInterval(() => {
          void this.refreshActiveOfficialProfile(false);
        }, activeOfficialProfileRefreshIntervalMs);
      }

      await this.loadAppState(false);
      void this.refreshActiveOfficialProfile(false);
    },

    async loadAppState(showSuccess = false) {
      this.loading = true;
      try {
        this.appState = await backend.getAppState();
        if (showSuccess) {
          this.notify(translate('notifications.listRefreshed'));
        }
      } catch (error) {
        this.notify(this.formatError(error), 'error');
      } finally {
        this.loading = false;
      }
    },

    openCreateApiDialog() {
      this.apiDialog = {
        open: true,
        mode: 'create',
        profileId: '',
        form: defaultApiForm(),
      };
    },

    async openEditApiDialog(profile: ProfileMeta) {
      await this.runAction(async () => {
        const form = await backend.getApiProfileInput(profile.id);
        this.apiDialog = {
          open: true,
          mode: 'edit',
          profileId: profile.id,
          form,
        };
      }, false);
    },

    async importOfficialProfileFile() {
      this.importingOfficialFile = true;
      try {
        this.appState = await backend.importOfficialProfileFile();
        this.notify(translate('notifications.officialProfileImported'));
      } catch (error) {
        const message = this.formatError(error);
        if (!this.isFilePickerCancelled(message)) {
          this.notify(message, 'error');
        }
      } finally {
        this.importingOfficialFile = false;
      }
    },

    async saveApiProfile(form: APIProfileInput) {
      await this.runAction(async () => {
        if (this.apiDialog.mode === 'create') {
          this.appState = await backend.createApiProfile(form);
          this.notify(translate('notifications.apiProfileCreated'));
        } else {
          this.appState = await backend.updateApiProfile(this.apiDialog.profileId, form);
          this.notify(translate('notifications.apiProfileUpdated'));
        }
        this.apiDialog.open = false;
      });
    },

    openSettingsDialog() {
      this.settingsDialog = {
        open: true,
        codexHomePath: this.settings.codexHomePath,
      };
    },

    async saveSettings(codexHomePath: string) {
      await this.runAction(async () => {
        this.appState = await backend.updateSettings({ codexHomePath });
        this.settingsDialog.open = false;
        this.notify(translate('notifications.settingsSaved'));
      });
    },

    askSwitch(profileId: string) {
      this.confirmDialog = {
        open: true,
        action: 'switch',
        profileId,
        title: translate('confirm.switchTitle'),
        text: translate('confirm.switchText'),
        confirmText: translate('confirm.switchConfirm'),
        color: 'primary',
      };
    },

    askDelete(profileId: string) {
      this.confirmDialog = {
        open: true,
        action: 'delete',
        profileId,
        title: translate('confirm.deleteTitle'),
        text: translate('confirm.deleteText'),
        confirmText: translate('confirm.deleteConfirm'),
        color: 'error',
      };
    },

    async submitConfirm() {
      if (!this.confirmDialog.profileId) {
        return;
      }

      await this.runAction(async () => {
        if (this.confirmDialog.action === 'switch') {
          this.appState = await backend.switchProfile(this.confirmDialog.profileId);
          this.notify(translate('notifications.switched'));
        } else {
          this.appState = await backend.deleteProfile(this.confirmDialog.profileId);
          this.notify(translate('notifications.deleted'));
        }
        this.confirmDialog.open = false;
      });
    },

    async refreshRateLimits(showSuccess = true) {
      if (this.officialProfileIds.length === 0) {
        return;
      }

      await this.runAction(async () => {
        this.appState = await backend.refreshRateLimits(this.officialProfileIds);
        if (showSuccess) {
          this.notify(translate('notifications.allRateLimitsRefreshed'));
        }
      }, showSuccess);
    },

    async refreshProfileRateLimit(profile: ProfileMeta, showSuccess = true) {
      if (profile.type !== 'official') {
        return;
      }
      if (this.refreshingProfileIds.includes(profile.id)) {
        return;
      }

      this.refreshingProfileIds = [...this.refreshingProfileIds, profile.id];
      try {
        this.appState = await backend.refreshRateLimits([profile.id]);
        if (showSuccess) {
          this.notify(
            translate('notifications.rateLimitRefreshed', {
              name: profile.displayName || translate('common.officialAccount'),
            }),
          );
        }
      } catch (error) {
        this.notify(this.formatError(error), 'error');
      } finally {
        this.refreshingProfileIds = this.refreshingProfileIds.filter((id) => id !== profile.id);
      }
    },

    async refreshActiveOfficialProfile(showSuccess = false) {
      const activeOfficialProfile = this.appState.profiles.find(
        (profile) => profile.isActive && profile.type === 'official',
      );
      if (!activeOfficialProfile) {
        return;
      }

      await this.refreshProfileRateLimit(activeOfficialProfile, showSuccess);
    },

    async testProfileLatency(profile: ProfileMeta, showSuccess = true) {
      if (profile.type !== 'api' && profile.type !== 'official') {
        return;
      }
      if (this.testingLatencyProfileIds.includes(profile.id)) {
        return;
      }

      this.testingLatencyProfileIds = [...this.testingLatencyProfileIds, profile.id];
      try {
        this.appState = await backend.refreshApiLatencyTests([profile.id]);
        if (showSuccess) {
          this.notify(
            translate('notifications.latencyTested', {
              name: profile.displayName || translate('common.account'),
            }),
          );
        }
      } catch (error) {
        this.notify(this.formatError(error), 'error');
      } finally {
        this.testingLatencyProfileIds = this.testingLatencyProfileIds.filter((id) => id !== profile.id);
      }
    },

    async testAllProfileLatency(showSuccess = true) {
      const targets = this.appState.profiles.filter(
        (profile) =>
          (profile.type === 'official' || profile.type === 'api') &&
          !this.testingLatencyProfileIds.includes(profile.id),
      );

      if (targets.length === 0) {
        return;
      }

      this.testingAllLatency = true;
      try {
        await Promise.allSettled(targets.map((profile) => this.testProfileLatency(profile, false)));
        if (showSuccess) {
          this.notify(translate('notifications.allLatencyTested'));
        }
      } finally {
        this.testingAllLatency = false;
      }
    },

    async runAction(action: () => Promise<void>, notifyOnError = true, trackActing = true) {
      if (trackActing) {
        this.acting = true;
      }
      try {
        await action();
      } catch (error) {
        if (notifyOnError) {
          this.notify(this.formatError(error), 'error');
        }
      } finally {
        if (trackActing) {
          this.acting = false;
        }
      }
    },

    formatError(error: unknown) {
      if (error instanceof Error) {
        return error.message;
      }
      return String(error ?? translate('common.unknownError'));
    },

    isFilePickerCancelled(message: string) {
      const candidates = [translate('runtime.filePickerCancelled'), ...runtimeMessageMarkers.filePickerCancelled];

      return candidates.some((candidate) => message.includes(candidate));
    },
  },
});
