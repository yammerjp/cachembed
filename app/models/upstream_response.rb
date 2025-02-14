require "base64"

class UpstreamResponse
  def initialize(body:, targets:, model:, dimensions:)
    @body = body
    @targets = targets
    @model = model
    @dimensions = dimensions
  end

  # targets の順番に対応した sha1sum と embedding のペアを返す
  def vector_cache_hashes
    @targets.zip(@body[:data]).map do |target, item|
      {
        input_hash: target.sha1sum,
        content: Base64.strict_decode64(item[:embedding]),
        model: @model,
        dimensions: @dimensions || VectorCache::DEFAULT_DIMENSIONS
      }
    end
  end

  def prompt_tokens
    @body[:usage][:prompt_tokens]
  end

  def total_tokens
    @body[:usage][:total_tokens]
  end
end
