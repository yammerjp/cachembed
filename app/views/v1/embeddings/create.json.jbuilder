json.object "list"
json.data @embeddings

json.model @model

json.usage do
  json.prompt_tokens @prompt_tokens
  json.total_tokens @total_tokens
end
