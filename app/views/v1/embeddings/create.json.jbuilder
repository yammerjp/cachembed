json.object "list"

json.data @embeddings do |data, index|
  json.object "embedding"
  json.embedding data
  json.index index
end

json.model @model

json.usage do
  json.prompt_tokens @prompt_tokens
  json.total_tokens @total_tokens
end
