# This file is auto-generated from the current state of the database. Instead
# of editing this file, please use the migrations feature of Active Record to
# incrementally modify your database, and then regenerate this schema definition.
#
# This file is the source Rails uses to define your schema when running `bin/rails
# db:schema:load`. When creating a new database, `bin/rails db:schema:load` tends to
# be faster and is potentially less error prone than running all of your
# migrations from scratch. Old migrations may fail to apply correctly if those
# migrations use external dependencies or application code.
#
# It's strongly recommended that you check this file into your version control system.

ActiveRecord::Schema[8.0].define(version: 2025_02_16_082236) do
  create_table "embedding_models", force: :cascade do |t|
    t.string "name", limit: 256, null: false
    t.integer "default_dimensions", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["name"], name: "index_embedding_models_on_name", unique: true
  end

  create_table "embedding_requests", force: :cascade do |t|
    t.string "input_hash", limit: 40, null: false
    t.integer "input_length", null: false
    t.integer "dimensions"
    t.string "model", limit: 255, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["created_at"], name: "index_embedding_requests_on_created_at"
  end

  create_table "vector_caches", force: :cascade do |t|
    t.string "input_hash", limit: 40, null: false
    t.string "model", limit: 128, null: false
    t.integer "dimensions", default: 0, null: false
    t.binary "content", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["input_hash", "model", "dimensions"], name: "index_vector_caches_on_input_hash_and_model_and_dimensions", unique: true
  end
end
