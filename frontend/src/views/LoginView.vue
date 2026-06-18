<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'
import AppLogo from '@/components/AppLogo.vue'

const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()
const form = ref({ username: '', password: '' })
const totpCode = ref('')
const tempToken = ref('')
const step = ref<'password' | 'totp'>('password')
const loading = ref(false)

async function handleLogin() {
  loading.value = true
  try {
    const result = await auth.login(form.value.username, form.value.password)
    if (result?.requireTotp) {
      tempToken.value = result.tempToken || ''
      step.value = 'totp'
      ElMessage.info(t('login.totpRequired'))
      return
    }
    ElMessage.success(t('login.success'))
    router.push('/dashboard')
  } catch (e: any) {
    const msg = e?.error || e?.message || t('login.failed')
    if (typeof msg === 'string' && (msg.includes('404') || msg.includes('entrance'))) {
      ElMessage.error(t('login.wrongEntrance'))
    } else {
      ElMessage.error(msg)
    }
  } finally {
    loading.value = false
  }
}

async function handleTotpLogin() {
  loading.value = true
  try {
    await auth.loginTotp(tempToken.value, totpCode.value)
    ElMessage.success(t('login.success'))
    router.push('/dashboard')
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('login.totpFailed'))
  } finally {
    loading.value = false
  }
}

function backToPassword() {
  step.value = 'password'
  totpCode.value = ''
}
</script>

<template>
  <div class="login-page">
    <div class="login-hero">
      <div class="login-bg-effects" aria-hidden="true">
        <span class="glow-orb glow-orb-1" />
        <span class="glow-orb glow-orb-2" />
        <span class="glow-orb glow-orb-3" />
        <span class="glow-orb glow-orb-4" />
        <span class="glow-beam glow-beam-1" />
        <span class="glow-beam glow-beam-2" />
      </div>
      <div class="hero-content">
        <div class="hero-logo"><AppLogo :size="56" /></div>
        <h1>{{ t('common.appName') }}</h1>
        <p>{{ t('login.subtitle') }}</p>
      </div>
    </div>
    <div class="login-form-panel">
      <div class="login-card">
        <h2>{{ step === 'totp' ? t('login.totpTitle') : t('common.login') }}</h2>
        <p class="login-desc">{{ step === 'totp' ? t('login.totpDesc') : t('login.welcome') }}</p>
        <el-form v-if="step === 'password'" :model="form" @submit.prevent="handleLogin">
          <el-form-item>
            <label class="field-label">{{ t('common.username') }}</label>
            <el-input v-model="form.username" :placeholder="t('common.username')" size="large" prefix-icon="User" />
          </el-form-item>
          <el-form-item>
            <label class="field-label">{{ t('common.password') }}</label>
            <el-input v-model="form.password" type="password" :placeholder="t('common.password')" size="large" prefix-icon="Lock" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="large" class="login-btn" :loading="loading" native-type="submit">{{ t('common.login') }}</el-button>
          </el-form-item>
        </el-form>
        <el-form v-else @submit.prevent="handleTotpLogin">
          <el-form-item>
            <label class="field-label">{{ t('login.totpCode') }}</label>
            <el-input v-model="totpCode" :placeholder="t('login.totpCodeHint')" size="large" maxlength="6" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="large" class="login-btn" :loading="loading" native-type="submit">{{ t('login.totpVerify') }}</el-button>
          </el-form-item>
          <el-button link @click="backToPassword">{{ t('login.back') }}</el-button>
        </el-form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  display: flex;
  min-height: 100vh;
  background: var(--cf-bg);
  position: relative;
  overflow: hidden;
}
.login-page::before {
  content: '';
  position: absolute;
  inset: -40%;
  background:
    radial-gradient(ellipse 55% 45% at 20% 30%, rgba(246, 130, 31, 0.07) 0%, transparent 55%),
    radial-gradient(ellipse 50% 40% at 80% 70%, rgba(37, 99, 235, 0.06) 0%, transparent 50%),
    radial-gradient(ellipse 45% 35% at 55% 15%, rgba(168, 85, 247, 0.05) 0%, transparent 45%);
  animation: page-ambient 28s ease-in-out infinite;
  pointer-events: none;
  z-index: 0;
}
.login-hero {
  flex: 1;
  background: #050508;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 64px 48px;
  position: relative;
  overflow: hidden;
  z-index: 1;
}
.login-hero::after {
  content: '';
  position: absolute;
  inset: -60%;
  background: conic-gradient(
    from 0deg at 50% 50%,
    rgba(246, 130, 31, 0.06) 0deg,
    rgba(37, 99, 235, 0.05) 110deg,
    rgba(168, 85, 247, 0.06) 220deg,
    rgba(255, 180, 80, 0.04) 300deg,
    rgba(246, 130, 31, 0.06) 360deg
  );
  animation: hero-gradient-spin 45s linear infinite;
  pointer-events: none;
}

