class EmbeddingForm
  class InvalidInputError < StandardError; end

  include ActiveModel::Model
  include ActiveModel::Attributes

  attr_accessor :model, :dimensions, :encoding_format, :api_key
  attr_reader :input

  MODEL_NAMES = %w[text-embedding-ada-002 text-embedding-3-small text-embedding-3-large].freeze
  validates :model, presence: true, inclusion: { in: MODEL_NAMES }

  validates :dimensions, numericality: { only_integer: true, greater_than: 0, less_than: 10000 }, allow_nil: true

  validates :input, presence: true
  validate :validate_input

  ENCODING_FORMATS = %w[float base64].freeze
  DEFAULT_ENCODING_FORMAT = ENCODING_FORMATS.first
  validates :encoding_format, inclusion: { in: ENCODING_FORMATS }, allow_nil: true

  def initialize(attributes = {})
    super()
    @model = attributes[:model]
    @dimensions = attributes[:dimensions]
    @encoding_format = attributes[:encoding_format]
    @api_key = attributes[:api_key]
    self.input = attributes[:input]
  end

  def input=(value)
    @input = value.is_a?(Input) ? value : Input.new(value)
  end

  def to_hash
    hash = {
      model: model,
      input: input&.to_hash
    }

    hash[:dimensions] = dimensions if dimensions.present?
    hash[:encoding_format] = encoding_format if encoding_format.present?

    hash
  end

  def do_embedding
    upstream_client = UpstreamClient.new(api_key: api_key, embedding_form: self)
    upstream_client.post
  end

  private

  def validate_input
    return unless input.present?
    return if input.valid?

    input.errors.each do |error|
      errors.add(:input, error.message)
    end
  end
end
