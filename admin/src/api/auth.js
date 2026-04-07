import request from './request'

export const authApi = {
  login(username, password) {
    return request.post('/auth/admin/login', { username, password })
  },

  logout() {
    return request.post('/auth/logout')
  },

  session() {
    return request.get('/auth/check')
  }
}
