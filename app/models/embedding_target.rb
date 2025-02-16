class EmbeddingTarget
  class InvalidInputError < StandardError; end

  def initialize(value)
    @value = value
  end

  def to_hash
    @value
  end

  def self.build_targets!(input)
    if input.is_a?(String)
      [ new(input) ]
    elsif input.is_a?(Array) && input.all? { |v| v.is_a?(Integer) }
      [ new(input) ]
    elsif input.is_a?(Array) && input.all? { |v| v.is_a?(String) }
      input.map { |str| new(str) }
    elsif input.is_a?(Array) && input.all? { |v| v.is_a?(Array) && v.all? { |j| j.is_a?(Integer) } }
      input.map { |tokens| new(tokens) }
    else
      raise "Invalid input format: #{input}, allowed formats: String, Array of Integers, Array of Strings, Array of Arrays of Integers"
    end
  end

  def sha1sum
    @sha1sum ||= Digest::SHA1.hexdigest(sha1sum_source)
  end
  
  def input_length
    if is_string?
      @value.bytesize
    else
      @value.size
    end
  end

  private

  def sha1sum_source
    if is_string?
      @value
    else
      @value.join(",")
    end
  end

  def is_string?
    @value.is_a?(String)
  end

  def is_token?
    !is_string?
  end
end
