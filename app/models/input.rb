class Input
  include ActiveModel::Model
  include ActiveModel::Attributes

  attribute :input

  validates :input, presence: true
  validate :validate_input_format

  def initialize(input)
    super()
    self.input = input
  end

  def input_format
    self.class.input_format(input)
  end

  def self.input_format(input)
    if input.is_a?(String)
      return :string
    end

    if input.is_a?(Array) && input.all? { |i| i.is_a?(Integer) }
      return :tokens
    end

    if input.is_a?(Array) && input.all? { |i| i.is_a?(String) }
      return :strings
    end

    if input.is_a?(Array) && input.all? { |i| i.is_a?(Array) && i.all? { |j| j.is_a?(Integer) } }
      return :tokens
    end

    nil
  end

  def to_hash
    input
  end

  private

  def validate_input_format
    errors.add(:input, 'has invalid format') unless input_format
  end
end
