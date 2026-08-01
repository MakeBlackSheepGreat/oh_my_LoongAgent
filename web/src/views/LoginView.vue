<script setup lang="ts">
// 登录视图：用户名 + 密码登录，无账户时引导注册（账号密码）。
import { onMounted, ref } from 'vue'
import { useSession } from '../stores/session'
import { useRouter } from '../router'
import { t } from '../i18n'
import * as api from '../api/client'

const session = useSession()
const router = useRouter()

const showRegister = ref(false)
const loginUsername = ref('')
const loginPassword = ref('')
const loginError = ref('')
const loginBusy = ref(false)

const registerName = ref('')
const registerUsername = ref('')
const registerPassword = ref('')
const registerLocale = ref('zh-CN')
const registerBusy = ref(false)
const registerError = ref('')

function mapError(err: unknown): string {
  if (err instanceof api.ApiError) {
    if (err.code === 'UNAUTHORIZED') return t('auth.loginError.badCredentials')
    if (err.code === 'CONFLICT' || err.message.includes('already exists')) return t('auth.registerError.conflict')
    if (err.message.includes('required')) return t('auth.registerError.required')
    if (err.message.includes('username')) return t('auth.registerError.badUsername')
    if (err.message.includes('password')) return t('auth.registerError.shortPassword')
    if (err.message.includes('locale')) return t('auth.registerError.invalidLocale')
    return err.message || t('auth.registerError.generic')
  }
  return err instanceof Error ? err.message : t('auth.registerError.generic')
}

onMounted(async () => {
  if (!session.state.booted) {
    await session.boot()
  }
  // 已登录直接进入工作台。
  if (session.state.me) {
    router.navigate('workbench')
  }
})

async function handleLogin(): Promise<void> {
  if (loginBusy.value) return
  const username = loginUsername.value.trim()
  const password = loginPassword.value
  if (!username || !password) {
    loginError.value = t('auth.loginError.required')
    return
  }
  loginBusy.value = true
  loginError.value = ''
  try {
    await session.login(username, password)
    router.navigate('workbench')
  } catch (e: unknown) {
    loginError.value = mapError(e)
  } finally {
    loginBusy.value = false
  }
}

async function handleRegister(): Promise<void> {
  if (registerBusy.value) return
  const username = registerUsername.value.trim()
  const name = registerName.value.trim()
  const password = registerPassword.value
  if (!name || !username || !password) {
    registerError.value = t('auth.registerError.required')
    return
  }
  registerBusy.value = true
  registerError.value = ''
  try {
    const acc = await api.register(username, name, password, registerLocale.value)
    // 注册成功后写入 session store 并导航
    session.state.me = acc
    router.navigate('workbench')
  } catch (e: unknown) {
    registerError.value = mapError(e)
  } finally {
    registerBusy.value = false
  }
}

