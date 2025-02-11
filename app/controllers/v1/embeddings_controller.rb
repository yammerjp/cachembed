class V1::EmbeddingsController < ApplicationController
  class InvalidInputError < StandardError; end

  skip_before_action :verify_authenticity_token
  before_action :require_api_key

  rescue_from InvalidInputError do |e|
    render json: { error: "Invalid input" }, status: :bad_request
  end

  def create
    form = EmbeddingForm.new(create_params)

    @embedding = form.do_embedding
  end

  private

  def create_params
    params.require(:embedding).permit(:model, :dimensions, :encoding_format).merge(input: input_param, api_key: api_key)
  end

  def input_param
    input = params.require(:embedding)[:input]

    if input.is_a?(String)
      # "hello"
      return input
    end

    if input.is_a?(Array) && input.all? { |i| i.is_a?(Integer) }
      # [1, 2, 3]
      return input
    end

    if input.is_a?(Array) && input.all? { |i| i.is_a?(String) }
      # ["hello", "world"]
      return input
    end

    if input.is_a?(Array) && input.all? { |i| i.is_a?(Array) && i.all? { |j| j.is_a?(Integer) } }
      # [[1, 2, 3], [4, 5, 6]]
      return input
    end

    raise InvalidInputError
  end

  def require_api_key
    unless api_key.present?
      render json: { error: "Unauthorized" }, status: :unauthorized
    end
  end

  def api_key
    request.headers["Authorization"]&.split(" ")&.last
  end
end
