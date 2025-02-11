class EmbeddingForm
  include ActiveModel::Model
  include ActiveModel::Attributes

  attr_accessor :model, :dimensions, :input, :encoding_format, :api_key

  MODEL_NAMES = %w[text-embedding-ada-002 text-embedding-3-small text-embedding-3-large].freeze
  validates :model, presence: true, inclusion: { in: MODEL_NAMES }

  validates :dimensions, numericality: { only_integer: true, greater_than: 0, less_than: 10000 }
  validates :input, presence: true

  ENCODING_FORMATS = %w[float base64].freeze
  DEFAULT_ENCODING_FORMAT = ENCODING_FORMATS.first
  validates :encoding_format, presence: true, inclusion: { in: ENCODING_FORMATS }

  def initialize(attributes = {})
    super
    @model = attributes[:model]
    @dimensions = attributes[:dimensions]
    @input = attributes[:input]
    @encoding_format = attributes[:encoding_format]
    @api_key = attributes[:api_key]
  end

  def do_embedding
    upstream_client = UpstreamClient.new(api_key: api_key, embedding_form: self.to_h)
    upstream_client.post
  end

  def to_h
    hash = {
      model: model,
      input: input,
    }

    if dimensions.present?
      hash[:dimensions] = dimensions
    end

    if encoding_format.present?
      hash[:encoding_format] = encoding_format
    end

    hash
  end
end
