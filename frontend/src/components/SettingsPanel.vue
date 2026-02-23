<template>
  <template v-if="visible">
    <div class="s-backdrop" @click="close"></div>
    <div class="s-panel">
      <div class="s-box" :style="{ fontFamily: resolveFont(fonts.ui) }">
        <button class="s-close-top" @click="close" title="关闭"></button>

        <!-- Nav sidebar -->
        <div class="s-nav">
          <div class="s-nav-header">
            <div class="s-nav-ico">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5">
                <rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/>
                <rect x="14" y="14" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/>
              </svg>
            </div>
            <span class="s-nav-t">{{ t('sysSettings') }}</span>
          </div>
          <div class="s-nav-list">
            <div v-for="tab in TABS" :key="tab.id" class="s-ni" :class="{ active: activeTab === tab.id }" @click="activeTab = tab.id">
              <span class="s-ico">{{ tab.icon }}</span>
              <span>{{ t(tab.labelKey) }}</span>
            </div>
          </div>
        </div>

        <!-- Content -->
        <div class="s-content">

          <!-- ── 我的信息 ── -->
          <div v-if="activeTab === 'account'">
            <div class="s-title">{{ t('titleAccount') }}</div>
            <div class="s-card">
              <div class="s-row"><label class="s-lbl">{{ t('lblUsername') }}</label><input class="s-inp" :value="user?.username || ''" disabled /></div>
              <div class="s-row"><label class="s-lbl">{{ t('lblNickname') }}</label><input class="s-inp" v-model="form.nick" :placeholder="t('nickPlaceholder')" /></div>
              <button class="btn btn-p" @click="saveNick">{{ t('saveNickBtn') }}</button>
            </div>
            <div class="s-card">
              <div class="s-title" style="font-size:14px;margin-bottom:14px;padding-bottom:10px;border-bottom-width:1px">{{ t('changePwdTitle') }}</div>
              <div class="s-row"><label class="s-lbl">{{ t('lblOldPwd') }}</label><input type="password" class="s-inp" v-model="form.oldPwd" /></div>
              <div class="s-row"><label class="s-lbl">{{ t('lblNewPwd') }}</label><input type="password" class="s-inp" v-model="form.newPwd" /></div>
              <button class="btn btn-p" @click="changePwd">{{ t('changePwdBtn') }}</button>
            </div>
            <button class="btn btn-d" style="width:100%;margin-top:6px" @click="emit('logout')">{{ t('logoutBtn') }}</button>
          </div>

          <!-- ── 显示 ── -->
          <div v-if="activeTab === 'display'">
            <div class="s-title">{{ t('titleDisplay') }}</div>
            <div class="s-card">
              <div class="s-card-title">主机名 &amp; LOGO</div>
              <div class="s-row"><label class="s-lbl">{{ t('lblHostnameDisplay') }}</label><input class="s-inp" v-model="form.hostname" placeholder="EasyPanel" /></div>
              <div class="s-row">
                <label class="s-lbl">LOGO</label>
                <div class="img-row" style="margin-bottom:8px">
                  <input class="s-inp" v-model="form.logo" placeholder="https://..." @input="form.logoPreview = form.logo" />
                  <label class="upbtn" for="logo-file-inp">📁</label>
                  <input type="file" id="logo-file-inp" accept="image/*" style="display:none" @change="uploadLogo" />
                </div>
                <img v-if="form.logoPreview" :src="form.logoPreview" style="width:48px;height:48px;border-radius:10px;object-fit:cover" />
              </div>
              <button class="btn btn-p" @click="savePanelCfg">{{ t('saveBtn') }}</button>
            </div>
            <div class="s-card" style="margin-top:12px">
              <div class="s-card-title">尺寸大小</div>
              <div class="s-row" v-for="sl in sizeSliders" :key="sl.key">
                <label class="s-lbl">{{ t(sl.label) }}</label>
                <div class="s-slider-row">
                  <input type="range" class="s-slider" v-model="form.display[sl.key]" :min="sl.min" :max="sl.max" />
                  <span class="s-slider-val">{{ form.display[sl.key] }}{{ sl.unit }}</span>
                </div>
              </div>
            </div>
            <div class="s-card" style="margin-top:12px">
              <div class="s-card-title">图标样式</div>
              <div class="s-row" v-for="sl in iconSliders" :key="sl.key">
                <label class="s-lbl">{{ t(sl.label) }}</label>
                <div class="s-slider-row">
                  <input type="range" class="s-slider" v-model="form.display[sl.key]" :min="sl.min" :max="sl.max" />
                  <span class="s-slider-val">{{ form.display[sl.key] }}{{ sl.unit }}</span>
                </div>
              </div>
            </div>
            <button class="btn btn-p" @click="saveDisplay" style="margin-top:12px">{{ t('saveDisplayBtn') }}</button>
          </div>

          <!-- ── 字体 ── -->
          <div v-if="activeTab === 'fonts'">
            <div class="s-title">{{ t('titleFonts') }}</div>
            <div class="s-card">
              <div class="s-card-title">选择字体</div>
              <div class="s-row" v-for="f in fontFields" :key="f.key">
                <label class="s-lbl">{{ t(f.label) }}</label>
                <select class="s-sel" v-model="form.fonts[f.key]">
                  <option v-for="opt in FONT_OPTIONS" :key="opt.v" :value="opt.v">{{ opt.l }}</option>
                </select>
              </div>
              <button class="btn btn-p" @click="saveFonts" style="margin-top:4px">{{ t('saveFonts') }}</button>
            </div>
          </div>

          <!-- ── 主题色 ── -->
          <div v-if="activeTab === 'theme'">
            <div class="s-title">{{ t('titleTheme') }}</div>
            <div class="s-card">
              <div class="theme-grid">
                <div v-for="th in THEMES" :key="th.id" class="theme-item" :class="{ sel: curThemeId === th.id }" @click="applyThemeAndSave(th.id)">
                  <div class="theme-dot" :style="{ background: th.dot }"></div>
                  <div class="theme-name">{{ th.name }}</div>
                </div>
              </div>
            </div>
          </div>

          <!-- ── 时钟 ── -->
          <div v-if="activeTab === 'clock'">
            <div class="s-title">{{ t('titleClock') }}</div>
            <div class="s-card">
              <div class="t-row" v-for="ck in clockToggles" :key="ck.key">
                <div><div class="t-lbl">{{ t(ck.label) }}</div><div v-if="ck.sub" class="t-sub">{{ t(ck.sub) }}</div></div>
                <label class="sw"><input type="checkbox" v-model="form.clock[ck.key]" @change="saveClk" /><span class="sl"></span></label>
              </div>
            </div>
          </div>

          <!-- ── 壁纸 ── -->
          <div v-if="activeTab === 'wallpaper'">
            <div class="s-title">{{ t('titleWallpaper') }}</div>
            <div class="wp-grid">
              <div v-for="url in WPS" :key="url" class="wp-thumb" :class="{ sel: form.wallpaper === url }" @click="selectWp(url)">
                <img :src="url" loading="lazy" />
              </div>
            </div>
            <div class="s-row">
              <label class="s-lbl">{{ t('lblCustomUrl') }}</label>
              <div class="img-row">
                <input class="s-inp" v-model="form.wallpaper" placeholder="https://..." />
                <label class="upbtn" for="wp-file-inp">📁</label>
                <input type="file" id="wp-file-inp" accept="image/*" style="display:none" @change="uploadWp" />
              </div>
            </div>
            <button class="btn btn-p" @click="saveWp">{{ t('applyWpBtn') }}</button>
          </div>

          <!-- ── 账号管理 ── -->
          <div v-if="activeTab === 'users'">
            <div class="s-title">{{ t('titleUsers') }}</div>
            <div class="info-box">{{ t('infoAccounts') }}</div>
            <div class="s-card" style="overflow-x:auto;padding:0">
              <table class="u-table">
                <thead><tr><th>{{ t('thColAccount') }}</th><th>{{ t('thColNickname') }}</th><th>{{ t('thColRole') }}</th><th>{{ t('thColAction') }}</th></tr></thead>
                <tbody>
                  <tr v-for="u in users" :key="u.username">
                    <td>{{ u.username }}<span v-if="u.username === user?.username" class="badge b-cur">{{ t('badgeCurrent') }}</span></td>
                    <td>{{ u.nickname || '-' }}</td>
                    <td><span class="badge" :class="u.is_admin ? 'b-admin' : 'b-user'">{{ u.is_admin ? t('roleAdmin') : t('roleUser') }}</span></td>
                    <td><button v-if="u.username !== user?.username" class="btn btn-d" style="padding:4px 10px;font-size:12px" @click="removeUser(u.username)">{{ t('deleteUserBtn') }}</button><span v-else>-</span></td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="s-card" style="margin-top:12px">
              <div class="s-title" style="font-size:14px;margin-bottom:12px;padding-bottom:10px;border-bottom-width:1px">{{ t('titleAddUser') }}</div>
              <div class="s-row"><label class="s-lbl">{{ t('lblNuName') }}</label><input class="s-inp" v-model="form.nuName" /></div>
              <div class="s-row"><label class="s-lbl">{{ t('lblNuPwd') }}</label><input type="password" class="s-inp" v-model="form.nuPwd" /></div>
              <div class="s-row"><label class="s-lbl">{{ t('lblNuNick') }}</label><input class="s-inp" v-model="form.nuNick" /></div>
              <button class="btn btn-p" @click="addUser">{{ t('addUserBtn') }}</button>
            </div>
          </div>

          <!-- ── 访问控制 ── -->
          <div v-if="activeTab === 'access'">
            <div class="s-title">{{ t('titleAccess') }}</div>
            <div class="s-card">
              <div class="t-row">
                <div><div class="t-lbl">{{ t('lblPubMode') }}</div><div class="t-sub">{{ t('subPubMode') }}</div></div>
                <label class="sw"><input type="checkbox" v-model="form.pubMode" @change="setPubMode" /><span class="sl"></span></label>
              </div>
            </div>
            <div class="info-box" style="margin-top:12px">
              <b>{{ t('lblPrivateMode') }}</b>{{ t('descPrivateMode') }}<br>
              <b>{{ t('lblPublicMode') }}</b>{{ t('descPublicMode') }}
            </div>
          </div>

          <!-- ── 数据管理 ── -->
          <div v-if="activeTab === 'backup'">
            <div class="s-title">{{ t('titleBackup') }}</div>
            <div class="info-box" v-html="t('infoBackup').replace('\n','<br>')"></div>
            <div class="s-card">
              <div class="backup-btns">
                <button class="backup-btn" @click="exportData"><span class="b-ico">📤</span><span>{{ t('exportBtn') }}</span><span class="b-sub">{{ t('exportDesc') }}</span></button>
                <label class="backup-btn" for="import-file-inp"><span class="b-ico">📥</span><span>{{ t('importBtn') }}</span><span class="b-sub">{{ t('importDesc') }}</span></label>
                <input type="file" id="import-file-inp" accept=".json" style="display:none" @change="importData" />
              </div>
            </div>
            <div class="s-card" style="margin-top:12px">
              <div class="t-lbl" style="margin-bottom:10px">{{ t('lblDanger') }}</div>
              <button class="btn btn-d" @click="emit('clearData')" style="width:100%">{{ t('clearBtn') }}</button>
            </div>
          </div>

          <!-- ── 语言 ── -->
          <div v-if="activeTab === 'language'">
            <div class="s-title">🌐 语言 / Language</div>
            <div class="s-card">
              <div class="s-row">
                <label class="s-lbl">语言 / Language</label>
                <div class="lang-btns">
                  <button class="lang-btn" :class="{ active: lang === 'zh' }" @click="setLang('zh')">🇨🇳 中文</button>
                  <button class="lang-btn" :class="{ active: lang === 'en' }" @click="setLang('en')">🇬🇧 English</button>
                </div>
              </div>
            </div>
          </div>

        </div>
      </div>
    </div>
  </template>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { useI18n, lang } from '../composables/useI18n.js'
