<template>
  <v-app>
    <v-app-bar flat class="app-bar">
      <div class="brand-block">
        <div class="eyebrow">Codex configuration cockpit</div>
        <div class="brand-title">CodexSwitch</div>
      </div>

      <div class="path-badge">
        目标目录
        <span>{{ settings.codexHomePath || '未设置' }}</span>
      </div>

      <v-spacer />

      <div class="toolbar-actions">
        <v-btn :loading="loading" @click="store.loadAppState(true)">刷新</v-btn>
        <v-btn :disabled="!officialProfileIds.length" :loading="acting" @click="store.refreshRateLimits()">
          刷新额度
        </v-btn>
        <v-btn :loading="acting" @click="store.importCurrentProfile()">导入当前配置</v-btn>
        <v-btn color="primary" :loading="acting" @click="store.openCreateApiDialog()">新增 API 配置</v-btn>
        <v-btn variant="outlined" :disabled="acting" @click="store.openSettingsDialog()">设置</v-btn>
      </div>
    </v-app-bar>

    <v-main>
      <v-container fluid class="page-shell">
        <section class="hero-panel">
          <div class="hero-copy">
            <div class="hero-title">把官方账号和 API 配置都收拢到一个切换台里。</div>
            <div class="hero-subtitle">
              启动会自动识别当前配置，手动导入、编辑 API、切换和删除都在同一个页面完成。
            </div>
          </div>
          <div class="hero-stats">
            <div class="stat-card">
              <div class="stat-label">托管配置</div>
              <div class="stat-value">{{ profiles.length }}</div>
            </div>
            <div class="stat-card">
              <div class="stat-label">当前状态</div>
              <div class="stat-value stat-sm">{{ currentStatus }}</div>
            </div>
          </div>
        </section>

        <v-alert
          v-if="current.error"
          type="warning"
          variant="tonal"
          class="status-alert"
        >
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
            <div>
              <div class="panel-title">托管配置列表</div>
              <div class="panel-subtitle">官方配置展示 Plan 和额度，API 配置展示 Base URL 和脱敏后的 Key。</div>
            </div>
            <div class="panel-meta">最近打开：{{ formatDate(settings.lastOpenedAt) }}</div>
          </div>

          <v-progress-linear v-if="loading" indeterminate color="primary" class="mb-4" />

          <div v-if="!loading && !profiles.length" class="empty-block">
            <div class="empty-title">还没有托管配置</div>
            <div class="empty-subtitle">可以先导入当前配置，或者直接新增一个 API 配置。</div>
          </div>

          <v-table v-else class="profiles-table">
            <thead>
              <tr>
                <th>显示名</th>
                <th>类型</th>
                <th>Plan / URL</th>
                <th>5h usage</th>
                <th>weekly usage</th>
                <th>模型</th>
                <th>状态</th>
                <th>最后同步</th>
                <th class="actions-column">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="profile in profiles" :key="profile.id" :class="{ 'is-active-row': profile.isActive }">
                <td>
                  <div class="primary-cell">
                    <div class="primary-name">{{ profile.displayName }}</div>
                    <div class="secondary-line">
                      <span v-if="profile.email">{{ profile.email }}</span>
                      <span v-else-if="profile.maskedApiKey">{{ profile.maskedApiKey }}</span>
                      <span v-else>{{ profile.id }}</span>
                    </div>
                  </div>
                </td>
                <td>
                  <v-chip size="small" :color="profile.type === 'official' ? 'secondary' : 'primary'" variant="flat">
                    {{ profile.type === 'official' ? '官方' : profile.type === 'api' ? 'API' : '未知' }}
                  </v-chip>
                </td>
                <td>{{ profile.type === 'official' ? profile.planType || '-' : profile.baseURL || '-' }}</td>
                <td>{{ renderUsage(profile.rateLimits.primary, profile.type) }}</td>
                <td>{{ renderUsage(profile.rateLimits.secondary, profile.type) }}</td>
                <td>{{ profile.model || '-' }}</td>
                <td>
                  <v-chip size="small" :color="statusColor(profile)" variant="tonal">
                    {{ statusText(profile) }}
                  </v-chip>
                </td>
                <td>{{ formatDate(profile.updatedAt) }}</td>
                <td class="actions-column">
                  <div class="row-actions">
                    <v-btn size="small" variant="text" :disabled="acting" @click="store.askSwitch(profile.id)">切换</v-btn>
                    <v-btn
                      v-if="profile.type === 'api'"
                      size="small"
                      variant="text"
                      :disabled="acting"
                      @click="store.openEditApiDialog(profile)"
                    >
                      编辑
                    </v-btn>
                    <v-btn size="small" variant="text" color="error" :disabled="acting" @click="store.askDelete(profile.id)">
                      删除
                    </v-btn>
                  </div>
                </td>
              </tr>
            </tbody>
          </v-table>
        </section>
      </v-container>
    </v-main>

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
const { acting, current, loading, officialProfileIds, profiles, settings } = storeToRefs(store);

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
  const duration = window.windowDurationMins ? ` / ${window.windowDurationMins}m` : '';
  return `${window.usedPercent}%${duration}`;
}

function formatDate(value?: string) {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}
</script>
