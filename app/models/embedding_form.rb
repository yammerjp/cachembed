class EmbeddingForm
  include ActiveModel::Model
  include ActiveModel::Attributes

  attr_reader :model, :dimensions, :encoding_format, :api_key, :targets

  MODEL_NAMES = %w[text-embedding-ada-002 text-embedding-3-small text-embedding-3-large].freeze
  validates :model, presence: true, inclusion: { in: MODEL_NAMES }

  validates :dimensions, numericality: { only_integer: true, greater_than: 1, less_than: 10000 }, allow_nil: true

  ENCODING_FORMATS = %w[float base64].freeze
  DEFAULT_ENCODING_FORMAT = ENCODING_FORMATS.first
  validates :encoding_format, inclusion: { in: ENCODING_FORMATS }, allow_nil: true

  def initialize(attributes = {})
    @model = attributes[:model]
    @dimensions = attributes[:dimensions]
    @encoding_format = attributes[:encoding_format]
    @api_key = attributes[:api_key]
    @targets = EmbeddingTarget.build_targets!(attributes[:input])
    @vector_by_sha1sum = {}
    @usage = {
      prompt_tokens: 0,
      total_tokens: 0,
    }
  end

  def do_embedding
    load_vectors_from_db
    load_vectors_from_upstream

    {
      model: model,
      data: targets.map.with_index do |target, index|
        {
          object: "embedding",
          index: index,
          embedding: @vector_by_sha1sum[target.sha1sum].content_with(encoding_format)
        }
      end,
      usage: @usage
    }
  end

  private

  def unloaded_targets
    targets.select { |t| @vector_by_sha1sum[t.sha1sum].blank? }
  end

  def load_vectors_from_db
    vectors = VectorCache.where(input_hash: targets.map(&:sha1sum), model: model, dimensions: dimensions || VectorCache::DEFAULT_DIMENSIONS).all
    vectors.each do |vector|
      @vector_by_sha1sum[vector.input_hash] = vector
    end
  end

  def load_vectors_from_upstream
    upstream_targets = unloaded_targets

    if upstream_targets.empty?
      return
    end

    upstream_client = UpstreamClient.new(api_key: api_key, model: model, dimensions: dimensions, targets: upstream_targets)
    upstream_client.post

    new_vectors = upstream_client.save_response_to_vector!
    new_vectors.each do |vector|
      @vector_by_sha1sum[vector.input_hash] = vector
    end
    @usage = upstream_client.usage
  end
end
