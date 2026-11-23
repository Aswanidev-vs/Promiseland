export {}

declare module 'vue-router' {
  interface RouteMeta {
    /** Redirect anonymous visitors (used by R1 account/profile routes). */
    requiresAuth?: boolean
    /** Redirect signed-in visitors (used by R1 register/login routes). */
    guestOnly?: boolean
    /** Document title for the route. */
    title?: string
  }
}
