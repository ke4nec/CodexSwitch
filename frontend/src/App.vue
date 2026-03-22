<template>
  <v-app>
    <div class="window-shell">
      <header class="app-bar">
        <div class="app-bar-inner">
          <div class="brand-block">
            <v-menu open-on-hover location="bottom start" offset="10">
              <template #activator="{ props }">
                <div v-bind="props" class="brand-title brand-title-trigger">CodexSwitch</div>
              </template>

              <v-card class="brand-hover-card">
                <div class="brand-hover-label">CodexSwitch</div>
                <div class="brand-hover-title">{{ t('brand.title') }}</div>
                <div class="brand-hover-subtitle">
                  {{ t('brand.subtitle') }}
                </div>
              </v-card>
            </v-menu>
          </div>

          <div class="toolbar-spacer" />

          <div class="toolbar-actions">
            <v-btn class="toolbar-btn" :loading="loading" @click="store.loadAppState(true)">
              {{ t('toolbar.refresh') }}
            </v-btn>
            <v-btn
              class="toolbar-btn"
              :disabled="!officialProfileIds.length"
              :loading="acting"
              @click="store.refreshRateLimits()"
            >
              {{ t('toolbar.refreshAllRateLimits') }}
            </v-btn>
            <v-btn
              class="toolbar-btn"
              :disabled="!latencyProfileIds.length || testingAllLatency"
              :loading="testingAllLatency"
              @click="store.testAllProfileLatency()"
            >
              {{ t('toolbar.testAll') }}
            </v-btn>
            <v-btn
              class="toolbar-btn"
              variant="outlined"
              :loading="importingOfficialFile"
              :disabled="importingOfficialFile"
              @click="store.importOfficialProfileFile()"
            >
              {{ t('toolbar.importAccountFile') }}
            </v-btn>
            <v-btn class="toolbar-btn" color="primary" :loading="acting" @click="store.openCreateApiDialog()">
              {{ t('toolbar.addApiProfile') }}
            </v-btn>
            <v-btn class="toolbar-btn" variant="outlined" :disabled="acting" @click="store.openSettingsDialog()">
              {{ t('toolbar.settings') }}
            </v-btn>
            <v-menu location="bottom end" offset="10">
              <template #activator="{ props }">
                <v-btn
                  v-bind="props"
                  icon
                  variant="text"
                  class="toolbar-icon-btn locale-trigger-btn"
                  :aria-label="localeMenuLabel"
                  :title="localeMenuLabel"
                >
                  <span class="locale-trigger-icon" aria-hidden="true">
                    <span class="locale-trigger-globe">
                      <span class="locale-trigger-globe-line locale-trigger-globe-line-horizontal" />
                      <span class="locale-trigger-globe-line locale-trigger-globe-line-vertical" />
                    </span>
                    <span class="locale-trigger-badge">{{ localeBadge }}</span>
                  </span>
                </v-btn>
              </template>

              <v-list density="compact" nav>
                <v-list-item
                  v-for="option in localeOptions"
                  :key="option.value"
                  :active="locale === option.value"
                  @click="setLocale(option.value)"
                >
                  <v-list-item-title>{{ option.label }}</v-list-item-title>
                </v-list-item>
              </v-list>
            </v-menu>
          </div>
        </div>
      </header>

      <main class="main-shell">
        <div class="page-shell">
          <section class="hero-panel overview-panel">
            <div class="overview-status">
              <div class="overview-label">{{ t('overview.currentWorkspace') }}</div>
              <v-alert :type="workspaceAlert.type" variant="tonal" class="status-alert embedded-status-alert">
                <div class="embedded-status-title">{{ workspaceAlert.title }}</div>
                <div class="embedded-status-text">{{ workspaceAlert.text }}</div>
                <div v-if="workspaceAlert.meta" class="embedded-status-meta" :title="workspaceAlert.meta">
                  {{ workspaceAlert.meta }}
                </div>
              </v-alert>
            </div>

            <div class="hero-stats overview-stats">
              <div class="stat-card">
                <div class="stat-label">{{ t('overview.managedProfiles') }}</div>
                <div class="stat-value">{{ profiles.length }}</div>
                <div class="stat-note">
                  {{ t('overview.managedProfilesNote', { official: officialProfileIds.length, latency: latencyProfileIds.length }) }}
                </div>
              </div>
              <div class="stat-card">
                <div class="stat-label">{{ t('overview.currentStatus') }}</div>
                <div class="stat-value stat-value-text">{{ currentStatus }}</div>
                <div class="stat-note" :title="current.displayName || t('overview.pendingCurrentProfile')">
                  {{ current.displayName || t('overview.pendingCurrentProfile') }}
                </div>
              </div>
            </div>
          </section>

          <section class="table-panel">
            <div class="panel-head">
              <div class="panel-head-copy">
                <div class="panel-title">{{ t('profiles.title') }}</div>
              </div>
            </div>

            <v-progress-linear v-if="loading" indeterminate color="primary" class="mb-4" />

            <div v-if="!loading && !profiles.length" class="empty-block">
              <div class="empty-title">{{ t('profiles.emptyTitle') }}</div>
              <div class="empty-subtitle">{{ t('profiles.emptySubtitle') }}</div>
            </div>

            <div v-else class="profiles-table">
              <div ref="profilesTableHeadRef" class="profiles-table-head">
                <table>
                  <colgroup>
                    <col class="col-display-name" />
                    <col class="col-type" />
                    <col class="col-plan" />
                    <col class="col-usage" />
                    <col class="col-usage" />
                    <col class="col-model" />
                    <col class="col-status" />
                    <col class="col-latency" />
                    <col class="col-updated" />
                    <col class="col-actions" />
                  </colgroup>
                  <thead>
                    <tr>
                      <th class="display-name-column">{{ t('profiles.headers.displayName') }}</th>
                      <th class="type-column">{{ t('profiles.headers.type') }}</th>
                      <th class="plan-column">{{ t('profiles.headers.planOrUrl') }}</th>
                      <th class="usage-column">{{ t('profiles.headers.usage5h') }}</th>
                      <th class="usage-column">{{ t('profiles.headers.usageWeekly') }}</th>
                      <th class="model-column">{{ t('profiles.headers.model') }}</th>
                      <th class="status-column">{{ t('profiles.headers.status') }}</th>
                      <th class="latency-column">{{ t('profiles.headers.latency') }}</th>
                      <th class="updated-column">{{ t('profiles.headers.updatedAt') }}</th>
                      <th class="actions-column">{{ t('profiles.headers.actions') }}</th>
                    </tr>
                  </thead>
                </table>
              </div>

              <div ref="profilesTableBodyRef" class="profiles-table-body" @scroll="syncProfilesTableScroll">
                <table :aria-label="t('profiles.tableLabel')">
                  <colgroup>
                    <col class="col-display-name" />
                    <col class="col-type" />
                    <col class="col-plan" />
                    <col class="col-usage" />
                    <col class="col-usage" />
                    <col class="col-model" />
                    <col class="col-status" />
                    <col class="col-latency" />
                    <col class="col-updated" />
                    <col class="col-actions" />
                  </colgroup>
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
                          {{ profileTypeText(profile.type) }}
                        </v-chip>
                      </td>

                      <td class="plan-column">
                        <template v-if="planOrURL(profile) !== EMPTY_VALUE">
                          <v-tooltip location="top">
                            <template #activator="{ props }">
                              <div
                                v-bind="props"
                                class="plan-cell"
                                @contextmenu.prevent="copyText(planOrURL(profile), t('common.copied'))"
                              >
                                <template v-if="profile.type === 'api'">
                                  <span
                                    class="plan-host"
                                    :class="`plan-host-${apiURLDisplay(profile).protocolTone}`"
                                  >
                                    {{ apiURLDisplay(profile).host }}
                                  </span>
                                </template>
                                <span v-else class="plan-text">{{ planOrURL(profile) }}</span>
                              </div>
                            </template>
                            <span>{{ planOrURL(profile) }}</span>
                          </v-tooltip>
                        </template>
                        <span v-else class="plan-cell plan-cell-empty">{{ EMPTY_VALUE }}</span>
                      </td>

                      <td class="usage-column">{{ renderUsage(profile.rateLimits.primary, profile.type) }}</td>
                      <td class="usage-column">{{ renderUsage(profile.rateLimits.secondary, profile.type) }}</td>

                      <td class="model-column">
                        <div class="model-cell" :title="profile.model || EMPTY_VALUE">
                          {{ profile.model || EMPTY_VALUE }}
                        </div>
                      </td>

                      <td class="status-column">
                        <v-chip size="small" :color="statusColor(profile)" variant="tonal">
                          {{ statusText(profile) }}
                        </v-chip>
                      </td>

                      <td class="latency-column">
                        <template v-if="profile.type === 'official' || profile.type === 'api'">
                          <v-tooltip location="top" :disabled="!latencyTooltip(profile)">
                            <template #activator="{ props }">
                              <div v-bind="props" class="latency-cell">
                                <v-chip size="small" :color="latencyColor(profile)" variant="tonal">
                                  {{ latencyPrimaryText(profile) }}
                                </v-chip>
                                <span class="latency-hint">{{ latencySecondaryText(profile) }}</span>
                              </div>
                            </template>
                            <span>{{ latencyTooltip(profile) }}</span>
                          </v-tooltip>
                        </template>
                        <span v-else class="latency-empty">{{ EMPTY_VALUE }}</span>
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
                            {{ t('profiles.actions.switch') }}
                          </v-btn>
                          <v-btn
                            v-if="profile.type === 'official' || profile.type === 'api'"
                            size="small"
                            density="compact"
                            variant="text"
                            class="row-action-btn"
                            :loading="isProfileLatencyTesting(profile.id)"
                            :disabled="acting || isProfileLatencyTesting(profile.id)"
                            @click="store.testProfileLatency(profile)"
                          >
                            {{ t('profiles.actions.test') }}
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
                            {{ t('profiles.actions.refresh') }}
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
                            {{ t('profiles.actions.edit') }}
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
                            {{ t('profiles.actions.delete') }}
                          </v-btn>
                        </div>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
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
import { computed, onMounted, ref } from 'vue';
import { storeToRefs } from 'pinia';

