<template>
  <v-dialog :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)">
    <v-card class="dialog-card">
      <v-card-title class="dialog-title">
        {{ dialogTitle }}
      </v-card-title>
      <v-card-text class="dialog-body">
        <v-text-field
          v-model="localForm.baseURL"
          :label="t('dialogs.apiProfile.baseUrl')"
          placeholder="https://api.openai.com/v1"
        />
        <v-text-field v-model="localForm.model" :label="t('dialogs.apiProfile.model')" placeholder="gpt-5.4" />
        <v-text-field
          v-model="localForm.modelReasoningEffort"
          :label="t('dialogs.apiProfile.reasoningEffort')"
          placeholder="xhigh"
        />
        <v-text-field
          v-model="localForm.modelContextWindow"
          :label="t('dialogs.apiProfile.contextWindow')"
          type="number"
          min="1"
          step="1"
          placeholder="1000000"
        />
        <v-textarea
          v-model="localForm.apiKey"
          :label="t('dialogs.apiProfile.apiKey')"
          rows="3"
          auto-grow
          placeholder="sk-..."
        />
        <div class="dialog-hint">
          {{ t('dialogs.apiProfile.hint') }}
        </div>
      </v-card-text>
      <v-card-actions class="dialog-actions">
        <v-spacer />
        <v-btn variant="text" :disabled="loading" @click="emit('update:modelValue', false)">
          {{ t('dialogs.confirm.cancel') }}
        </v-btn>
        <v-btn color="primary" :loading="loading" @click="submit">{{ t('common.save') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue';

import { useI18n } from '../i18n';
import type { APIProfileInput } from '../types';

const props = defineProps<{
  modelValue: boolean;
  mode: 'create' | 'edit';
  loading: boolean;
  form: APIProfileInput;
}>();

const emit = defineEmits<{
  'update:modelValue': [boolean];
  save: [APIProfileInput];
}>();

const { t } = useI18n();

const localForm = reactive<APIProfileInput>({
  baseURL: '',
  model: '',
  modelReasoningEffort: '',
  modelContextWindow: '',
  apiKey: '',
});

const dialogTitle = computed(() =>
  props.mode === 'create' ? t('dialogs.apiProfile.createTitle') : t('dialogs.apiProfile.editTitle'),
);

watch(
  () => props.form,
  (value) => {
    Object.assign(localForm, value);
  },
  { deep: true, immediate: true },
);

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      Object.assign(localForm, props.form);
    }
  },
);

function submit() {
  emit('save', { ...localForm });
}
</script>
