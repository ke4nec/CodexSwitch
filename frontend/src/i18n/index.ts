import { computed, ref } from 'vue';

export type AppLocale = 'zh-CN' | 'en-US';

type TranslationParams = Record<string, number | string | undefined>;

type MessageTree = {
  [key: string]: MessageTree | string;
};

const STORAGE_KEY = 'codexswitch.locale';

const messages: Record<AppLocale, MessageTree> = {
  'zh-CN': {
    toolbar: {
      refresh: '刷新',
      refreshAllRateLimits: '刷新全部额度',
      testAll: '全部测试',
      importAccountFile: '导入账号文件',
      addApiProfile: '新增 API 配置',
      settings: '设置',
      languageLabel: '语言',
      localeName: '中文',
    },
    brand: {
      title: '统一管理 Codex 账号与 API 配置',
      subtitle: '启动自动识别当前配置，支持托管、切换、编辑 API、刷新额度和快速回滚。',
    },
    overview: {
      currentWorkspace: '当前工作区',
      managedProfiles: '托管配置',
      managedProfilesNote: '官方 {official} · 可测速 {latency}',
      currentStatus: '当前状态',
      pendingCurrentProfile: '等待识别当前配置',
    },
    workspace: {
      targetPath: '目标目录：{path}',
      directoryError: '目录异常',
      managed: '已托管',
      unmanaged: '未托管',
      notDetected: '未检测到',
      errorTitle: '当前目录状态异常',
      managedTitle: '已识别并托管 {name}',
      managedText: '当前目录已经接入托管流，可以直接切换、刷新额度和执行延迟测试。',
      availableTitle: '已识别 {name}',
      availableText: '当前配置尚未纳入托管，切换前会自动保护现场并写入托管仓库。',
      emptyTitle: '当前目录还没有检测到配置',
      emptyText: '可以先新增 API 配置或导入账号文件，开始接入第一个可切换配置。',
      currentConfig: '当前配置',
    },
    profiles: {
      title: '托管配置列表',
      emptyTitle: '还没有托管配置',
      emptySubtitle: '可以先新增一个 API 配置，或先让工具自动识别当前配置。',
      tableLabel: '托管配置列表',
      headers: {
        displayName: '显示名',
        type: '类型',
        planOrUrl: '套餐 / URL',
        usage5h: '5 小时',
        usageWeekly: '每周',
        model: '模型',
        status: '状态',
        latency: '延迟测试',
        updatedAt: '最后同步',
        actions: '操作',
      },
      type: {
        official: '官方',
        api: 'API',
        unknown: '未知',
      },
      status: {
        invalid: '异常',
        disabled: '禁用',
        active: '激活',
        ready: '就绪',
      },
      usage: {
        unavailable: '未获取',
      },
      latency: {
        testing: '测试中',
        failed: '测试失败',
        idle: '未测试',
        responded: '已返回',
        pleaseWait: '请稍候',
        connectionError: '连接异常',
        clickToTest: '点击测试',
        available: '可用',
        unavailableWithStatus: '不可用 · {statusCode}',
        unavailable: '不可用',
        runningTooltip: '正在进行延迟测试，完成后会自动刷新当前账号结果',
        checkedAt: '最后测试：{time}',
        officialIdleTooltip: '通过 GET /v1/models 测试当前官方账号的响应延迟和可用性',
        apiIdleTooltip: '通过 POST /responses 发送 "hi" 测试当前 API Key 的响应延迟和可用性',
        failedDefault: '延迟测试失败',
        accountAvailable: '账号可用',
        accountUnavailable: '账号不可用',
        tooltipResult: '结果：{result}',
        tooltipLatency: '延迟：{latency}',
        tooltipType: 'type：{type}',
        tooltipMessage: 'message：{message}',
        tooltipCode: 'code：{code}',
        tooltipResultAvailable: '可用',
        tooltipResultUnavailable: '不可用',
        tooltipResultFailed: '测试失败',
      },
      actions: {
        switch: '切换',
        test: '测试',
        refresh: '刷新',
        edit: '编辑',
        delete: '删除',
        enable: '启用',
        disable: '禁用',
      },
    },
    dialogs: {
      apiProfile: {
        createTitle: '新增 API 配置',
        editTitle: '编辑 API 配置',
        baseUrl: 'Base URL',
        model: '模型',
        reasoningEffort: '推理强度',
        contextWindow: '上下文大小',
        apiKey: 'OPENAI_API_KEY',
        hint: '保存后会重新生成 `auth.json` 和 `config.toml`。',
      },
      settings: {
        title: '设置',
        codexHomePath: '目标 Codex 配置目录',
        hint: '保存后会立即重扫该目录，并重新识别当前激活配置。',
      },
      confirm: {
        cancel: '取消',
      },
    },
    confirm: {
      switchTitle: '切换配置',
      switchText: '切换前会先保护当前配置，并把目标配置写入目标 Codex 目录。',
      switchConfirm: '确认切换',
      deleteTitle: '删除配置',
      deleteText: '删除只会影响 CodexSwitch 的托管仓库，不会主动清空目标 Codex 目录。',
      deleteConfirm: '确认删除',
    },
    common: {
      save: '保存',
      copied: '已复制到剪贴板',
      unknownError: '未知错误',
      account: '账号',
      officialAccount: '官方账号',
      english: 'English',
      chinese: '中文',
    },
    notifications: {
      listRefreshed: '列表已刷新',
      officialProfileImported: '官方账号文件已导入',
      apiProfileCreated: 'API 配置已创建',
      apiProfileUpdated: 'API 配置已更新',
      settingsSaved: '设置已保存并完成重扫',
      switched: '配置切换成功',
      deleted: '配置已删除',
      allRateLimitsRefreshed: '全部官方账号额度已刷新',
      rateLimitRefreshed: '{name} 额度已刷新',
      latencyTested: '{name} 延迟已测试',
      allLatencyTested: '全部账号延迟已测试',
      apiAvailabilityRefreshed: 'API 可用性已自动刷新',
      profileEnabled: '{name} 已启用',
      profileDisabled: '{name} 已禁用',
    },
    runtime: {
      wailsNotReady: 'Wails runtime 未就绪，请通过 Wails 启动应用',
      filePickerCancelled: '已取消文件选择',
    },
  },
  'en-US': {
    toolbar: {
      refresh: 'Refresh',
      refreshAllRateLimits: 'Refresh Limits',
      testAll: 'Test All',
      importAccountFile: 'Import Account File',
      addApiProfile: 'Add API Profile',
      settings: 'Settings',
      languageLabel: 'Language',
      localeName: 'English',
    },
    brand: {
      title: 'Manage Codex accounts and API configs in one place',
      subtitle:
        'Detect the current config on launch, then manage, switch, edit APIs, refresh limits, and roll back quickly.',
    },
    overview: {
      currentWorkspace: 'Current Workspace',
      managedProfiles: 'Managed Profiles',
      managedProfilesNote: 'Official {official} · Latency-ready {latency}',
      currentStatus: 'Current Status',
      pendingCurrentProfile: 'Waiting to detect the current config',
    },
    workspace: {
      targetPath: 'Target directory: {path}',
      directoryError: 'Directory Error',
      managed: 'Managed',
      unmanaged: 'Unmanaged',
      notDetected: 'Not Detected',
      errorTitle: 'Current directory status has an issue',
      managedTitle: 'Detected and managing {name}',
      managedText: 'This directory is already under management, so you can switch, refresh limits, and run latency tests directly.',
      availableTitle: 'Detected {name}',
      availableText: 'This config is not managed yet. We will protect the current state and add it to the managed store before switching.',
      emptyTitle: 'No config detected in the current directory',
      emptyText: 'Add an API profile or import an account file to start managing your first switchable config.',
      currentConfig: 'Current config',
    },
    profiles: {
      title: 'Managed Profiles',
      emptyTitle: 'No managed profiles yet',
      emptySubtitle: 'Add an API profile first, or let the app detect the current config automatically.',
      tableLabel: 'Managed profiles',
      headers: {
        displayName: 'Display Name',
        type: 'Type',
        planOrUrl: 'Plan / URL',
        usage5h: '5h',
        usageWeekly: 'Weekly',
        model: 'Model',
        status: 'Status',
        latency: 'Latency Test',
        updatedAt: 'Last Synced',
        actions: 'Actions',
      },
      type: {
        official: 'Official',
        api: 'API',
        unknown: 'Unknown',
      },
      status: {
        invalid: 'Invalid',
        disabled: 'Disabled',
        active: 'Active',
        ready: 'Ready',
      },
      usage: {
        unavailable: 'Unavailable',
      },
      latency: {
        testing: 'Testing',
        failed: 'Test Failed',
        idle: 'Not Tested',
        responded: 'Responded',
        pleaseWait: 'Please wait',
        connectionError: 'Connection issue',
        clickToTest: 'Click to test',
        available: 'Available',
        unavailableWithStatus: 'Unavailable · {statusCode}',
        unavailable: 'Unavailable',
        runningTooltip: 'Latency testing is in progress and will refresh the current account result automatically when it finishes.',
        checkedAt: 'Last test: {time}',
        officialIdleTooltip: 'Test response latency and availability for the current official account through GET /v1/models.',
        apiIdleTooltip: 'Test response latency and availability for the current API key by sending "hi" through POST /responses.',
        failedDefault: 'Latency test failed',
        accountAvailable: 'Account available',
        accountUnavailable: 'Account unavailable',
        tooltipResult: 'Result: {result}',
        tooltipLatency: 'Latency: {latency}',
        tooltipType: 'type: {type}',
        tooltipMessage: 'message: {message}',
        tooltipCode: 'code: {code}',
        tooltipResultAvailable: 'available',
        tooltipResultUnavailable: 'unavailable',
        tooltipResultFailed: 'failed',
      },
      actions: {
        switch: 'Switch',
        test: 'Test',
        refresh: 'Refresh',
        edit: 'Edit',
        delete: 'Delete',
        enable: 'Enable',
        disable: 'Disable',
      },
    },
    dialogs: {
      apiProfile: {
        createTitle: 'Add API Profile',
        editTitle: 'Edit API Profile',
        baseUrl: 'Base URL',
        model: 'Model',
        reasoningEffort: 'Reasoning Effort',
        contextWindow: 'Context Window',
        apiKey: 'OPENAI_API_KEY',
        hint: 'Saving will regenerate `auth.json` and `config.toml`.',
      },
      settings: {
        title: 'Settings',
        codexHomePath: 'Target Codex config directory',
        hint: 'Saving will rescan this directory immediately and detect the active profile again.',
      },
      confirm: {
        cancel: 'Cancel',
      },
    },
    confirm: {
      switchTitle: 'Switch Profile',
      switchText: 'Before switching, the current config will be protected and the target profile will be written to the target Codex directory.',
      switchConfirm: 'Confirm Switch',
      deleteTitle: 'Delete Profile',
      deleteText: 'Deleting only affects the CodexSwitch managed store. It will not proactively clear the target Codex directory.',
      deleteConfirm: 'Confirm Delete',
    },
    common: {
      save: 'Save',
      copied: 'Copied to clipboard',
      unknownError: 'Unknown error',
      account: 'Account',
      officialAccount: 'Official Account',
      english: 'English',
      chinese: 'Chinese',
    },
    notifications: {
      listRefreshed: 'List refreshed',
      officialProfileImported: 'Official account file imported',
      apiProfileCreated: 'API profile created',
      apiProfileUpdated: 'API profile updated',
      settingsSaved: 'Settings saved and rescan completed',
      switched: 'Profile switched successfully',
      deleted: 'Profile deleted',
      allRateLimitsRefreshed: 'All official account limits refreshed',
      rateLimitRefreshed: '{name} limits refreshed',
      latencyTested: '{name} latency tested',
      allLatencyTested: 'All account latency tests completed',
      apiAvailabilityRefreshed: 'API availability refreshed',
      profileEnabled: '{name} enabled',
      profileDisabled: '{name} disabled',
    },
    runtime: {
      wailsNotReady: 'Wails runtime is not ready. Please launch the app through Wails.',
      filePickerCancelled: 'File selection canceled',
    },
  },
};

