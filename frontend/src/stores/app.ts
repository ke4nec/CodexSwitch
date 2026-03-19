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

export const useAppStore = defineStore('app', {
  state: () => ({
    appState: emptyAppState as AppState,
    loading: false,
    acting: false,
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
    profiles: (state) => state.appState.profiles,
    current: (state) => state.appState.current,
    settings: (state) => state.appState.settings,
    officialProfileIds: (state) =>
      state.appState.profiles.filter((profile) => profile.type === 'official').map((profile) => profile.id),
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
      await this.loadAppState(false);
      if (this.officialProfileIds.length > 0) {
        void this.refreshRateLimits(false);
      }
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

    async importCurrentProfile() {
      await this.runAction(async () => {
        this.appState = await backend.importCurrentProfile();
        this.notify('当前配置已导入');
      });
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

    closeApiDialog() {
      this.apiDialog.open = false;
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

    closeSettingsDialog() {
      this.settingsDialog.open = false;
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

    closeConfirmDialog() {
      this.confirmDialog.open = false;
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
          this.notify('额度信息已刷新');
        }
      }, showSuccess);
    },

    async runAction(action: () => Promise<void>, notifyOnError = true) {
      this.acting = true;
      try {
        await action();
      } catch (error) {
        if (notifyOnError) {
          this.notify(this.formatError(error), 'error');
        }
      } finally {
        this.acting = false;
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
