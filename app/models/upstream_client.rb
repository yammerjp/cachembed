require "net/http"
require "uri"

class UpstreamClient
  URL = ENV.fetch("CACHEMBED_UPSTREAM_URL", "https://api.openai.com/v1/embeddings")

  attr_accessor :api_key, :embedding_form

  def initialize(api_key:, embedding_form:)
    @api_key = api_key
    @embedding_form = embedding_form
  end

  def post
    uri = URI.parse(URL)
    req = Net::HTTP::Post.new(uri.path)
    req["Authorization"] = "Bearer #{@api_key}"
    req["Content-Type"] = "application/json"
    req.body = @embedding_form.to_json

    response = Net::HTTP.start(uri.host, uri.port, use_ssl: uri.scheme == "https") do |http|
      http.request(req)
    end

    JSON.parse(response.body).deep_symbolize_keys
  end
end