function normalizeLocale(value?: string | null): AppLocale {
  if (!value) {
    return 'zh-CN';
  }

  return value.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en-US';
}

function resolveInitialLocale(): AppLocale {
  if (typeof window === 'undefined') {
    return 'zh-CN';
  }

  const stored = window.localStorage.getItem(STORAGE_KEY);
  if (stored) {
    return normalizeLocale(stored);
  }

  return normalizeLocale(window.navigator.language);
}

const locale = ref<AppLocale>(resolveInitialLocale());

function syncDocumentLanguage(value: AppLocale) {
  if (typeof document !== 'undefined') {
    document.documentElement.lang = value;
  }
}

function getMessage(localeValue: AppLocale, key: string): string {
  const result = key.split('.').reduce<MessageTree | string | undefined>((current, segment) => {
    if (!current || typeof current === 'string') {
      return undefined;
    }
    return current[segment];
  }, messages[localeValue]);

  if (typeof result === 'string') {
    return result;
  }

  return key;
}

function interpolate(template: string, params?: TranslationParams) {
  if (!params) {
    return template;
  }

  return template.replace(/\{(\w+)\}/g, (_, key: string) => String(params[key] ?? `{${key}}`));
}

export function setLocale(value: AppLocale) {
  locale.value = value;
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(STORAGE_KEY, value);
  }
  syncDocumentLanguage(value);
}

export function translate(key: string, params?: TranslationParams) {
  return interpolate(getMessage(locale.value, key), params);
}

export const runtimeMessageMarkers = {
  filePickerCancelled: Array.from(
    new Set((Object.keys(messages) as AppLocale[]).map((localeValue) => getMessage(localeValue, 'runtime.filePickerCancelled'))),
  ),
};

export function useI18n() {
  const localeName = computed(() =>
    locale.value === 'zh-CN' ? translate('common.chinese') : translate('common.english'),
  );

  return {
    locale,
    localeName,
    setLocale,
    t: translate,
  };
}

syncDocumentLanguage(locale.value);
