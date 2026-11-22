import axios from 'axios'

type Translate = (key: string) => string

const codeKeys: Record<string, string> = {
  'auth.unauthorized': 'errors.unauthorized',
  'auth.invalid_credentials': 'errors.invalidCredentials',
  'auth.registration_closed': 'errors.registrationClosed',
  'auth.username_exists': 'errors.usernameExists',
  'project.not_found': 'errors.notFound',
  'project.name_exists': 'errors.projectNameExists',
  'validation.invalid_request': 'errors.validation',
  internal_error: 'errors.unknown',
}

export function translateApiError(error: unknown, t: Translate): string {
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