.login-bg-effects {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
}

.glow-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(60px);
  opacity: 0.85;
  will-change: transform, opacity;
}

.glow-orb-1 {
  width: min(620px, 55vw);
  height: min(620px, 55vw);
  top: -8%;
  left: -5%;
  background: radial-gradient(circle, rgba(246, 130, 31, 0.6) 0%, rgba(246, 130, 31, 0.1) 42%, transparent 68%);
  animation: orb-drift-1 16s ease-in-out infinite;
}

.glow-orb-2 {
  width: min(500px, 48vw);
  height: min(500px, 48vw);
  bottom: -12%;
  right: -8%;
  background: radial-gradient(circle, rgba(37, 99, 235, 0.5) 0%, rgba(37, 99, 235, 0.08) 48%, transparent 70%);
  animation: orb-drift-2 22s ease-in-out infinite reverse;
}

.glow-orb-3 {
  width: min(380px, 38vw);
  height: min(380px, 38vw);
  top: 40%;
  left: 35%;
  background: radial-gradient(circle, rgba(168, 85, 247, 0.42) 0%, transparent 65%);
  animation: orb-drift-3 13s ease-in-out infinite;
}

.glow-orb-4 {
  width: min(340px, 32vw);
  height: min(340px, 32vw);
  top: 8%;
  right: 5%;
  background: radial-gradient(circle, rgba(255, 180, 80, 0.38) 0%, transparent 68%);
  animation: orb-drift-4 19s ease-in-out infinite;
}

.glow-beam {
  position: absolute;
  width: 180%;
  height: 45%;
  left: -40%;
  filter: blur(52px);
  opacity: 0.4;
  transform: rotate(-14deg);
}

.glow-beam-1 {
  top: 12%;
  background: linear-gradient(90deg, transparent 0%, rgba(246, 130, 31, 0.28) 30%, rgba(96, 165, 250, 0.22) 55%, rgba(246, 130, 31, 0.14) 75%, transparent 100%);
  animation: beam-sweep-1 14s ease-in-out infinite;
}

.glow-beam-2 {
  bottom: 15%;
  background: linear-gradient(90deg, transparent 0%, rgba(139, 92, 246, 0.2) 35%, rgba(246, 130, 31, 0.16) 60%, transparent 100%);
  animation: beam-sweep-2 18s ease-in-out infinite;
}

@keyframes page-ambient {
  0%, 100% { transform: translate(0, 0) scale(1); opacity: 0.85; }
  33% { transform: translate(4%, -3%) scale(1.05); opacity: 1; }
  66% { transform: translate(-3%, 4%) scale(0.98); opacity: 0.75; }
}

@keyframes hero-gradient-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@keyframes orb-drift-1 {
  0% { transform: translate(-8vw, -6vh) scale(0.88); opacity: 0.7; }
  20% { transform: translate(28vw, -18vh) scale(1.12); opacity: 0.95; }
  40% { transform: translate(42vw, 22vh) scale(1.05); opacity: 0.88; }
  60% { transform: translate(12vw, 38vh) scale(0.96); opacity: 0.78; }
  80% { transform: translate(-18vw, 12vh) scale(1.08); opacity: 0.92; }
  100% { transform: translate(-8vw, -6vh) scale(0.88); opacity: 0.7; }
}

@keyframes orb-drift-2 {
  0% { transform: translate(6vw, 8vh) scale(1); opacity: 0.65; }
  25% { transform: translate(-32vw, -12vh) scale(1.14); opacity: 0.9; }
  50% { transform: translate(-48vw, 18vh) scale(0.92); opacity: 0.75; }
  75% { transform: translate(-10vw, 32vh) scale(1.1); opacity: 0.85; }
  100% { transform: translate(6vw, 8vh) scale(1); opacity: 0.65; }
}

