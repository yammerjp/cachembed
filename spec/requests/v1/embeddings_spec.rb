require 'rails_helper'
require 'webmock/rspec'

RSpec.describe "V1::Embeddings", type: :request do
  describe "POST /create" do
    context "input is string" do
      context 'without cache' do
        # mock upstream http request
        # validate upstream http request
        before do
          build_stub_request(
            model: "text-embedding-ada-002",
            input: [ "Hello, world!" ],
            base64s: [ "AAAAPgAAgD4AAAA/" ],
          )
        end

        it "returns a 200 status code" do
          post v1_embeddings_path, headers: {
            "Authorization" => "Bearer sk-abc123",
            "Content-Type" => "application/json",
            "Accept" => "application/json"
          }, params: {
            embedding: {
              model: "text-embedding-ada-002",
              input: "Hello, world!"
            }
          }.to_json

          expect(response).to be_successful
          expect(JSON.parse(response.body)).to eq({
            "object" => "list",
            "data" => [
              { "object" => "embedding", "embedding" => [ 0.125, 0.25, 0.5 ], "index" => 0 }
            ],
            "model" => "text-embedding-ada-002",
            "usage" => { "prompt_tokens" => 8, "total_tokens" => 8 }
          })
        end
      end

      context 'with cache' do
        before do
          EmbeddingModel.create!(name: "text-embedding-ada-002", default_dimensions: 1536)
          VectorCache.create!(input_hash: "943a702d06f34599aee1f8da8ef9f7296031d699", content: "AAAAPgAAgD4AAAA/", model: "text-embedding-ada-002", dimensions: 1536)
        end

        it "returns a 200 status code" do
          post v1_embeddings_path, headers: {
            "Authorization" => "Bearer sk-abc123",
            "Content-Type" => "application/json",
            "Accept" => "application/json"
          }, params: {
            embedding: {
              model: "text-embedding-ada-002",
              input: "Hello, world!"
            }
          }.to_json

          expect(response).to be_successful
          expect(JSON.parse(response.body)).to eq({
            "object" => "list",
            "data" => [
              { "object" => "embedding", "embedding" => [ 12.078431129455566, 12.087722778320312, 11.26669979095459, 1.757643058875047e-10 ], "index" => 0 }
            ],
            "model" => "text-embedding-ada-002",
            "usage" => { "prompt_tokens" => 0, "total_tokens" => 0 }
          })
        end
      end
    end

    context "input is string array" do
      before do
        build_stub_request(
          model: "text-embedding-ada-002",
          input: [ "Hello, world!", "Goodbye, world!" ],
          base64s: [ "AAAAPgAAgD4AAAA/", "AADAPgAAQD8AAGA/" ],
        )
      end

      it "returns a 200 status code" do
        post v1_embeddings_path, headers: {
          "Authorization" => "Bearer sk-abc123",
          "Content-Type" => "application/json",
          "Accept" => "application/json"
        }, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: [ "Hello, world!", "Goodbye, world!" ]
          }
        }.to_json

        expect(response).to be_successful
        expect(JSON.parse(response.body)).to eq({
          "object" => "list",
          "data" => [
            { "embedding" => [ 0.125, 0.25, 0.5 ], "index" => 0, "object" => "embedding" },
            { "embedding" => [ 0.375, 0.75, 0.875 ], "index" => 1, "object" => "embedding" }
          ],
          "model" => "text-embedding-ada-002",
          "usage" => { "prompt_tokens" => 8, "total_tokens" => 8 }
        })
      end
    end

    context "input is integer array" do
      before do
        build_stub_request(
          model: "text-embedding-ada-002",
          input: [ [ 1, 2, 3 ] ],
          base64s: [ "AAAAPgAAgD4AAAA/" ],
        )
      end

      it "returns a 200 status code" do
        post v1_embeddings_path, headers: {
          "Authorization" => "Bearer sk-abc123",
          "Content-Type" => "application/json",
          "Accept" => "application/json"
        }, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: [ 1, 2, 3 ]
          }
        }.to_json

        expect(response).to be_successful
        expect(JSON.parse(response.body)).to eq({
          "object" => "list",
          "data" => [
            { "embedding" => [ 0.125, 0.25, 0.5 ], "index" => 0, "object" => "embedding" }
          ],
          "model" => "text-embedding-ada-002",
          "usage" => { "prompt_tokens" => 8, "total_tokens" => 8 }
        })
      end
    end

    context "input is integer array array" do
      before do
        build_stub_request(
          model: "text-embedding-ada-002",
          input: [ [ 1, 2 ], [ 3, 4 ] ],
          base64s: [ "AAAAPgAAgD4AAAA/", "AADAPgAAQD8AAGA/" ],
        )
      end

      it "returns a 200 status code" do
        post v1_embeddings_path, headers: {
          "Authorization" => "Bearer sk-abc123",
          "Content-Type" => "application/json",
          "Accept" => "application/json"
        }, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: [
              [ 1, 2 ],
              [ 3, 4 ]
            ]
          }
        }.to_json

        expect(response).to be_successful
        expect(JSON.parse(response.body)).to eq({
          "object" => "list",
          "data" => [
            { "embedding" => [ 0.125, 0.25, 0.5 ], "index" => 0, "object" => "embedding" },
            { "embedding" => [ 0.375, 0.75, 0.875 ], "index" => 1, "object" => "embedding" }
          ],
          "model" => "text-embedding-ada-002",
          "usage" => { "prompt_tokens" => 8, "total_tokens" => 8 }
        })
      end
    end
  end

  def build_stub_request(model:, input:, base64s:)
    upstream_response = {
      data: base64s.map.with_index do |base64, index|
        {
          embedding: base64,
          index: index,
          object: "embedding"
        }
      end,
      model: model,
      object: "list",
      usage: { prompt_tokens: 8, total_tokens: 8 }
    }.to_json

    stub_request(:post, "https://api.openai.com/v1/embeddings")
      .with(
        headers: {
          "Accept" => "*/*",
          "Accept-Encoding" => "gzip;q=1.0,deflate;q=0.6,identity;q=0.3",
          "Authorization" => "Bearer sk-abc123",
          "Content-Type" => "application/json",
          "User-Agent" => "Faraday v2.12.2"
        },
        body: {
          model: model,
          input: input,
          encoding_format: "base64"
        }.to_json
      )
      .to_return(
        status: 200,
        headers: { "Content-Type" => "application/json" },
        body: upstream_response
      )
  end
end
