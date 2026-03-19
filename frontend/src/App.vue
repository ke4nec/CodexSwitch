<template>
  <v-app>
    <div class="window-shell">
      <header class="app-bar">
        <div class="app-bar-inner">
          <div class="brand-block">
            <div class="brand-title">CodexSwitch</div>
          </div>

          <div class="toolbar-spacer" />

          <div class="toolbar-actions">
            <v-btn class="toolbar-btn" :loading="loading" @click="store.loadAppState(true)">刷新</v-btn>
            <v-btn
              class="toolbar-btn"
              :disabled="!officialProfileIds.length"
              :loading="acting"
              @click="store.refreshRateLimits()"
            >
              刷新全部额度
            </v-btn>
            <v-btn
              class="toolbar-btn"
              variant="outlined"
              :loading="importingOfficialFile"
              :disabled="importingOfficialFile"
              @click="store.importOfficialProfileFile()"
            >
              导入账号文件
            </v-btn>
            <v-btn class="toolbar-btn" color="primary" :loading="acting" @click="store.openCreateApiDialog()">
              新增 API 配置
            </v-btn>
            <v-btn class="toolbar-btn" variant="outlined" :disabled="acting" @click="store.openSettingsDialog()">
              设置
            </v-btn>
          </div>
        </div>
      </header>

      <main class="main-shell">
        <div class="page-shell">
          <section class="hero-panel">
            <div class="hero-summary">
              <div class="hero-title">统一管理 Codex 账号与 API 配置</div>
              <div class="hero-subtitle">
                启动自动识别当前配置，支持托管、切换、编辑 API、刷新额度和快速回滚。
              </div>
            </div>

            <div class="hero-stats">
              <div class="stat-card">
                <div class="stat-label">托管配置</div>
                <div class="stat-value">{{ profiles.length }}</div>
              </div>
              <div class="stat-card">
                <div class="stat-label">当前状态</div>
                <div class="stat-value">{{ currentStatus }}</div>
              </div>
            </div>
          </section>

          <v-alert v-if="current.error" type="warning" variant="tonal" class="status-alert">
            {{ current.error }}
          </v-alert>

          <v-alert
            v-else-if="current.available"
            type="info"
            variant="tonal"
            class="status-alert"
          >
            当前目录中的配置已识别为
            <strong>{{ current.displayName || '当前配置' }}</strong>
            ，{{ current.managed ? '已纳入托管。' : '尚未托管，切换前会自动保护。' }}
          </v-alert>

          <section class="table-panel">
            <div class="panel-head">
              <div class="panel-head-copy">
                <div class="panel-title">托管配置列表</div>
              </div>
            </div>

            <v-progress-linear v-if="loading" indeterminate color="primary" class="mb-4" />

            <div v-if="!loading && !profiles.length" class="empty-block">
              <div class="empty-title">还没有托管配置</div>
              <div class="empty-subtitle">可以先新增一个 API 配置，或先让工具自动识别当前配置。</div>
            </div>

            <v-table v-else class="profiles-table">
              <colgroup>
                <col class="col-display-name" />
                <col class="col-type" />
                <col class="col-plan" />
                <col class="col-usage" />
                <col class="col-usage" />
                <col class="col-model" />
                <col class="col-status" />
                <col class="col-updated" />
                <col class="col-actions" />
              </colgroup>
              <thead>
                <tr>
                  <th class="display-name-column">显示名</th>
                  <th class="type-column">类型</th>
                  <th class="plan-column">Plan / URL</th>
                  <th class="usage-column">5h</th>
                  <th class="usage-column">weekly</th>
                  <th class="model-column">模型</th>
                  <th class="status-column">状态</th>
                  <th class="updated-column">最后同步</th>
                  <th class="actions-column">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="profile in profiles"
                  :key="profile.id"
                  :class="{ 'is-active-row': profile.isActive }"
                >
                  <td class="display-name-column">
                    <div class="primary-cell">
                      <div class="primary-name" :title="displayNameText(profile)">
                        {{ displayNameText(profile) }}
                      </div>
                    </div>
                  </td>

                  <td class="type-column">
                    <v-chip size="small" :color="profile.type === 'official' ? 'secondary' : 'primary'" variant="flat">
                      {{ profile.type === 'official' ? '官方' : profile.type === 'api' ? 'API' : '未知' }}
                    </v-chip>
                  </td>

                  <td class="plan-column">
                    <template v-if="planOrURL(profile) !== '-'">
                      <v-tooltip location="top">
                        <template #activator="{ props }">
                          <div
                            v-bind="props"
                            class="plan-cell"
                            @contextmenu.prevent="copyText(planOrURL(profile), '已复制到剪贴板')"
                          >
                            {{ planOrURL(profile) }}
                          </div>
                        </template>
                        <span>{{ planOrURL(profile) }}</span>
                      </v-tooltip>
                    </template>
                    <span v-else class="plan-cell plan-cell-empty">-</span>
                  </td>

                  <td class="usage-column">{{ renderUsage(profile.rateLimits.primary, profile.type) }}</td>
                  <td class="usage-column">{{ renderUsage(profile.rateLimits.secondary, profile.type) }}</td>

                  <td class="model-column">
                    <div class="model-cell" :title="profile.model || '-'">{{ profile.model || '-' }}</div>
                  </td>

                  <td class="status-column">
                    <v-chip size="small" :color="statusColor(profile)" variant="tonal">
                      {{ statusText(profile) }}
                    </v-chip>
                  </td>

                  <td class="updated-column">
                    <div class="updated-cell">
                      <span class="updated-date">{{ formatDateParts(profile.updatedAt).date }}</span>
                      <span class="updated-time">{{ formatDateParts(profile.updatedAt).time }}</span>
                    </div>
                  </td>

                  <td class="actions-column">
                    <div class="row-actions">
                      <v-btn
                        size="small"
                        density="compact"
                        variant="text"
                        class="row-action-btn"
                        :disabled="acting"
                        @click="store.askSwitch(profile.id)"
                      >
                        切换
                      </v-btn>
                      <v-btn
                        v-if="profile.type === 'official'"
                        size="small"
                        density="compact"
                        variant="text"
                        class="row-action-btn"
                        :loading="isProfileRefreshing(profile.id)"
                        :disabled="acting || isProfileRefreshing(profile.id)"
                        @click="store.refreshProfileRateLimit(profile)"
                      >
                        刷新
                      </v-btn>
                      <v-btn
                        v-if="profile.type === 'api'"
                        size="small"
                        density="compact"
                        variant="text"
                        class="row-action-btn"
                        :disabled="acting"
                        @click="store.openEditApiDialog(profile)"
                      >
                        编辑
                      </v-btn>
                      <v-btn
                        size="small"
                        density="compact"
                        variant="text"
                        color="error"
                        class="row-action-btn"
                        :disabled="acting"
                        @click="store.askDelete(profile.id)"
                      >
                        删除
                      </v-btn>
                    </div>
                  </td>
                </tr>
              </tbody>
            </v-table>
          </section>
        </div>
      </main>
    </div>

    <ApiProfileDialog
      v-model="store.apiDialog.open"
      :mode="store.apiDialog.mode"
      :form="store.apiDialog.form"
      :loading="acting"
      @save="store.saveApiProfile"
    />

    <SettingsDialog
      v-model="store.settingsDialog.open"
      :codex-home-path="store.settingsDialog.codexHomePath"
      :loading="acting"
      @save="store.saveSettings"
    />

    <ConfirmDialog
      v-model="store.confirmDialog.open"
      :loading="acting"
      :title="store.confirmDialog.title"
      :text="store.confirmDialog.text"
      :confirm-text="store.confirmDialog.confirmText"
      :color="store.confirmDialog.color"
      @confirm="store.submitConfirm"
    />

    <v-snackbar
      v-model="store.snackbar.show"
      :color="store.snackbar.color"
      location="bottom right"
      timeout="2600"
    >
      {{ store.snackbar.text }}
    </v-snackbar>
  </v-app>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { storeToRefs } from 'pinia';

