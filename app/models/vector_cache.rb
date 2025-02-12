class VectorCache < ApplicationRecord
  validates :sha1sum, presence: true, uniqueness: true
  validates :embedding, presence: true
  validates :model, presence: true
  validates :dimensions, presence: true

  def self.import_from_response!(response)
    transaction do
      response.vectors.each do |sha1sum, embedding|
        create!(
          sha1sum: sha1sum,
          embedding: embedding,
          model: response.model,
          dimensions: response.dimensions
        )
      end
    end
  end
end
