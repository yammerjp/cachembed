class EmbeddingForm
  include ActiveModel::Model
  include ActiveModel::Attributes

  attr_accessor :model, :dimensions, :encoding_format, :api_key, :targets

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
  end

  def save!
    raise ActiveRecord::RecordInvalid.new(self) unless valid?

    response = upstream_client.post
    VectorCache.import_from_response!(response)
  end

  private

  def upstream_client
    UpstreamClient.new(
      api_key: api_key,
      model: model,
      dimensions: dimensions,
      targets: targets
    )
  end
end
