// Client-side auth plugin - initializes Keycloak on app start
export default defineNuxtPlugin(async () => {
  const { init } = useAuth()

  // Initialize Keycloak authentication
  await init()
})
