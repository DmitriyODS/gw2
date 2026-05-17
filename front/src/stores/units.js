import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getActiveUnit, stopUnit as apiStop } from '@/api/units.js'

export const useUnitsStore = defineStore('units', () => {
  const activeUnit = ref(null)

  async function fetchActiveUnit() {
    try {
      const data = await getActiveUnit()
      activeUnit.value = data
    } catch {
      activeUnit.value = null
    }
  }

  function setActiveUnit(unit) { activeUnit.value = unit }
  function clearActiveUnit() { activeUnit.value = null }

  async function stop() {
    if (!activeUnit.value) return
    await apiStop(activeUnit.value.id)
    activeUnit.value = null
  }

  return { activeUnit, fetchActiveUnit, setActiveUnit, clearActiveUnit, stop }
})
