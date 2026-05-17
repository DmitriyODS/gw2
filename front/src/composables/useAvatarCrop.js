import { ref, reactive } from 'vue'

export function useAvatarCrop() {
  const imageDataUrl = ref(null)
  const cropX = ref(0)
  const cropY = ref(0)
  const cropSize = ref(100)
  const canvasRef = ref(null)
  const imgEl = ref(null)

  function loadImage(file) {
    return new Promise((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = (e) => {
        imageDataUrl.value = e.target.result
        resolve(e.target.result)
      }
      reader.onerror = reject
      reader.readAsDataURL(file)
    })
  }

  async function getCroppedBlob(targetSize = 400) {
    const canvas = document.createElement('canvas')
    canvas.width = targetSize
    canvas.height = targetSize
    const ctx = canvas.getContext('2d')

    const img = new Image()
    img.src = imageDataUrl.value
    await new Promise(r => { img.onload = r })

    const scale = img.naturalWidth / (imgEl.value?.offsetWidth || img.naturalWidth)

    ctx.drawImage(
      img,
      cropX.value * scale, cropY.value * scale,
      cropSize.value * scale, cropSize.value * scale,
      0, 0, targetSize, targetSize
    )

    return new Promise((resolve) => {
      canvas.toBlob((blob) => resolve(blob), 'image/jpeg', 0.9)
    })
  }

  async function compressToLimit(blob, maxBytes = 2 * 1024 * 1024) {
    if (blob.size <= maxBytes) return blob
    const canvas = document.createElement('canvas')
    const img = new Image()
    const url = URL.createObjectURL(blob)
    img.src = url
    await new Promise(r => { img.onload = r })
    URL.revokeObjectURL(url)
    let quality = 0.8
    let result = blob
    while (result.size > maxBytes && quality > 0.1) {
      canvas.width = img.naturalWidth
      canvas.height = img.naturalHeight
      const ctx = canvas.getContext('2d')
      ctx.drawImage(img, 0, 0)
      result = await new Promise(r => canvas.toBlob(r, 'image/jpeg', quality))
      quality -= 0.1
    }
    return result
  }

  function reset() {
    imageDataUrl.value = null
    cropX.value = 0
    cropY.value = 0
    cropSize.value = 100
  }

  return {
    imageDataUrl, cropX, cropY, cropSize, canvasRef, imgEl,
    loadImage, getCroppedBlob, compressToLimit, reset
  }
}