import ApiProfileDialog from './components/ApiProfileDialog.vue';
import ConfirmDialog from './components/ConfirmDialog.vue';
import SettingsDialog from './components/SettingsDialog.vue';
import { useI18n } from './i18n';
import { useAppStore } from './stores/app';
import type { ProfileMeta, RateLimitWindow } from './types';

const EMPTY_VALUE = '-';

const store = useAppStore();
const {
  acting,
  current,
  importingOfficialFile,
  latencyProfileIds,
  loading,
  officialProfileIds,
  profiles,
  refreshingProfileIds,
  testingAllLatency,
  testingLatencyProfileIds,
} = storeToRefs(store);
const { locale, localeName, setLocale, t } = useI18n();

const localeOptions = computed(() => [
  {
    value: 'zh-CN' as const,
    label: t('common.chinese'),
  },
  {
    value: 'en-US' as const,
    label: t('common.english'),
  },
]);

const localeBadge = computed(() => (locale.value === 'zh-CN' ? '中' : 'EN'));
const localeMenuLabel = computed(() => `${t('toolbar.languageLabel')}: ${localeName.value}`);

const currentStatus = computed(() => {
  if (current.value.error) {
    return t('workspace.directoryError');
  }
  if (current.value.available && current.value.managed) {
    return t('workspace.managed');
  }
  if (current.value.available) {
    return t('workspace.unmanaged');
  }
  return t('workspace.notDetected');
});