function toggleRegister(): void {
  showRegister.value = !showRegister.value
  loginError.value = ''
  registerError.value = ''
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="brand">
        <div class="brand-mark">A</div>
        <div>
          <h1 class="brand-title">{{ t('app.title') }}</h1>
          <p class="brand-subtitle">{{ t('app.subtitle') }}</p>
        </div>
      </div>

      <p class="login-hint">{{ t('auth.loginHint') }}</p>

      <div v-if="!session.state.booted" class="login-loading">
        <span class="spinner"></span>
        <span>{{ t('common.loading') }}</span>
      </div>

      <!-- 注册表单 -->
      <form v-else-if="showRegister" class="register-form" @submit.prevent="handleRegister">
        <p class="form-title">{{ t('auth.registerTitle') }}</p>
        <p class="form-hint">{{ t('auth.registerHint') }}</p>

        <div class="field">
          <label class="field-label">{{ t('auth.username') }}</label>
          <input
            v-model="registerUsername"
            class="field-input"
            :placeholder="t('auth.usernamePlaceholder')"
            maxlength="32"
            autocomplete="username"
            :disabled="registerBusy"
          />
        </div>

        <div class="field">
          <label class="field-label">{{ t('auth.displayName') }}</label>
          <input
            v-model="registerName"
            class="field-input"
            :placeholder="t('auth.displayNamePlaceholder')"
            maxlength="64"
            autocomplete="name"
            :disabled="registerBusy"
          />
        </div>

        <div class="field">
          <label class="field-label">{{ t('auth.password') }}</label>
          <input
            v-model="registerPassword"
            type="password"
            class="field-input"
            :placeholder="t('auth.passwordPlaceholder')"
            maxlength="128"
            autocomplete="new-password"
            :disabled="registerBusy"
          />
        </div>

        <div class="field">
          <label class="field-label">{{ t('auth.locale') }}</label>
          <select v-model="registerLocale" class="field-input" :disabled="registerBusy">
            <option value="zh-CN">中文</option>
            <option value="en">English</option>
          </select>
        </div>

        <p v-if="registerError" class="field-error">{{ registerError }}</p>

        <button class="btn btn-primary register-btn" type="submit" :disabled="registerBusy">
          <span v-if="registerBusy" class="spinner"></span>
          <span>{{ registerBusy ? t('auth.registering') : t('auth.register') }}</span>
        </button>

        <button class="btn btn-ghost back-btn" type="button" :disabled="registerBusy" @click="toggleRegister">
          {{ t('auth.orLogin') }}
        </button>
      </form>

      <!-- 登录表单 -->
      <form v-else class="login-form" @submit.prevent="handleLogin">
        <p class="form-title">{{ t('auth.loginTitle') }}</p>

        <div class="field">
          <label class="field-label">{{ t('auth.username') }}</label>
          <input
            v-model="loginUsername"
            class="field-input"
            :placeholder="t('auth.usernamePlaceholder')"
            maxlength="32"
            autocomplete="username"
            :disabled="loginBusy"
          />
        </div>

        <div class="field">
          <label class="field-label">{{ t('auth.password') }}</label>
          <input
            v-model="loginPassword"
            type="password"
            class="field-input"
            :placeholder="t('auth.passwordPlaceholder')"
            maxlength="128"
            autocomplete="current-password"
            :disabled="loginBusy"
          />
        </div>

        <p v-if="loginError" class="field-error">{{ loginError }}</p>

        <button class="btn btn-primary register-btn" type="submit" :disabled="loginBusy">
          <span v-if="loginBusy" class="spinner"></span>
          <span>{{ loginBusy ? t('auth.loggingIn') : t('auth.login') }}</span>
        </button>

        <div class="account-list-footer">
          <button class="btn btn-ghost create-btn" type="button" :disabled="loginBusy" @click="toggleRegister">
            {{ t('auth.createAccount') }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.login-card {
  width: 100%;
  max-width: 420px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 20px;
  box-shadow: var(--shadow-lg);
  padding: 34px 36px 30px;
  animation: rise 0.32s ease;
}
@keyframes rise {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.brand {
  display: flex;
  align-items: center;
  gap: 14px;
}
.brand-mark {
  width: 46px;
  height: 46px;
  flex: none;
  border-radius: 14px;
  background: linear-gradient(135deg, #14a596, #0a7569);
  color: #fff;
  font-size: 22px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 6px 16px rgba(14, 147, 132, 0.35), inset 0 1px 0 rgba(255, 255, 255, 0.25);
}
.brand-title {
  font-size: 19px;
  color: var(--text);
}
.brand-subtitle {
  font-size: 12.5px;
  color: var(--text-3);
}

.login-hint {
  margin: 18px 0 14px;
  font-size: 12.5px;
  color: var(--text-3);
}
.login-loading {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-2);
  font-size: 13px;
  padding: 18px 0;
}

/* ---- 表单 ---- */
.login-form,
.register-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.form-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--text);
  margin-top: 4px;
}
.form-hint {
  font-size: 12.5px;
  color: var(--text-3);
  margin-top: -8px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.field-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-2);
}
.field-input {
  padding: 9px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  color: var(--text);
  font-size: 13.5px;
  transition: border-color 0.16s ease, box-shadow 0.2s ease;
}
.field-input:focus {
  outline: none;
  border-color: var(--cobalt);
  box-shadow: 0 0 0 3px rgba(54, 127, 232, 0.18);
}
.field-input:disabled {
  opacity: 0.6;
}
select.field-input {
  cursor: pointer;
  appearance: auto;
}
.field-error {
  font-size: 12px;
  color: var(--coral);
  margin: -4px 0;
}
.register-btn {
  margin-top: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}
.back-btn {
  font-size: 12.5px;
  color: var(--text-3);
}
.create-btn {
  font-size: 12.5px;
  color: var(--text-3);
}
.create-btn:hover {
  color: var(--teal);
}
.account-list-footer {
  display: flex;
  justify-content: center;
  margin-top: 4px;
  padding-top: 10px;
  border-top: 1px solid var(--border);
}
</style>
