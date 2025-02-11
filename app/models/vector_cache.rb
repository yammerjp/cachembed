class VectorCache < ApplicationRecord
  DEFAULT_DIMENSIONS = 0

  def content_with(format)
    if format == "base64"
      base64_content
    else
      float_array_content
    end
  end

  private

  def base64_content
    Base64.strict_encode64(content)
  end

  def float_array_content
    content.unpack("e*")
  end
end
