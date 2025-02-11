class V1::EmbeddingsController < ApplicationController
  skip_before_action :verify_authenticity_token

  def create
    # TODO: 実際の埋め込み処理をここに実装
    @embedding = {
      object: "list",
      data: [
        {
          object: "embedding",
          embedding: [0.0023064255, -0.009327292, 0.015954146],  # サンプル値
          index: 0
        }
      ],
      model: "text-embedding-ada-002",
      usage: {
        prompt_tokens: 8,
        total_tokens: 8
      }
    }
  end

  private

  def create_params
    permitted_params = params.require(:embedding).permit(:model, :dimensions, :input, :encoding_format)
    input = params[:input]
    if input.is_a?(Array)
      permitted_params[:input] = input
    else
      permitted_params[:input] = input
    end
    permitted_params
  end
end
