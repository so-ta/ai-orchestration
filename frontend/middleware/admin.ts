// Admin route guard middleware
// Redirects non-admin users to the dashboard
export default defineNuxtRouteMiddleware((to) => {
  const { isAdmin, isLoading } = useAuth()

  // Skip check while loading
  if (isLoading.value) {
    return
  }

  // Check if user has admin role
  if (!isAdmin()) {
    return navigateTo('/')
  }
})
