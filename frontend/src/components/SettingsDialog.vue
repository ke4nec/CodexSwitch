<template>
  <v-dialog :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)">
    <v-card class="dialog-card">
      <v-card-title class="dialog-title">{{ t('dialogs.settings.title') }}</v-card-title>
      <v-card-text class="dialog-body">
        <v-text-field
          v-model="path"
          :label="t('dialogs.settings.codexHomePath')"
          placeholder="C:\\Users\\You\\.codex"
        />
        <div class="dialog-hint">
          {{ t('dialogs.settings.hint') }}
        </div>
      </v-card-text>
      <v-card-actions class="dialog-actions">
        <v-spacer />
        <v-btn variant="text" :disabled="loading" @click="emit('update:modelValue', false)">
          {{ t('dialogs.confirm.cancel') }}
        </v-btn>
        <v-btn color="primary" :loading="loading" @click="emit('save', path)">{{ t('common.save') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';

import { useI18n } from '../i18n';

const props = defineProps<{
  modelValue: boolean;
  loading: boolean;
  codexHomePath: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [boolean];
  save: [string];
}>();

const { t } = useI18n();

const path = ref('');

watch(
  () => props.codexHomePath,
  (value) => {
    path.value = value;
  },
  { immediate: true },
);

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      path.value = props.codexHomePath;
    }
  },
);
</script>
