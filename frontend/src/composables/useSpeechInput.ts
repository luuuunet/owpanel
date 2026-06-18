import { onBeforeUnmount, ref } from 'vue'

type SpeechRecognitionLike = {
  lang: string
  continuous: boolean
  interimResults: boolean
  maxAlternatives: number
  start: () => void
  stop: () => void
  abort: () => void
  onresult: ((event: any) => void) | null
  onerror: ((event: any) => void) | null
  onend: (() => void) | null
  onstart: (() => void) | null
}

function getSpeechRecognitionCtor(): (new () => SpeechRecognitionLike) | null {
  const w = window as Window & {
    SpeechRecognition?: new () => SpeechRecognitionLike
    webkitSpeechRecognition?: new () => SpeechRecognitionLike
  }
  return w.SpeechRecognition || w.webkitSpeechRecognition || null
}

export function speechLangFromLocale(locale: string) {
  if (locale.startsWith('zh-TW') || locale === 'zh-HK') return 'zh-TW'
  if (locale.startsWith('zh')) return 'zh-CN'
  if (locale.startsWith('ja')) return 'ja-JP'
  if (locale.startsWith('ko')) return 'ko-KR'
  return 'en-US'
}

export function appendSpeechTranscript(current: string, chunk: string) {
  const add = chunk.trim()
  if (!add) return current
  if (!current.trim()) return add
  return `${current.replace(/\s+$/, '')} ${add}`
}

export function useSpeechInput(getLang: () => string) {
  const supported = ref(!!getSpeechRecognitionCtor())
  const listening = ref(false)
  const interimText = ref('')

  let recognition: SpeechRecognitionLike | null = null

  function ensureRecognition() {
    const Ctor = getSpeechRecognitionCtor()
    if (!Ctor) return null
    if (!recognition) {
      recognition = new Ctor()
      recognition.continuous = false
      recognition.interimResults = true
      recognition.maxAlternatives = 1
    }
    recognition.lang = speechLangFromLocale(getLang())
    return recognition
  }

  function stop() {
    if (!recognition || !listening.value) return
    try {
      recognition.stop()
    } catch {
      /* ignore */
    }
  }

  function start(onFinal: (text: string) => void, onError: (key: string) => void) {
    const rec = ensureRecognition()
    if (!rec) {
      onError('unsupported')
      return
    }
    if (listening.value) {
      stop()
      return
    }

    interimText.value = ''
    rec.onstart = () => {
      listening.value = true
    }
    rec.onresult = (event: any) => {
      let interim = ''
      let finalText = ''
      for (let i = event.resultIndex; i < event.results.length; i++) {
        const result = event.results[i]
        const text = String(result?.[0]?.transcript || '')
        if (result.isFinal) finalText += text
        else interim += text
      }
      interimText.value = interim.trim()
      if (finalText.trim()) {
        onFinal(finalText.trim())
        interimText.value = ''
      }
    }
    rec.onerror = (event: any) => {
      const code = String(event?.error || 'unknown')
      if (code === 'aborted') return
      if (code === 'not-allowed' || code === 'service-not-allowed') onError('denied')
      else if (code === 'no-speech') onError('noSpeech')
      else onError('failed')
    }
    rec.onend = () => {
      listening.value = false
      interimText.value = ''
    }

    try {
      rec.start()
    } catch {
      onError('failed')
      listening.value = false
    }
  }

  onBeforeUnmount(() => {
    if (recognition) {
      try {
        recognition.abort()
      } catch {
        /* ignore */
      }
    }
  })

  return { supported, listening, interimText, start, stop }
}