const workspaceAlert = computed(() => {
  const pathText = current.value.path ? t('workspace.targetPath', { path: current.value.path }) : '';

  if (current.value.error) {
    return {
      type: 'warning' as const,
      title: t('workspace.errorTitle'),
      text: current.value.error,
      meta: pathText,
    };
  }

  if (current.value.available && current.value.managed) {
    return {
      type: 'success' as const,
      title: t('workspace.managedTitle', {
        name: current.value.displayName || t('workspace.currentConfig'),
      }),
      text: t('workspace.managedText'),
      meta: pathText,
    };
  }

  if (current.value.available) {
    return {
      type: 'info' as const,
      title: t('workspace.availableTitle', {
        name: current.value.displayName || t('workspace.currentConfig'),
      }),
      text: t('workspace.availableText'),
      meta: pathText,
    };
  }

  return {
    type: 'info' as const,
    title: t('workspace.emptyTitle'),
    text: t('workspace.emptyText'),
    meta: pathText,
  };
});

const profilesTableHeadRef = ref<HTMLDivElement | null>(null);
const profilesTableBodyRef = ref<HTMLDivElement | null>(null);

onMounted(() => {
  void store.bootstrap();
});

function syncProfilesTableScroll() {
  if (!profilesTableHeadRef.value || !profilesTableBodyRef.value) {
    return;
  }

  profilesTableHeadRef.value.scrollLeft = profilesTableBodyRef.value.scrollLeft;
}

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
    return t('profiles.status.invalid');
  }
  if (profile.isActive) {
    return t('profiles.status.active');
  }
  return t('profiles.status.ready');
}

