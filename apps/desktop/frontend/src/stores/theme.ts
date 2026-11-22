import { defineStore } from 'pinia'

export type ThemeMode = 'dark' | 'light' | 'system'

const THEME_KEY = 'act.theme'

function systemIsDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function resolved(mode: ThemeMode): 'dark' | 'light' {
  return mode === 'system' ? (systemIsDark() ? 'dark' : 'light') : mode
}

function savedMode(): ThemeMode {
  const value = localStorage.getItem(THEME_KEY)
  return value === 'light' || value === 'system' || value === 'dark' ? value : 'system'
}

function syncNativeWindowBackground(theme: 'dark' | 'light') {
  if (!('_wails' in window)) return
  const rgb = theme === 'dark' ? [30, 30, 30] : [243, 243, 243]
  // @ts-expect-error Wails injects this module into the webview at runtime.
  void import(/* @vite-ignore */ '/wails/runtime.js')
    .then((runtime: { Window?: { SetBackgroundColour?: (r: number, g: number, b: number, a: number) => void } }) => {
      runtime.Window?.SetBackgroundColour?.(rgb[0], rgb[1], rgb[2], 255)
    })
    .catch(() => {
      // Plain browser previews do not expose the Wails runtime.
    })
}

export const useThemeStore = defineStore('theme', {
  state: () => ({
    mode: savedMode() as ThemeMode,
  }),
  getters: {
    resolvedTheme: (state) => resolved(state.mode),
  },
  actions: {
    applyTheme() {
      const theme = resolved(this.mode)
      document.documentElement.classList.toggle('dark', theme === 'dark')
      document.documentElement.classList.toggle('light', theme === 'light')
      document.documentElement.dataset.theme = theme
      syncNativeWindowBackground(theme)
    },
    setMode(mode: ThemeMode) {
      this.mode = mode
      localStorage.setItem(THEME_KEY, mode)
      this.applyTheme()
    },
    initialize() {
      this.applyTheme()
      const media = window.matchMedia('(prefers-color-scheme: dark)')
      media.addEventListener('change', () => {
        if (this.mode === 'system') {
          this.applyTheme()
        }
      })
    },
  },
})
