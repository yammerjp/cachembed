class CreateEmbeddingRequests < ActiveRecord::Migration[8.0]
  def change
    create_table :embedding_requests do |t|
      t.string :input_hash, null: false, limit: 40
      t.integer :input_length, null: false
      t.integer :dimensions
      t.string :model, null: false, limit: 255

      t.timestamps

      t.index :created_at
    end
  end
end
