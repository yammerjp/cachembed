class EmbeddingForm
  include ActiveModel::Model
  include ActiveModel::Attributes

  attr_accessor :model, :dimensions, :encoding_format, :api_key, :targets, :input
  attr_reader :prompt_tokens, :total_tokens

  MODEL_NAMES = ENV.fetch("CACHEMBED_ALLOWED_MODELS", "text-embedding-ada-002,text-embedding-3-small,text-embedding-3-large").split(",")
  #  MODEL_NAMES = ENV.fetch("CACHEMBED_ALLOWED_MODELS", "unknown").split(",")
  ENCODING_FORMATS = %w[float base64].freeze
  DEFAULT_ENCODING_FORMAT = ENCODING_FORMATS.first

  validates :model, presence: true, inclusion: { in: MODEL_NAMES }
  validates :dimensions, numericality: { only_integer: true, greater_than: 1, less_than: 10_000 }, allow_nil: true
  validates :encoding_format, inclusion: { in: ENCODING_FORMATS }, allow_nil: true

  API_KEY_PATTERN = ENV.fetch("CACHEMBED_API_KEY_PATTERN", "^sk-[a-zA-Z0-9_-]+$")

  validates :api_key, presence: true, format: { with: /\A#{API_KEY_PATTERN}\z/ }
  validates :targets, presence: true

  def initialize(attributes = {})
    super
    self.encoding_format ||= DEFAULT_ENCODING_FORMAT
    self.targets = EmbeddingTarget.build_targets!(attributes[:input])
    @prompt_tokens = 0
    @total_tokens = 0
  end

  def save!
    raise ActiveRecord::RecordInvalid.new(self) unless valid?

    save_embedding_requests!

    vector_by_sha1sum = cached_vectors.index_by(&:input_hash)

    if upstream_targets.any?
      response = upstream_client.post
      upstream_vectors = VectorCache.import_from_response!(response)
      if dimensions.nil? && default_dimensions.nil?
        save_default_dimensions!(upstream_vectors.first.dimensions)
      end
      @prompt_tokens = response.prompt_tokens
      @total_tokens = response.total_tokens
      upstream_vectors.each do |vector|
        vector_by_sha1sum[vector.input_hash] = vector
      end
    end

    targets.map.with_index do |target, index|
      {
        object: "embedding",
        embedding: vector_by_sha1sum[target.sha1sum].formatted_content(encoding_format),
        index: index
      }
    end
  end

  private

  def cached_vectors
    @cached_vectors ||= VectorCache.where(input_hash: targets.map(&:sha1sum), model: model, dimensions: dimensions || default_dimensions).all
  end

  def upstream_targets
    cached_sha1sums = cached_vectors.map(&:input_hash)
    targets.reject { |target| cached_sha1sums.include?(target.sha1sum) }
  end

  def upstream_client
    UpstreamClient.new(
      api_key: api_key,
      model: model,
      dimensions: dimensions,
      targets: upstream_targets,
    )
  end

  def default_dimensions
    @default_dimensions ||= EmbeddingModel.find_by(name: model)&.default_dimensions
  end

  def save_default_dimensions!(d)
    EmbeddingModel.create!(name: model, default_dimensions: d)
  end

  def save_embedding_requests!
    EmbeddingRequest.insert_all!(
      targets.map do |target|
        {
          input_hash: target.sha1sum,
          input_length: target.input_length,
          model: model,
          dimensions: dimensions
        }
      end
    )
  end
end
