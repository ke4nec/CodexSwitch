import { defineStore } from 'pinia';

import { backend } from '../lib/backend';
import type { APIProfileInput, AppState, ProfileMeta } from '../types';

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

let activeOfficialProfileRefreshTimer: ReturnType<typeof setInterval> | null = null;

function profileLatencySortBucket(profile: ProfileMeta) {
  const latencyMs = profile.latencyTest.latencyMs;
  if (profile.latencyTest.status !== 'success' || typeof latencyMs !== 'number' || latencyMs <= 0) {
    return 2;
  }
  return profile.latencyTest.available ? 0 : 1;
}

function sortProfilesForDisplay(profiles: ProfileMeta[]) {
  return profiles
    .map((profile, index) => ({ profile, index }))
    .sort((left, right) => {
      const leftBucket = profileLatencySortBucket(left.profile);
      const rightBucket = profileLatencySortBucket(right.profile);
      if (leftBucket !== rightBucket) {
        return leftBucket - rightBucket;
      }

      if (leftBucket < 2) {
        const leftLatency = left.profile.latencyTest.latencyMs ?? Number.MAX_SAFE_INTEGER;
        const rightLatency = right.profile.latencyTest.latencyMs ?? Number.MAX_SAFE_INTEGER;
        if (leftLatency !== rightLatency) {
          return leftLatency - rightLatency;
        }
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
    profiles: (state) => sortProfilesForDisplay(state.appState.profiles),
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
          this.notify('列表已刷新');
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
        this.notify('官方账号文件已导入');
      } catch (error) {
        const message = this.formatError(error);
        if (!message.includes('已取消文件选择')) {
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
          this.notify('API 配置已创建');
        } else {
          this.appState = await backend.updateApiProfile(this.apiDialog.profileId, form);
          this.notify('API 配置已更新');
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
        this.notify('设置已保存并完成重扫');
      });
    },

    askSwitch(profileId: string) {
      this.confirmDialog = {
        open: true,
        action: 'switch',
        profileId,
        title: '切换配置',
        text: '切换前会先保护当前配置，并把目标配置写入目标 Codex 目录。',
        confirmText: '确认切换',
        color: 'primary',
      };
    },

    askDelete(profileId: string) {
      this.confirmDialog = {
        open: true,
        action: 'delete',
        profileId,
        title: '删除配置',
        text: '删除只会影响 CodexSwitch 的托管仓库，不会主动清空目标 Codex 目录。',
        confirmText: '确认删除',
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
          this.notify('配置切换成功');
        } else {
          this.appState = await backend.deleteProfile(this.confirmDialog.profileId);
          this.notify('配置已删除');
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
          this.notify('全部官方账号额度已刷新');
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
          this.notify(`${profile.displayName || '官方账号'} 额度已刷新`);
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
          this.notify(`${profile.displayName || '账号'} 延迟已测试`);
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
          this.notify('全部账号延迟已测试');
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
      return String(error ?? '未知错误');
    },
  },
});
