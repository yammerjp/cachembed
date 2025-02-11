require "net/http"
require "uri"

class UpstreamClient
  URL = ENV.fetch("CACHEMBED_UPSTREAM_URL", "https://api.openai.com/v1/embeddings")

  attr_accessor :api_key, :response

  def initialize(api_key:, model:, dimensions:, targets:)
    @api_key = api_key
    @model = model
    @dimensions = dimensions
    @targets = targets
  end

  def request_body
    b = {
      model: @model,
      input: @targets.map(&:to_hash),
      encoding_format: "base64"
    }
    if @dimensions.present?
      b[:dimensions] = @dimensions
    end
    b
  end

  def post
    uri = URI.parse(URL)
    req = Net::HTTP::Post.new(uri.path)
    req["Authorization"] = "Bearer #{@api_key}"
    req["Content-Type"] = "application/json"
    req.body = request_body.to_json

    response = Net::HTTP.start(uri.host, uri.port, use_ssl: uri.scheme == "https") do |http|
      http.request(req)
    end
    if response.code != "200"
      raise "Failed to get embedding from upstream: #{response.code}: #{response.body}"
    end

    @response = JSON.parse(response.body).deep_symbolize_keys
  end

  def save_response_to_vector!
    vectors = @targets.map.with_index do |target, index|
      VectorCache.new(
        input_hash: target.sha1sum,
        model: @model,
        dimensions: @dimensions || VectorCache::DEFAULT_DIMENSIONS,
        content: Base64.decode64(@response[:data][index][:embedding]),
      )
    end

    VectorCache.import vectors
    vectors
  end

  def usage
    @response[:usage]
  end
end
