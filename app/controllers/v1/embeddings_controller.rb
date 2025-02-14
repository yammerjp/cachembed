class V1::EmbeddingsController < ApplicationController
  skip_before_action :verify_authenticity_token
  before_action :require_api_key

  rescue_from EmbeddingTarget::InvalidInputError, with: -> { render_error("Invalid input", :bad_request) }
  rescue_from ActiveRecord::RecordInvalid do |e|
    render_error(e.record.errors.full_messages, :unprocessable_entity)
  end

  def create
    form = EmbeddingForm.new(create_params)
    @embeddings = form.save!
    @model = form.model
    @prompt_tokens = form.prompt_tokens
    @total_tokens = form.total_tokens
  end

  private

  def create_params
    embedding_params.permit(:model, :dimensions, :encoding_format).merge(api_key: api_key, input: input_param)
  end

  def embedding_params
    if params[:embedding].present?
      params[:embedding]
    else
      params
    end
  end

  def input_param
    # input is an array of strings or arrays of integers, skip the validation of strong parameters
    embedding_params[:input]
  end

  def require_api_key
    render_error("Unauthorized", :unauthorized) unless api_key.present?
  end

  def api_key
    request.headers["Authorization"]&.split(" ")&.last
  end

  def render_error(messages, status)
    render json: { errors: Array(messages) }, status: status
  end
end
