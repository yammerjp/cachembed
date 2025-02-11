class V1::EmbeddingsController < ApplicationController
  skip_before_action :verify_authenticity_token
  before_action :require_api_key

  rescue_from EmbeddingTarget::InvalidInputError do |e|
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
    params.require(:embedding)[:input]
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
