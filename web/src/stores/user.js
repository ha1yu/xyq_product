import { defineStore } from 'pinia'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    isLoggedIn: !!localStorage.getItem('token')
  }),
  actions: {
    setToken(token) {
      this.token = token
      this.isLoggedIn = true
      localStorage.setItem('token', token)
    },
    logout() {
      this.token = ''
      this.isLoggedIn = false
      localStorage.removeItem('token')
    }
  }
})
