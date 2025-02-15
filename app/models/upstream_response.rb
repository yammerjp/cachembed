require "base64"

class UpstreamResponse
  attr_reader :body, :targets, :model

  def initialize(body:, targets:, model:)
    @body = body
    @targets = targets
    @model = model
  end

  # targets の順番に対応した sha1sum と embedding のペアを返す
  def vector_cache_hashes
    @targets.zip(body[:data]).map do |target, item|
      {
        input_hash: target.sha1sum,
        content: base64_decode(item[:embedding]),
        model: @model,
        dimensions: dimensions
      }
    end
  end

  def prompt_tokens
    @body[:usage][:prompt_tokens]
  end

  def total_tokens
    @body[:usage][:total_tokens]
  end
  
  def dimensions
    @dimensions ||= base64_decode(body[:data].first[:embedding]).unpack("f*").size
  end
  
  private
  
  def base64_decode(content)
    Base64.strict_decode64(content)
  end
end
