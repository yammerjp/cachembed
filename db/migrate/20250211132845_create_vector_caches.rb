class CreateVectorCaches < ActiveRecord::Migration[8.0]
  def change
    create_table :vector_caches do |t|
      t.string :input_hash, null: false, limit: 40
      t.string :model, null: false, limit: 128
      t.integer :dimensions, null: false, default: 0, comment: "0 means default dimension"
      t.binary :content, null: false

      t.timestamps
      t.index [ :input_hash, :model, :dimensions ], unique: true
    end
  end
end
