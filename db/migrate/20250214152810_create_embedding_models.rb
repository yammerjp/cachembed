class CreateEmbeddingModels < ActiveRecord::Migration[8.0]
  def change
    create_table :embedding_models do |t|
      t.string :name, null: false, limit: 256
      t.integer :default_dimensions, null: false

      t.timestamps

      t.index :name, unique: true
    end
  end
end
