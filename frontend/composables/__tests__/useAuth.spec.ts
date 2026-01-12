import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Mock keycloak-js
const mockKeycloak = {
  init: vi.fn(),
  login: vi.fn(),
  logout: vi.fn(),
  updateToken: vi.fn(),
  token: 'mock-token',
  tokenParsed: {
    sub: 'user-123',
    email: 'test@example.com',
    name: 'Test User',
    realm_access: { roles: ['user', 'admin'] },
    tenant_id: 'tenant-123',
  },
}

vi.mock('keycloak-js', () => ({
  default: vi.fn().mockImplementation(() => mockKeycloak),
}))

// Mock useRuntimeConfig
vi.mock('#app', () => ({
  useRuntimeConfig: () => ({
    public: {
      keycloakUrl: 'http://localhost:8180',
      keycloakRealm: 'test-realm',
      keycloakClientId: 'test-client',
    },
  }),
}))

// Mock window.location
const mockLocation = {
  origin: 'http://localhost:3000',
}
Object.defineProperty(global, 'window', {
  value: { location: mockLocation },
  writable: true,
})

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
}
Object.defineProperty(global, 'localStorage', {
  value: localStorageMock,
  writable: true,
})

// Mock import.meta.client
vi.stubGlobal('import', {
  meta: {
    client: true,
  },
})

describe('useAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorageMock.getItem.mockReturnValue(null)
    // Reset keycloak mock state
    mockKeycloak.init.mockReset()
    mockKeycloak.login.mockReset()
    mockKeycloak.logout.mockReset()
    mockKeycloak.updateToken.mockReset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('hasRole', () => {
    it('should return false when user is null', async () => {
      // Import fresh instance
      const { useAuth } = await import('../useAuth')
      const auth = useAuth()

      // User is null initially
      expect(auth.hasRole('admin')).toBe(false)
    })
  })

  describe('hasAnyRole', () => {
    it('should return false when user is null', async () => {
      const { useAuth } = await import('../useAuth')
      const auth = useAuth()

      expect(auth.hasAnyRole(['admin', 'user'])).toBe(false)
    })
  })

  describe('isAdmin', () => {
    it('should return false when user is null', async () => {
      const { useAuth } = await import('../useAuth')
      const auth = useAuth()

      expect(auth.isAdmin()).toBe(false)
    })
  })

  describe('setDevRole', () => {
    it('should update devRole and persist to localStorage', async () => {
      const { useAuth } = await import('../useAuth')
      const auth = useAuth()

      auth.setDevRole('user')

      expect(localStorageMock.setItem).toHaveBeenCalledWith('devRole', 'user')
    })

    it('should switch to admin role', async () => {
      const { useAuth } = await import('../useAuth')
      const auth = useAuth()

      auth.setDevRole('admin')

      expect(localStorageMock.setItem).toHaveBeenCalledWith('devRole', 'admin')
    })
  })

  describe('getToken', () => {
    it('should return undefined when not authenticated', async () => {
      const { useAuth } = await import('../useAuth')
      const auth = useAuth()

      const token = await auth.getToken()

      expect(token).toBeUndefined()
    })
  })

  describe('initial state', () => {
    it('should have isLoading as true initially', async () => {
      const { useAuth } = await import('../useAuth')
      const auth = useAuth()

      expect(auth.isLoading.value).toBe(true)
    })

    it('should have isAuthenticated as false initially', async () => {
      const { useAuth } = await import('../useAuth')
      const auth = useAuth()

      expect(auth.isAuthenticated.value).toBe(false)
    })
  })

  describe('init', () => {
    it('should fall back to dev mode when Keycloak fails', async () => {
      // Since Keycloak is mocked but initialization fails in test environment,
      // we expect the auth to fall back to dev mode
      const { useAuth } = await import('../useAuth')
      const auth = useAuth()

      const result = await auth.init()

      // Should fall back to dev mode on failure
      expect(result).toBe(false)
      expect(auth.isDevMode.value).toBe(true)
    })
  })
})
