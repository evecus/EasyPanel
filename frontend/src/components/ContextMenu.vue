<template>
  <div id="ctx-menu" v-if="visible" class="ctx"
    :style="{ left: x + 'px', top: y + 'px' }">
    <div class="ctx-item" @mousedown.stop="onEdit">✏️ &nbsp;{{ t('editLabel') }}</div>
    <div class="ctx-item danger" @mousedown.stop="onDelete">🗑 &nbsp;{{ t('deleteBtn') }}</div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from '../composables/useI18n.js'

const { t } = useI18n()
const emit = defineEmits(['edit', 'delete'])

const visible = ref(false)
const x = ref(0), y = ref(0)
let currentId = null

function show(clientX, clientY, id) {
  currentId = id
  const w = 150, h = 90
  x.value = clientX + w > window.innerWidth ? clientX - w : clientX
  y.value = clientY + h > window.innerHeight ? clientY - h : clientY
  visible.value = true
}
function hide() { visible.value = false; currentId = null }

function onEdit() { const id = currentId; hide(); emit('edit', id) }
function onDelete() { const id = currentId; hide(); emit('delete', id) }

defineExpose({ show, hide })
</script>

<style scoped>
.ctx {
  position: fixed; z-index: 9999;
  background: rgba(255,255,255,.97); backdrop-filter: blur(20px);
  border-radius: 14px;
  box-shadow: 0 12px 40px rgba(0,0,0,.18), 0 1px 0 rgba(255,255,255,.8) inset;
  padding: 5px; min-width: 140px;
  border: 1px solid rgba(255,255,255,.6);
  animation: pop-in .14s ease;
}
.ctx-item { display: flex; align-items: center; gap: 9px; padding: 10px 14px; border-radius: 9px; cursor: pointer; font-size: 14px; font-weight: 500; color: #1e1b2e; transition: background var(--tr); user-select: none; }
.ctx-item:hover { background: rgba(168,85,247,.08); }
.ctx-item.danger { color: #ef4444; }
.ctx-item.danger:hover { background: #fef2f2; }
</style>
