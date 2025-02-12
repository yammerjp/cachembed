class V1::EmbeddingsController < ApplicationController
  skip_before_action :verify_authenticity_token
  before_action :require_api_key

  rescue_from EmbeddingTarget::InvalidInputError, with: -> { render_error('Invalid input', :bad_request) }
  rescue_from ActiveRecord::RecordInvalid do |e|
    render_error(e.record.errors.full_messages, :unprocessable_entity)
  end

  def create
    form = EmbeddingForm.new(create_params)

    if form.save
      render json: { message: 'Embedding created successfully' }, status: :created
    else
      render_error(form.errors.full_messages, :unprocessable_entity)
    end
  end

  private

  def create_params
    params.require(:embedding).permit(:model, :dimensions, :encoding_format, input: []).merge(api_key: api_key)
  end

  def require_api_key
    render_error('Unauthorized', :unauthorized) unless api_key.present?
  end

  def api_key
    request.headers['Authorization']&.split(' ')&.last
  end

  def render_error(messages, status)
    render json: { errors: Array(messages) }, status: status
  end
end