function renderUsage(window: RateLimitWindow | undefined, type: ProfileMeta['type']) {
  if (type !== 'official') {
    return EMPTY_VALUE;
  }
  if (!window) {
    return t('profiles.usage.unavailable');
  }
  return `${Math.max(0, 100 - window.usedPercent)}%`;
}

function latencyColor(profile: ProfileMeta) {
  if (profile.type !== 'official' && profile.type !== 'api') {
    return 'default';
  }
  if (isProfileLatencyTesting(profile.id)) {
    return 'primary';
  }
  if (profile.latencyTest.status === 'error') {
    return 'warning';
  }
  if (profile.latencyTest.status === 'idle') {
    return 'default';
  }
  return profile.latencyTest.available ? 'success' : 'error';
}

function latencyPrimaryText(profile: ProfileMeta) {
  if (profile.type !== 'official' && profile.type !== 'api') {
    return EMPTY_VALUE;
  }
  if (isProfileLatencyTesting(profile.id)) {
    return t('profiles.latency.testing');
  }
  if (profile.latencyTest.status === 'error') {
    return t('profiles.latency.failed');
  }
  if (profile.latencyTest.status === 'idle') {
    return t('profiles.latency.idle');
  }
  if (typeof profile.latencyTest.latencyMs === 'number' && profile.latencyTest.latencyMs > 0) {
    return `${profile.latencyTest.latencyMs} ms`;
  }
  return t('profiles.latency.responded');
}

function latencySecondaryText(profile: ProfileMeta) {
  if (profile.type !== 'official' && profile.type !== 'api') {
    return EMPTY_VALUE;
  }
  if (isProfileLatencyTesting(profile.id)) {
    return t('profiles.latency.pleaseWait');
  }
  if (profile.latencyTest.status === 'error') {
    return t('profiles.latency.connectionError');
  }
  if (profile.latencyTest.status === 'idle') {
    return t('profiles.latency.clickToTest');
  }
  if (profile.latencyTest.available) {
    return t('profiles.latency.available');
  }
  if (profile.latencyTest.statusCode) {
    return t('profiles.latency.unavailableWithStatus', { statusCode: profile.latencyTest.statusCode });
  }
  return t('profiles.latency.unavailable');
}