import { apiCall } from '../composables/useApi.js'
import { THEMES, WPS, FONT_OPTIONS, curThemeId, applyThemeCss } from '../composables/useTheme.js'
import { resolveFont } from '../composables/useTheme.js'

const { t } = useI18n()
const emit = defineEmits(['toast', 'logout', 'panelUpdated', 'clearData'])

const props = defineProps({
  user: Object,
  panelInfo: Object,
  pubModeValue: Boolean,
  dispSet: Object,
  fontSet: Object,
  clkCfg: Object,
  apps: Array,
})

const visible = ref(false)
const activeTab = ref('account')
const users = ref([])

const form = reactive({
  nick: '', oldPwd: '', newPwd: '',
  hostname: '', logo: '', logoPreview: '', wallpaper: '',
  pubMode: false,
  nuName: '', nuPwd: '', nuNick: '',
  display: { hostnameSize: 56, clockSize: 16, iconSize: 78, appNameSize: 12, iconRadius: 26, iconGap: 22, sidePadding: 52 },
  fonts: { hostname: 'system', clock: 'system', appname: 'system', ui: 'system' },
  clock: { show_time: true, show_date: true, show_weekday: true, show_lunar: false, show_seconds: false },
})

const TABS = [
  { id: 'account',   icon: '👤', labelKey: 'niAccount' },
  { id: 'display',   icon: '📐', labelKey: 'niDisplay' },
  { id: 'fonts',     icon: '🔤', labelKey: 'niFonts' },
  { id: 'theme',     icon: '🌈', labelKey: 'niTheme' },
  { id: 'clock',     icon: '🕐', labelKey: 'niClock' },
  { id: 'wallpaper', icon: '🖼️', labelKey: 'niWallpaper' },
  { id: 'users',     icon: '👥', labelKey: 'niUsers' },
  { id: 'access',    icon: '🔓', labelKey: 'niAccess' },
  { id: 'backup',    icon: '💾', labelKey: 'niBackup' },
  { id: 'language',  icon: '🌐', labelKey: 'niLanguage' },
]

