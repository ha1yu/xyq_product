import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000
})

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  res => res.data,
  err => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export default {
  login: (data) => api.post('/login', data),
  getCategories: () => api.get('/categories'),
  getProducts: (params) => api.get('/products', { params }),
  createProduct: (data) => api.post('/products', data),
  updateProduct: (id, data) => api.put(`/products/${id}`, data),
  deleteProduct: (id) => api.delete(`/products/${id}`),
  getPrices: (productId) => api.get(`/products/${productId}/prices`),
  createPrice: (data) => api.post('/prices', data),
  deletePrice: (id) => api.delete(`/prices/${id}`),
  getTrend: (productIds) => api.get('/trend', { params: { product_ids: productIds } })
}
