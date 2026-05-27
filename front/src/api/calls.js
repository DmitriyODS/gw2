import { apiRequest } from './client.js'

export function getIceServers() {
  return apiRequest('/calls/ice-servers')
}

export function getCallHistory() {
  return apiRequest('/calls/history')
}

export function getActiveCall() {
  return apiRequest('/calls/active')
}