const sizeSliders = [
  { key: 'hostnameSize', label: 'lblHostnameSize', min: 24, max: 96,  unit: 'px' },
  { key: 'clockSize',    label: 'lblClockSize',    min: 10, max: 36,  unit: 'px' },
  { key: 'iconSize',     label: 'lblIconSize',     min: 40, max: 130, unit: 'px' },
  { key: 'appNameSize',  label: 'lblAppnameSize',  min: 8,  max: 22,  unit: 'px' },
]
const iconSliders = [
  { key: 'iconRadius',  label: 'iconRadius',  min: 0,  max: 50,  unit: '%' },
  { key: 'iconGap',     label: 'iconGap',     min: 8,  max: 60,  unit: 'px' },
  { key: 'sidePadding', label: 'sidePadding', min: 10, max: 120, unit: 'px' },
]
const fontFields = [
  { key: 'hostname', label: 'fontHostname' },
  { key: 'clock',    label: 'fontClock' },
  { key: 'appname',  label: 'fontAppname' },
  { key: 'ui',       label: 'fontUi' },
]
const clockToggles = [
  { key: 'show_time',    label: 'ckTime' },
  { key: 'show_seconds', label: 'ckSec' },
  { key: 'show_date',    label: 'ckDate', sub: 'ckDateSub' },
  { key: 'show_weekday', label: 'ckWeek' },
  { key: 'show_lunar',   label: 'ckLunar', sub: 'ckLunarSub' },
]

