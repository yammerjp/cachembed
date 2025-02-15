#!/bin/bash

OPENAI_API_KEY={{op://vbudy2dovxcqmtirklrjhrtboe/f2jcrvep3par3c6pgmb6qxwt6e/OPENAI_API_KEY}}

#curl https://api.openai.com/v1/embeddings \
curl http://localhost:3000/v1/embeddings \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "The food was delicious and the waiter...",
    "model": "text-embedding-ada-002",
    "encoding_format": "base64"
  }'