@keyframes orb-drift-3 {
  0% { transform: translate(0, 0) scale(0.9); opacity: 0.55; }
  16% { transform: translate(-35vw, -28vh) scale(1.18); opacity: 0.85; }
  33% { transform: translate(30vw, -22vh) scale(1.05); opacity: 0.72; }
  50% { transform: translate(38vw, 30vh) scale(1.12); opacity: 0.9; }
  66% { transform: translate(-28vw, 26vh) scale(0.94); opacity: 0.68; }
  83% { transform: translate(-40vw, -8vh) scale(1.08); opacity: 0.8; }
  100% { transform: translate(0, 0) scale(0.9); opacity: 0.55; }
}

@keyframes orb-drift-4 {
  0%, 100% { transform: translate(0, 0) scale(0.92); opacity: 0.5; }
  20% { transform: translate(-38vw, 20vh) scale(1.15); opacity: 0.82; }
  40% { transform: translate(-22vw, -32vh) scale(1.02); opacity: 0.68; }
  60% { transform: translate(34vw, -14vh) scale(1.2); opacity: 0.88; }
  80% { transform: translate(18vw, 28vh) scale(0.96); opacity: 0.72; }
}

@keyframes beam-sweep-1 {
  0% { transform: rotate(-20deg) translateX(-30%); opacity: 0.22; }
  25% { transform: rotate(-8deg) translateX(0%); opacity: 0.48; }
  50% { transform: rotate(6deg) translateX(28%); opacity: 0.38; }
  75% { transform: rotate(-4deg) translateX(8%); opacity: 0.52; }
  100% { transform: rotate(-20deg) translateX(-30%); opacity: 0.22; }
}

@keyframes beam-sweep-2 {
  0% { transform: rotate(10deg) translateX(25%); opacity: 0.2; }
  25% { transform: rotate(-6deg) translateX(-5%); opacity: 0.42; }
  50% { transform: rotate(-18deg) translateX(-32%); opacity: 0.35; }
  75% { transform: rotate(2deg) translateX(-12%); opacity: 0.5; }
  100% { transform: rotate(10deg) translateX(25%); opacity: 0.2; }
}

.hero-content { position: relative; z-index: 1; max-width: 480px; color: #fff; text-align: center; }
.hero-logo { width: 64px; height: 64px; display: flex; align-items: center; justify-content: center; margin: 0 auto 32px; }
.hero-logo :deep(.app-logo) {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  box-shadow: 0 12px 40px rgba(246, 130, 31, 0.35);
  animation: logo-glow 4s ease-in-out infinite;
}

@keyframes logo-glow {
  0%, 100% { box-shadow: 0 12px 40px rgba(246, 130, 31, 0.35); }
  50% { box-shadow: 0 16px 56px rgba(246, 130, 31, 0.55), 0 0 80px rgba(246, 130, 31, 0.15); }
}

.hero-content h1 { font-size: 48px; font-weight: 600; letter-spacing: -0.035em; line-height: 1.08; margin-bottom: 16px; }
.hero-content p { font-size: 19px; line-height: 1.47; color: rgba(255, 255, 255, 0.72); }
.login-form-panel {
  width: 440px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 40px;
  background: var(--cf-surface);
  position: relative;
  overflow: hidden;
  z-index: 1;
}
.login-form-panel::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse 80% 60% at 0% 50%, rgba(246, 130, 31, 0.06) 0%, transparent 55%);
  pointer-events: none;
  animation: panel-glow 10s ease-in-out infinite;
}
@keyframes panel-glow {
  0%, 100% { opacity: 0.6; }
  50% { opacity: 1; }
}
.login-card { width: 100%; max-width: 340px; position: relative; z-index: 1; }
.login-card h2 { font-size: 28px; font-weight: 600; color: var(--cf-text); margin-bottom: 8px; }
.login-desc { color: var(--cf-text-muted); font-size: 15px; margin-bottom: 28px; }
.field-label { display: block; font-size: 12px; font-weight: 600; color: var(--cf-text-muted); text-transform: uppercase; margin-bottom: 8px; }
.login-btn { width: 100%; height: 48px; font-size: 16px; font-weight: 600; margin-top: 12px; border-radius: var(--apple-radius-pill, 980px) !important; }
@media (max-width: 860px) { .login-page { flex-direction: column; } .login-hero { min-height: 280px; padding: 40px 24px; } .login-form-panel { width: 100%; padding: 32px 24px 48px; } }
@media (prefers-reduced-motion: reduce) {
  .login-page::before, .login-hero::after, .glow-orb, .glow-beam, .hero-logo :deep(.app-logo), .login-form-panel::before { animation: none; }
}
</style>