function buildPayload() {
  return {
    hostname: form.hostname, logo: form.logo, wallpaper: form.wallpaper,
    clock: { ...form.clock }, theme: curThemeId.value, language: lang.value,
    hostname_size: +form.display.hostnameSize, clock_size: +form.display.clockSize,
    icon_size: +form.display.iconSize, app_name_size: +form.display.appNameSize,
    icon_radius: +form.display.iconRadius, icon_gap: +form.display.iconGap,
    side_padding: +form.display.sidePadding,
    font_hostname: form.fonts.hostname, font_clock: form.fonts.clock,
    font_appname: form.fonts.appname, font_ui: form.fonts.ui,
  }
}

async function open() {
  form.nick = props.user?.nickname || ''
  form.oldPwd = ''; form.newPwd = ''
  form.hostname = props.panelInfo?.hostname || ''
  form.logo = props.panelInfo?.logo || ''
  form.logoPreview = props.panelInfo?.logo || ''
  form.wallpaper = props.panelInfo?.wallpaper || ''
  form.pubMode = props.pubModeValue || false
  Object.assign(form.display, props.dispSet)
  Object.assign(form.fonts, props.fontSet)
  Object.assign(form.clock, props.clkCfg)
  form.nuName = ''; form.nuPwd = ''; form.nuNick = ''
  visible.value = true   // 先显示面板，不等 loadUsers
  loadUsers()            // 异步加载用户列表，不阻塞打开
}
function close() { visible.value = false }

