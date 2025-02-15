require "base64"
# ! /usr/bin/env ruby

# 次のコードをRubyで書き直してください
# function base64ToFloat(base64) {
#   const buffer = Buffer.from(base64, 'base64')
#   const result = new Array(buffer.length / 4)
#
#   for (let i = 0; i < buffer.length; i += 4) {
#     const bits = buffer[i] |
#       (buffer[i + 1] << 8) |
#       (buffer[i + 2] << 16) |
#       (buffer[i + 3] << 24)
#     result[i/4] = new Float32Array(new Uint32Array([bits]).buffer)[0]
#   }
#
#   return result
# }
def float_to_base64(float_array)
  buffer = float_array.pack("e*")
  Base64.strict_encode64(buffer)
end

def binary_to_base64(binary)
  Base64.strict_encode64(binary)
end

def base64_to_binary(base64)
  Base64.strict_decode64(base64)
end

def binary_to_float(binary)
  binary.unpack("e*")
end

def float_to_binary(float_array)
  float_array.pack("e*")
end

def base64_to_float(base64)
  binary_to_float(base64_to_binary(base64))
end

def float_to_base64(float_array)
  binary_to_base64(float_to_binary(float_array))
end

puts float_to_base64([ 0.125, 0.25, 0.5 ])
puts base64_to_float(float_to_base64([ 0.125, 0.25, 0.5 ]))

puts float_to_base64([ 0.375, 0.75, 0.875 ])
puts base64_to_float(float_to_base64([ 0.375, 0.75, 0.875 ]))

puts float_to_base64([ 0.875, 0.9375, 0.15625 ])
puts base64_to_float(float_to_base64([ 0.875, 0.9375, 0.15625 ]))
