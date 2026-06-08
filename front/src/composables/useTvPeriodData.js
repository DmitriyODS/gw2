import { ref } from 'vue'
import { getStatsCommon, getStatsExtended } from '@/api/stats.js'

export function useTvPeriodData(makeRange) {
  const commonByPeriod = ref({})
  const extendedByPeriod = ref({})
  const loading = ref(false)
  const periodLoadPromises = new Map()

  async function loadPeriod(period, { silent = false } = {}) {
    if (periodLoadPromises.has(period)) return periodLoadPromises.get(period)
    const { from, to } = makeRange(period)
    if (!silent) loading.value = true

    const request = (async () => {
      try {
        const [common, extended] = await Promise.all([
          getStatsCommon(from, to),
          getStatsExtended(from, to),
        ])
        commonByPeriod.value = { ...commonByPeriod.value, [period]: common }
        extendedByPeriod.value = { ...extendedByPeriod.value, [period]: extended }
      } catch {
        // TV mode is unattended; keep the last good frame.
      } finally {
        if (!silent) loading.value = false
        periodLoadPromises.delete(period)
      }
    })()

    periodLoadPromises.set(period, request)
    return request
  }

  return {
    commonByPeriod,
    extendedByPeriod,
    loading,
    loadPeriod,
  }
}