async function saveNick() {
  try { await apiCall('/api/me/nickname', { method: 'PUT', body: JSON.stringify({ nickname: form.nick.trim() }) }); emit('toast', t('tNickSaved')); emit('panelUpdated') }
  catch { emit('toast', t('tFailed')) }
}
async function changePwd() {
  if (!form.oldPwd || !form.newPwd) { emit('toast', t('fillPwd')); return }
  try { await apiCall('/api/me/password', { method: 'PUT', body: JSON.stringify({ old_password: form.oldPwd, new_password: form.newPwd }) }); form.oldPwd = ''; form.newPwd = ''; emit('toast', t('tPwdChanged')) }
  catch { emit('toast', t('tPwdFailed')) }
}
async function savePanelCfg() {
  const sv = buildPayload(); sv.hostname = form.hostname.trim(); sv.logo = form.logo.trim()
  try { await apiCall('/api/settings', { method: 'PUT', body: JSON.stringify(sv) }); emit('panelUpdated'); emit('toast', t('tSaved')) }
  catch { emit('toast', t('tFailed')) }
}
async function uploadLogo(e) {
  const f = e.target.files[0]; if (!f) return
  const fd = new FormData(); fd.append('logo', f)
  try { const r = await fetch('/api/upload/logo', { method: 'POST', body: fd, credentials: 'include' }); const d = await r.json(); form.logo = d.url; form.logoPreview = d.url; emit('toast', t('tUploaded')) }
  catch { emit('toast', t('tUploadFailed')) }
  e.target.value = ''
}
async function saveDisplay() {
  const sv = buildPayload()
  try { await apiCall('/api/settings', { method: 'PUT', body: JSON.stringify(sv) }); emit('panelUpdated'); emit('toast', t('tSaved')) }
  catch { emit('toast', t('tFailed')) }
}
async function saveFonts() {
  const sv = buildPayload()
  try { await apiCall('/api/settings', { method: 'PUT', body: JSON.stringify(sv) }); emit('panelUpdated'); emit('toast', t('tSaved')) }
  catch { emit('toast', t('tFailed')) }
}
async function applyThemeAndSave(id) {
  curThemeId.value = id
  applyThemeCss(THEMES.find(x => x.id === id))
  const sv = buildPayload()
  try { await apiCall('/api/settings', { method: 'PUT', body: JSON.stringify(sv) }) }
  catch {}
}
async function saveClk() {
  const sv = buildPayload()
  try { await apiCall('/api/settings', { method: 'PUT', body: JSON.stringify(sv) }); emit('panelUpdated') }
  catch {}
}
function selectWp(url) { form.wallpaper = url }
async function uploadWp(e) {
  const f = e.target.files[0]; if (!f) return
  const fd = new FormData(); fd.append('wallpaper', f)
  try { const r = await fetch('/api/upload/wallpaper', { method: 'POST', body: fd, credentials: 'include' }); const d = await r.json(); form.wallpaper = d.url; emit('toast', t('tUploaded')) }
  catch { emit('toast', t('tUploadFailed')) }
  e.target.value = ''
}
async function saveWp() {
  const sv = buildPayload(); sv.wallpaper = form.wallpaper.trim()
  try { await apiCall('/api/settings', { method: 'PUT', body: JSON.stringify(sv) }); emit('panelUpdated'); emit('toast', t('tSaved')) }
  catch { emit('toast', t('tFailed')) }
}
async function loadUsers() {
  try { users.value = await apiCall('/api/users') } catch {}
}
async function addUser() {
  if (!form.nuName.trim() || !form.nuPwd) { emit('toast', t('fillAccountPwd')); return }
  try { await apiCall('/api/users', { method: 'POST', body: JSON.stringify({ username: form.nuName.trim(), password: form.nuPwd, nickname: form.nuNick.trim() }) }); form.nuName = ''; form.nuPwd = ''; form.nuNick = ''; await loadUsers(); emit('toast', t('tUserAdded')) }
  catch { emit('toast', t('tUserAddFailed')) }
}
async function removeUser(username) {
  if (!confirm(`${t('confirmDeleteUser')} ${username}？`)) return
  try { await apiCall(`/api/users/${username}`, { method: 'DELETE' }); await loadUsers(); emit('toast', t('tDeleted')) }
  catch { emit('toast', t('tDeleteFailed')) }
}
async function setPubMode() {
  const v = form.pubMode
  try { await apiCall('/api/publicmode', { method: 'PUT', body: JSON.stringify({ public_mode: v }) }); emit('toast', v ? t('pubOn') : t('pubOff')); emit('panelUpdated') }
  catch { emit('toast', t('tFailed')); form.pubMode = !v }
}
async function setLang(l) {
  lang.value = l; localStorage.setItem('ep_lang', l)
  const sv = buildPayload(); sv.language = l
  await apiCall('/api/settings', { method: 'PUT', body: JSON.stringify(sv) }).catch(() => {})
  emit('toast', t('tSaved'))
}
async function exportData() {
  try {
    const [appsData, settingsData] = await Promise.all([apiCall('/api/apps'), apiCall('/api/settings')])
    const backup = { version: 1, exported_at: new Date().toISOString(), apps: appsData, settings: settingsData, theme: curThemeId.value }
    const blob = new Blob([JSON.stringify(backup, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a'); a.href = url; a.download = `easypanel_backup_${new Date().toISOString().slice(0, 10)}.json`; a.click()
    URL.revokeObjectURL(url); emit('toast', t('tExported'))
  } catch { emit('toast', t('tExportFailed')) }
}
async function importData(e) {
  const f = e.target.files[0]; if (!f) return
  if (!confirm(t('confirmImport'))) { e.target.value = ''; return }
  try {
    const text = await f.text(); const b = JSON.parse(text)
    if (!b.version || !b.apps) { emit('toast', t('badBackup')); e.target.value = ''; return }
    if (b.settings) await apiCall('/api/settings', { method: 'PUT', body: JSON.stringify(b.settings) })
    for (const app of props.apps || []) await apiCall(`/api/apps/${app.id}`, { method: 'DELETE' }).catch(() => {})
    for (const app of b.apps) { const { id: _, order: __, ...rest } = app; await apiCall('/api/apps', { method: 'POST', body: JSON.stringify(rest) }) }
    if (b.theme) { const th = THEMES.find(x => x.id === b.theme); if (th) { curThemeId.value = th.id; applyThemeCss(th) } }
    emit('panelUpdated'); emit('toast', t('tImported'))
  } catch { emit('toast', t('tImportFailed')) }
  e.target.value = ''
}

defineExpose({ open, close })
</script>

<style scoped>
.s-backdrop { position: fixed; inset: 0; z-index: 750; background: rgba(0,0,0,.28); }
.s-panel { position: fixed; inset: 0; z-index: 760; display: flex; align-items: center; justify-content: center; padding: 20px; }
.s-box { position: relative; background: white; border-radius: 22px; width: min(940px,100%); height: min(690px,90vh); display: flex; overflow: hidden; box-shadow: 0 32px 80px rgba(168,85,247,.2); }
.s-close-top { position: absolute; top: 14px; right: 14px; width: 22px; height: 22px; border-radius: 50%; background: #ff5f57; border: none; cursor: pointer; display: flex; align-items: center; justify-content: center; transition: all var(--tr); box-shadow: 0 2px 6px rgba(255,95,87,.45); z-index: 10; }
.s-close-top:hover::after { content: '✕'; font-size: 11px; font-weight: 700; color: rgba(0,0,0,.5); }
.s-nav { width: 215px; flex-shrink: 0; background: linear-gradient(180deg,#faf5ff 0%,#fdf2f8 100%); border-right: 1px solid #ede8f5; display: flex; flex-direction: column; overflow-y: auto; position: relative; }
.s-nav-header { padding: 20px 18px 14px; display: flex; align-items: center; gap: 10px; }
.s-nav-ico { width: 32px; height: 32px; background: var(--grad); border-radius: 9px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; box-shadow: 0 4px 12px color-mix(in srgb,var(--h1) 35%,transparent); }
.s-nav-t { font-size: 14px; font-weight: 800; background: var(--grad); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }
.s-nav-list { padding: 8px; flex: 1; }
.s-ni { display: flex; align-items: center; gap: 9px; padding: 10px 13px; border-radius: 11px; cursor: pointer; font-size: 13px; font-weight: 600; color: #6b7280; transition: all var(--tr); margin-bottom: 3px; }
.s-ni:hover { background: rgba(168,85,247,.07); color: #1e1b2e; }
.s-ni.active { background: var(--grad); color: white; box-shadow: 0 4px 14px color-mix(in srgb,var(--h1) 30%,transparent); }
.s-ico { font-size: 16px; width: 22px; text-align: center; }
.s-content { flex: 1; overflow-y: auto; padding: 28px 30px; }
.theme-grid { display: grid; grid-template-columns: repeat(4,1fr); gap: 10px; margin-bottom: 4px; }
.theme-item { border-radius: 13px; padding: 12px 8px; cursor: pointer; transition: all var(--tr); background: white; border: 2px solid #ede8f5; text-align: center; }
.theme-item:hover { transform: scale(1.04); border-color: var(--h1); }
.theme-item.sel { border-color: var(--h1); box-shadow: 0 4px 14px color-mix(in srgb,var(--h1) 25%,transparent); }
.theme-dot { width: 36px; height: 36px; border-radius: 50%; margin: 0 auto 8px; box-shadow: 0 3px 10px rgba(0,0,0,.15); }
.theme-name { font-size: 11px; font-weight: 600; color: #4b5563; }
.u-table { width: 100%; border-collapse: collapse; }
.u-table th { font-size: 11px; font-weight: 700; color: #94a3b8; text-transform: uppercase; letter-spacing: .5px; padding: 8px 13px; text-align: left; background: #f5f3ff; border-bottom: 1px solid #ede8f5; }
.u-table th:first-child { border-radius: 10px 0 0 0; } .u-table th:last-child { border-radius: 0 10px 0 0; }
.u-table td { padding: 12px 13px; border-bottom: 1px solid #faf5ff; font-size: 14px; color: #1e1b2e; }
.u-table tr:last-child td { border-bottom: none; }
.wp-grid { display: grid; grid-template-columns: repeat(3,1fr); gap: 9px; margin-bottom: 14px; }
.wp-thumb { aspect-ratio: 16/9; border-radius: 11px; cursor: pointer; overflow: hidden; border: 3px solid transparent; transition: all var(--tr); }
.wp-thumb:hover, .wp-thumb.sel { border-color: var(--h1); transform: scale(1.03); box-shadow: 0 4px 14px color-mix(in srgb,var(--h1) 25%,transparent); }
.wp-thumb img { width: 100%; height: 100%; object-fit: cover; display: block; }
.backup-btns { display: flex; gap: 10px; flex-wrap: wrap; }
.backup-btn { flex: 1; min-width: 140px; padding: 13px; border-radius: 12px; border: 2px dashed #d8b4fe; background: #faf5ff; cursor: pointer; font-size: 14px; font-weight: 600; color: #7c3aed; transition: all var(--tr); display: flex; flex-direction: column; align-items: center; gap: 6px; }
.backup-btn:hover { background: #f3e8ff; border-color: var(--h1); transform: translateY(-2px); }
.b-ico { font-size: 24px; } .b-sub { font-size: 11px; color: #94a3b8; font-weight: 500; }
.lang-btns { display: flex; gap: 8px; }
.lang-btn { flex: 1; padding: 9px 14px; border: 2px solid #ede8f5; border-radius: 11px; background: white; cursor: pointer; font-size: 14px; font-weight: 700; color: #6b7280; transition: all var(--tr); text-align: center; }
.lang-btn:hover { border-color: var(--h1); color: var(--h1); }
.lang-btn.active { border-color: var(--h1); color: var(--h1); background: #faf5ff; }
</style>
