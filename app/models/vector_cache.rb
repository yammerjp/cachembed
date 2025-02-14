require "base64"

class VectorCache < ApplicationRecord
  DEFAULT_DIMENSIONS = 0

  validates :sha1sum, presence: true, uniqueness: true
  validates :embedding, presence: true
  validates :model, presence: true
  validates :dimensions, presence: true

  def self.import_from_response!(response)
    self.insert_all(response.vector_cache_hashes)
  end

  def base64_content
    Base64.strict_encode64(content)
  end

  def float_array_content
    content.unpack("f*")
  end

  def formatted_content(format)
    if format == "base64"
      base64_content
    else
      # default
      float_array_content
    end
  end
end
