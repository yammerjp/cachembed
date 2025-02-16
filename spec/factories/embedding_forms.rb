FactoryBot.define do
  factory :embedding_form do
    sequence(:name) { |n| "埋め込みフォーム#{n}" }
    description { "テスト用の埋め込みフォームです。" }
  end
end 