<script setup lang="ts">
import { onMounted } from 'vue'
import AppShell from '@/layouts/AppShell.vue'
import { useTheme } from '@/composables/useTheme'
import { useAppStore } from '@/stores/app'
import { useMailStore } from '@/stores/mail'
import { useSmtpStore } from '@/stores/smtp'
import { useToastStore } from '@/stores/toast'
import { errorMessage } from '@/utils/errors'

useTheme().apply()

const app = useAppStore()
const smtp = useSmtpStore()
const mail = useMailStore()

onMounted(async () => {
  try {
    await Promise.all([app.load(), smtp.init(), mail.init()])
  } catch (err) {
    useToastStore().error('MKMailLab failed to initialise', errorMessage(err))
  }
})
</script>

<template>
  <AppShell />
</template>