import ApiProfileDialog from './components/ApiProfileDialog.vue';
import ConfirmDialog from './components/ConfirmDialog.vue';
import SettingsDialog from './components/SettingsDialog.vue';
import { useAppStore } from './stores/app';
import type { ProfileMeta, RateLimitWindow } from './types';

const store = useAppStore();
const { acting, current, importingOfficialFile, loading, officialProfileIds, profiles, refreshingProfileIds } = storeToRefs(store);

const currentStatus = computed(() => {
  if (current.value.error) {
    return '目录异常';
  }
  if (current.value.available && current.value.managed) {
    return '已托管';
  }
  if (current.value.available) {
    return '未托管';
  }
  return '未检测到';
});

onMounted(() => {
  void store.bootstrap();
});

function statusColor(profile: ProfileMeta) {
  if (!profile.isValid) {
    return 'warning';
  }
  if (profile.isActive) {
    return 'success';
  }
  return 'primary';
}

function statusText(profile: ProfileMeta) {
  if (!profile.isValid) {
    return '异常';
  }
  if (profile.isActive) {
    return '激活';
  }
  return '就绪';
}

function renderUsage(window: RateLimitWindow | undefined, type: ProfileMeta['type']) {
  if (type !== 'official') {
    return '-';
  }
  if (!window) {
    return '未获取';
  }
  return `${window.usedPercent}%`;
}

function formatDateParts(value?: string) {
  if (!value) {
    return {
      date: '-',
      time: '--:--',
    };
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return {
      date: value,
      time: '',
    };
  }

  return {
    date: date.toLocaleDateString('zh-CN', {
      year: '2-digit',
      month: '2-digit',
      day: '2-digit',
    }),
    time: date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    }),
  };
}

function planOrURL(profile: ProfileMeta) {
  return profile.type === 'official' ? profile.planType || '-' : profile.baseURL || '-';
}

function displayNameText(profile: ProfileMeta) {
  if (profile.type === 'api' && profile.maskedApiKey) {
    return profile.maskedApiKey.replace(/\*{7,}/g, '**********');
  }

  return profile.displayName.replace(/\*{7,}/g, '**********');
}

function isProfileRefreshing(profileId: string) {
  return refreshingProfileIds.value.includes(profileId);
}

async function copyText(value?: string, message = '已复制到剪贴板') {
  if (!value || value === '-') {
    return;
  }

  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(value);
    } else {
      const textArea = document.createElement('textarea');
      textArea.value = value;
      textArea.setAttribute('readonly', 'true');
      textArea.style.position = 'fixed';
      textArea.style.opacity = '0';
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand('copy');
      document.body.removeChild(textArea);
    }

    store.notify(message);
  } catch (error) {
    store.notify(store.formatError(error), 'error');
  }
}
</script>
