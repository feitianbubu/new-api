import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'

export function hasApiKeyPermission(): boolean {
  const user = useAuthStore.getState().auth.user
  if (!user) return false
  if (typeof user.role === 'number' && user.role >= ROLE.ADMIN) {
    return true
  }
  return Boolean((user as unknown as { api_key?: boolean }).api_key)
}
