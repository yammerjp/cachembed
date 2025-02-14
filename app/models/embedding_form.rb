class EmbeddingForm
  include ActiveModel::Model
  include ActiveModel::Attributes

  attr_accessor :model, :dimensions, :encoding_format, :api_key, :targets, :input
  attr_reader :prompt_tokens, :total_tokens

  MODEL_NAMES = %w[text-embedding-ada-002 text-embedding-3-small text-embedding-3-large].freeze
  ENCODING_FORMATS = %w[float base64].freeze
  DEFAULT_ENCODING_FORMAT = ENCODING_FORMATS.first

  validates :model, presence: true, inclusion: { in: MODEL_NAMES }
  validates :dimensions, numericality: { only_integer: true, greater_than: 1, less_than: 10_000 }, allow_nil: true
  validates :encoding_format, inclusion: { in: ENCODING_FORMATS }, allow_nil: true
  validates :api_key, presence: true
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

    embedding_by_sha1sum = {}

    cached_vectors.each do |cached_vector|
      embedding_by_sha1sum[cached_vector.input_hash] = cached_vector.formatted_content(encoding_format)
    end

    if upstream_targets.any?
      response = upstream_client.post
      upstream_vectors = VectorCache.import_from_response!(response)
      @prompt_tokens = response.prompt_tokens
      @total_tokens = response.total_tokens

      upstream_vectors.each do |vector|
        embedding_by_sha1sum[vector.input_hash] = vector.formatted_content(encoding_format)
      end
    end

    targets.map do |target|
      embedding_by_sha1sum[target.sha1sum]
    end
  end

  def cached_vectors
    VectorCache.where(input_hash: targets.map(&:sha1sum), model: model, dimensions: dimensions || 0).all
  end

  def upstream_targets
    cached_sha1sums = cached_vectors.map(&:input_hash)
    targets.reject { |target| cached_sha1sums.include?(target.sha1sum) }
  end

  private

  def upstream_client
    UpstreamClient.new(
      api_key: api_key,
      model: model,
      dimensions: dimensions,
      targets: upstream_targets,
    )
  end
end
