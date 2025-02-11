json.object @embedding[:object]

json.data @embedding[:data] do |data|
  json.object data[:object]
  json.embedding data[:embedding]
  json.index data[:index]
end

json.model @embedding[:model]

json.usage do
  json.prompt_tokens @embedding[:usage][:prompt_tokens]
  json.total_tokens @embedding[:usage][:total_tokens]
end 
