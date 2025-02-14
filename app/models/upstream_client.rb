require "net/http"
require "uri"

class UpstreamClient
  URL = ENV.fetch("CACHEMBED_UPSTREAM_URL", "https://api.openai.com/v1/embeddings")

  attr_accessor :api_key

  def initialize(api_key:, model:, dimensions:, targets:)
    @api_key = api_key
    @model = model
    @dimensions = dimensions
    @targets = targets
  end

  def request_body
    body = {
      model: @model,
      input: @targets.map(&:to_hash),
      encoding_format: "base64"
    }
    body[:dimensions] = @dimensions if @dimensions.present?
    body
  end

  def post
    conn = Faraday.new(url: URL) do |faraday|
      faraday.request :json
      faraday.response :json, parser_options: { symbolize_names: true }
      faraday.adapter Faraday.default_adapter
    end

    response = conn.post do |req|
      req.headers["Authorization"] = "Bearer #{@api_key}"
      req.headers["Content-Type"] = "application/json"
      req.body = request_body
    end

    json_response = JSON.parse(response.body, symbolize_names: true)

    raise "Failed to get embedding from upstream: #{response.status}: #{response.body}" unless response.success?

    UpstreamResponse.new(body: json_response, targets: @targets, model: @model, dimensions: @dimensions)
  end
end
