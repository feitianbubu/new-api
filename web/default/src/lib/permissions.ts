import { useAuthStore, type AuthUser } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'

function check(user: AuthUser | null | undefined): boolean {
  if (!user) return false
  if (user.role >= ROLE.ADMIN) return true
  return Boolean((user as AuthUser & { api_key?: boolean }).api_key)
}

export function hasApiKeyPermission(): boolean {
  return check(useAuthStore.getState().auth.user)
}

export function useHasApiKeyPermission(): boolean {
  return useAuthStore((s) => check(s.auth.user))
}