function latencyTooltip(profile: ProfileMeta) {
  if (profile.type !== 'official' && profile.type !== 'api') {
    return '';
  }
  if (isProfileLatencyTesting(profile.id)) {
    return t('profiles.latency.runningTooltip');
  }

  const checkedAt = profile.latencyTest.checkedAt
    ? t('profiles.latency.checkedAt', { time: formatDateTime(profile.latencyTest.checkedAt) })
    : '';
  if (profile.latencyTest.status === 'idle') {
    return profile.type === 'official'
      ? t('profiles.latency.officialIdleTooltip')
      : t('profiles.latency.apiIdleTooltip');
  }
  if (profile.latencyTest.status === 'error') {
    return [checkedAt, profile.latencyTest.errorMessage || t('profiles.latency.failedDefault')]
      .filter(Boolean)
      .join(' | ');
  }

  const statusCode = profile.latencyTest.statusCode ? `HTTP ${profile.latencyTest.statusCode}` : '';
  const message =
    profile.latencyTest.errorMessage ||
    (profile.latencyTest.available ? t('profiles.latency.accountAvailable') : t('profiles.latency.accountUnavailable'));
  return [checkedAt, statusCode, message].filter(Boolean).join(' | ');
}

function formatDateParts(value?: string) {
  if (!value) {
    return {
      date: EMPTY_VALUE,
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
    date: date.toLocaleDateString(locale.value, {
      year: '2-digit',
      month: '2-digit',
      day: '2-digit',
    }),
    time: date.toLocaleTimeString(locale.value, {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    }),
  };
}

function formatDateTime(value?: string) {
  if (!value) {
    return EMPTY_VALUE;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return `${date.toLocaleDateString(locale.value, {
    year: '2-digit',
    month: '2-digit',
    day: '2-digit',
  })} ${date.toLocaleTimeString(locale.value, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })}`;
}

type URLProtocolTone = 'http' | 'https' | 'other';

interface URLDisplay {
  host: string;
  protocolTone: URLProtocolTone;
}

function apiURLDisplay(profile: ProfileMeta): URLDisplay {
  return formatURLDisplay(profile.baseURL);
}

function formatURLDisplay(value?: string): URLDisplay {
  const fullValue = value?.trim();
  if (!fullValue) {
    return {
      host: EMPTY_VALUE,
      protocolTone: 'other',
    };
  }

  const fallbackHost = fullValue.replace(/^[a-z][a-z\d+.-]*:\/\//i, '').split(/[/?#]/, 1)[0] || fullValue;

  try {
    const parsed = new URL(fullValue);
    const normalizedProtocol = parsed.protocol.replace(/:$/, '').toLowerCase();

    return {
      host: parsed.host || fallbackHost,
      protocolTone: normalizedProtocol === 'http' || normalizedProtocol === 'https' ? normalizedProtocol : 'other',
    };
  } catch {
    const matched = fullValue.match(/^(https?):\/\/([^/?#]+)/i);
    if (matched) {
      const protocol = matched[1].toLowerCase() as Extract<URLProtocolTone, 'http' | 'https'>;

      return {
        host: matched[2],
        protocolTone: protocol,
      };
    }

    return {
      host: fallbackHost,
      protocolTone: 'other',
    };
  }
}

function planOrURL(profile: ProfileMeta) {
  return profile.type === 'official' ? profile.planType || EMPTY_VALUE : profile.baseURL || EMPTY_VALUE;
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

function isProfileLatencyTesting(profileId: string) {
  return testingLatencyProfileIds.value.includes(profileId);
}

function profileTypeText(type: ProfileMeta['type']) {
  if (type === 'official') {
    return t('profiles.type.official');
  }
  if (type === 'api') {
    return t('profiles.type.api');
  }
  return t('profiles.type.unknown');
}

async function copyText(value?: string, message = t('common.copied')) {
  if (!value || value === EMPTY_VALUE) {
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
