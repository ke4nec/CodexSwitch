<template>
  <v-dialog :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)">
    <v-card class="dialog-card">
      <v-card-title class="dialog-title">
        {{ mode === 'create' ? '新增 API 配置' : '编辑 API 配置' }}
      </v-card-title>
      <v-card-text class="dialog-body">
        <v-text-field
          v-model="localForm.baseURL"
          label="Base URL"
          placeholder="https://api.openai.com/v1"
        />
        <v-text-field v-model="localForm.model" label="模型" placeholder="gpt-5.4" />
        <v-text-field
          v-model="localForm.modelReasoningEffort"
          label="推理强度"
          placeholder="xhigh"
        />
        <v-textarea
          v-model="localForm.apiKey"
          label="OPENAI_API_KEY"
          rows="3"
          auto-grow
          placeholder="sk-..."
        />
        <div class="dialog-hint">
          保存后会重新生成 `auth.json` 和 `config.toml`。
        </div>
      </v-card-text>
      <v-card-actions class="dialog-actions">
        <v-spacer />
        <v-btn variant="text" :disabled="loading" @click="emit('update:modelValue', false)">取消</v-btn>
        <v-btn color="primary" :loading="loading" @click="submit">保存</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue';

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

const localForm = reactive<APIProfileInput>({
  baseURL: '',
  model: '',
  modelReasoningEffort: '',
  apiKey: '',
});

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
