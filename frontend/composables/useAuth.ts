import Keycloak from 'keycloak-js'

// Auth state
const keycloak = ref<Keycloak | null>(null)
const isAuthenticated = ref(false)
const isLoading = ref(true)
const user = ref<{
  id: string
  email: string
  name: string
  roles: string[]
  tenantId: string
} | null>(null)

// Initialize flag to prevent multiple initializations
let initPromise: Promise<boolean> | null = null

export function useAuth() {
  const config = useRuntimeConfig()

  async function init(): Promise<boolean> {
    // Return existing promise if already initializing
    if (initPromise) {
      return initPromise
    }

    // Skip if already initialized
    if (keycloak.value) {
      return isAuthenticated.value
    }

    initPromise = (async () => {
      try {
        const kc = new Keycloak({
          url: config.public.keycloakUrl,
          realm: config.public.keycloakRealm,
          clientId: config.public.keycloakClientId
        })

        // Initialize with check-sso to silently check if user is logged in
        const authenticated = await kc.init({
          onLoad: 'check-sso',
          silentCheckSsoRedirectUri: window.location.origin + '/silent-check-sso.html',
          checkLoginIframe: false,
          pkceMethod: 'S256'
        })

        keycloak.value = kc
        isAuthenticated.value = authenticated

        if (authenticated) {
          await updateUserInfo()
          setupTokenRefresh()
        }

        isLoading.value = false
        return authenticated
      } catch (error) {
        console.error('Keycloak init error:', error)
        isLoading.value = false
        return false
      }
    })()

    return initPromise
  }

  async function login(redirectUri?: string): Promise<void> {
    if (!keycloak.value) {
      await init()
    }
    await keycloak.value?.login({
      redirectUri: redirectUri || window.location.origin
    })
  }

  async function logout(redirectUri?: string): Promise<void> {
    if (!keycloak.value) return
    await keycloak.value.logout({
      redirectUri: redirectUri || window.location.origin
    })
    isAuthenticated.value = false
    user.value = null
  }

  async function getToken(): Promise<string | undefined> {
    if (!keycloak.value || !isAuthenticated.value) {
      return undefined
    }

    try {
      // Refresh token if it expires in the next 30 seconds
      await keycloak.value.updateToken(30)
      return keycloak.value.token
    } catch (error) {
      console.error('Token refresh failed:', error)
      // Token refresh failed, redirect to login
      await login()
      return undefined
    }
  }

  async function updateUserInfo(): Promise<void> {
    if (!keycloak.value || !keycloak.value.tokenParsed) return

    const tokenParsed = keycloak.value.tokenParsed as {
      sub?: string
      email?: string
      name?: string
      preferred_username?: string
      realm_access?: { roles?: string[] }
      tenant_id?: string
    }

    user.value = {
      id: tokenParsed.sub || '',
      email: tokenParsed.email || tokenParsed.preferred_username || '',
      name: tokenParsed.name || tokenParsed.email || '',
      roles: tokenParsed.realm_access?.roles || [],
      tenantId: tokenParsed.tenant_id || ''
    }
  }

  function setupTokenRefresh(): void {
    if (!keycloak.value) return

    // Refresh token every minute
    setInterval(async () => {
      if (keycloak.value && isAuthenticated.value) {
        try {
          const refreshed = await keycloak.value.updateToken(60)
          if (refreshed) {
            console.debug('Token refreshed')
          }
        } catch (error) {
          console.error('Token refresh failed:', error)
          isAuthenticated.value = false
          user.value = null
        }
      }
    }, 60000)
  }

  function hasRole(role: string): boolean {
    return user.value?.roles.includes(role) || false
  }

  function hasAnyRole(roles: string[]): boolean {
    return roles.some(role => hasRole(role))
  }

  return {
    // State
    isAuthenticated: readonly(isAuthenticated),
    isLoading: readonly(isLoading),
    user: readonly(user),
    keycloak: readonly(keycloak),

    // Methods
    init,
    login,
    logout,
    getToken,
    hasRole,
    hasAnyRole
  }
}
