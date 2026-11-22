const ACCESS_KEY = 'act.access_token'
const REFRESH_KEY = 'act.refresh_token'

export const tokenStorage = {
  getAccess(): string {
    return localStorage.getItem(ACCESS_KEY) || ''
  },
  getRefresh(): string {
    return localStorage.getItem(REFRESH_KEY) || ''
  },
  set(accessToken: string, refreshToken: string) {
    localStorage.setItem(ACCESS_KEY, accessToken)
    localStorage.setItem(REFRESH_KEY, refreshToken)
  },
  clear() {
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
  },
}