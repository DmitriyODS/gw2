import { apiRequest } from './client.js'

export const changelogApi = {
  get: () => apiRequest('/changelog'),
}
