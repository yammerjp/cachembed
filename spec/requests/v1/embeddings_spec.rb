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
            "Content-Type" => "application/json"
          }, params: {
            embedding: {
              model: "text-embedding-ada-002",
              input: "Hello, world!"
            }
          }.to_json
        end
        # TODO: validate response structure
        # expect(response).to be_successful
      end

      context 'with cache' do
        before do
          EmbeddingModel.create!(name: "text-embedding-ada-002", default_dimensions: 1536)
          VectorCache.create!(input_hash: "943a702d06f34599aee1f8da8ef9f7296031d699", content: "AAAAPgAAgD4AAAA/", model: "text-embedding-ada-002", dimensions: 1536)
        end

        it "returns a 200 status code" do
          post v1_embeddings_path, headers: {
            "Authorization" => "Bearer sk-abc123",
            "Content-Type" => "application/json"
          }, params: {
            embedding: {
              model: "text-embedding-ada-002",
              input: "Hello, world!"
            }
          }.to_json
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
          "Content-Type" => "application/json"
        }, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: [ "Hello, world!", "Goodbye, world!" ]
          }
        }.to_json
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
          "Content-Type" => "application/json"
        }, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: [ 1, 2, 3 ]
          }
        }.to_json
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
          "Content-Type" => "application/json"
        }, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: [
              [ 1, 2 ],
              [ 3, 4 ]
            ]
          }
        }.to_json
      end
    end
  end

  def build_stub_request(model:, input:, base64s:)
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
      .to_return(status: 200, body: {
        object: "list",
        data: base64s.map.with_index { |base64, index| {
          object: "embedding",
          embedding: base64,
          index: index
        } },
        model: model,
        usage: {
          prompt_tokens: 8,
          total_tokens: 8
        }
      }.to_json)
  end
end
