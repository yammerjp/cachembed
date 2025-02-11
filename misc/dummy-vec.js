#!/usr/bin/env node

function base64ToFloat(base64) {
  const buffer = Buffer.from(base64, 'base64')
  const result = new Array(buffer.length / 4)

  for (let i = 0; i < buffer.length; i += 4) {
    const bits = buffer[i] |
      (buffer[i + 1] << 8) |
      (buffer[i + 2] << 16) |
      (buffer[i + 3] << 24)
    result[i/4] = new Float32Array(new Uint32Array([bits]).buffer)[0]
  }

  return result
}

function floatToBase64(floatArray) {
  const buffer = Buffer.alloc(floatArray.length * 4)

  floatArray.forEach((value, i) => {
    const bits = new Uint32Array(new Float32Array([value]).buffer)[0]
    buffer[i * 4] = bits & 0xff
    buffer[i * 4 + 1] = (bits >> 8) & 0xff
    buffer[i * 4 + 2] = (bits >> 16) & 0xff
    buffer[i * 4 + 3] = (bits >> 24) & 0xff
  })

  return buffer.toString('base64')
}

const testArray = [0.125, 0.25, 0.5] // 1/8, 1/4, 1/2
const base64 = floatToBase64(testArray)
console.log('Base64:', base64)
console.log('Decoded:', base64ToFloat(base64))

const testArray2 = [0.375, 0.75, 0.875] // 3/8, 3/4, 7/8
const base642 = floatToBase64(testArray2)
console.log('Base64:', base642)
console.log('Decoded:', base64ToFloat(base642))

const testArray3 = [0.875, 0.9375, 0.15625] // 7/8, 15/16, 5/32
const base643 = floatToBase64(testArray3)
console.log('Base64:', base643)
console.log('Decoded:', base64ToFloat(base643))
