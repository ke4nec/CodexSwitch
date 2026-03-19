<template>
  <v-dialog :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)">
    <v-card class="dialog-card">
      <v-card-title class="dialog-title">设置</v-card-title>
      <v-card-text class="dialog-body">
        <v-text-field
          v-model="path"
          label="目标 Codex 配置目录"
          placeholder="C:\\Users\\You\\.codex"
        />
        <div class="dialog-hint">
          保存后会立即重扫该目录，并重新识别当前激活配置。
        </div>
      </v-card-text>
      <v-card-actions class="dialog-actions">
        <v-spacer />
        <v-btn variant="text" :disabled="loading" @click="emit('update:modelValue', false)">取消</v-btn>
        <v-btn color="primary" :loading="loading" @click="emit('save', path)">保存</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';

const props = defineProps<{
  modelValue: boolean;
  loading: boolean;
  codexHomePath: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [boolean];
  save: [string];
}>();

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
