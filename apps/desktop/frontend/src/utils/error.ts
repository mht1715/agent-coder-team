import axios from 'axios'
import { ApiError } from '../api/client'

type Translate = (key: string) => string

const codeKeys: Record<string, string> = {
  'auth.unauthorized': 'errors.unauthorized',
  'auth.invalid_credentials': 'errors.invalidCredentials',
  'auth.registration_closed': 'errors.registrationClosed',
  'auth.username_exists': 'errors.usernameExists',
  'project.not_found': 'errors.notFound',
  'project.name_exists': 'errors.projectNameExists',
  'agent.plan_exists': 'errors.planExists',
  'agent.not_found': 'errors.notFound',
  'sandbox.unavailable': 'errors.sandboxUnavailable',
  'validation.invalid_request': 'errors.validation',
  internal_error: 'errors.unknown',
}

export function translateApiError(error: unknown, t: Translate): string {
  if (error instanceof ApiError) {
    const key = codeKeys[error.code]
    if (key) return t(key)
    if (error.status === 401) return t('errors.unauthorized')
    if (error.status === 403) return t('errors.forbidden')
    if (error.status === 404) return t('errors.notFound')
    if (error.status === 400) return t('errors.validation')
    return t('errors.unknown')
  }

  if (axios.isAxiosError(error)) {
    if (!error.response) {
      return t('errors.network')
    }
    const body = error.response.data as { code?: string; message?: string } | undefined
    const key = body?.code ? codeKeys[body.code] : undefined
    if (key) {
      return t(key)
    }
    if (body?.message) {
      return body.message
    }
  }
  return t('errors.unknown')
}
