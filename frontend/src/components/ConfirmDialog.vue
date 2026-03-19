<template>
  <v-dialog :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)">
    <v-card class="dialog-card">
      <v-card-title class="dialog-title">{{ title }}</v-card-title>
      <v-card-text class="dialog-body">
        <p class="confirm-text">{{ text }}</p>
      </v-card-text>
      <v-card-actions class="dialog-actions">
        <v-spacer />
        <v-btn variant="text" :disabled="loading" @click="emit('update:modelValue', false)">取消</v-btn>
        <v-btn :color="color" :loading="loading" @click="emit('confirm')">{{ confirmText }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: boolean;
  loading: boolean;
  title: string;
  text: string;
  confirmText: string;
  color: 'primary' | 'error';
}>();

const emit = defineEmits<{
  'update:modelValue': [boolean];
  confirm: [];
}>();
</script>
