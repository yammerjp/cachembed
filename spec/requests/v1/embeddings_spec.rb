require 'rails_helper'
require 'webmock/rspec'

RSpec.describe "V1::Embeddings", type: :request do
  describe "POST /create" do
    context "input is string" do
      # mock upstream http request
      # validate upstream http request
      before do
        build_stub_request(
          model: "text-embedding-ada-002",
          input: "Hello, world!",
          vectors: [0.0023064255, -0.009327292, 0.015954146],
        )
      end

      it "returns a 200 status code" do
        post v1_embeddings_path, headers: {
          "Authorization" => "Bearer sk-abc123",
          "Content-Type" => "application/json"
        }, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: "Hello, world!",
          }
        }.to_json
      end
    end

    context "input is string array" do
      before do
        build_stub_request(
          model: "text-embedding-ada-002",
          input: ["Hello, world!", "Goodbye, world!"],
          vectors: [
            [0.0023064255, -0.009327292, 0.015954146],
            [0.0023064255, -0.009327292, 0.015954146],
          ]
        )
      end
      
      it "returns a 200 status code" do
        post v1_embeddings_path, headers: {
          "Authorization" => "Bearer sk-abc123",
          "Content-Type" => "application/json"
        }, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: ["Hello, world!", "Goodbye, world!"],
          }
        }.to_json
      end 
    end

    context "input is integer array" do
      before do
        build_stub_request(
          model: "text-embedding-ada-002",
          input: [1, 2, 3],
          vectors: [
            [0.0023064255, -0.009327292, 0.015954146],
          ]
        )
      end
      
      it "returns a 200 status code" do
        post v1_embeddings_path, headers: {
          "Authorization" => "Bearer sk-abc123",
          "Content-Type" => "application/json"
        }, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: [1, 2, 3],
          }
        }.to_json
      end
    end

    context "input is integer array array" do
      before do
        build_stub_request(
          model: "text-embedding-ada-002",
          input: [[1, 2], [3, 4]],
          vectors: [
            [0.0023064255, -0.009327292, 0.015954146],
            [0.0023064255, -0.009327292, 0.015954146],
          ]
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
              [1, 2],
              [3, 4]
            ],
          } 
        }.to_json
      end
    end
  end

  def build_stub_request(model:, input:, vectors:)
    stub_request(:post, "https://api.openai.com/v1/embeddings")
      .with(
        headers: {
          "Accept" => "*/*",
          "Accept-Encoding" => "gzip;q=1.0,deflate;q=0.6,identity;q=0.3",
          "Authorization" => "Bearer sk-abc123",
          "Content-Type" => "application/json",
          "User-Agent" => "Ruby"
        },
        body: {
          model: model,
          input: input,
        }.to_json
      )
      .to_return(status: 200, body: {
        object: "list",
        data: vectors.map.with_index { |vector, index| {
          object: "embedding",
          embedding: vector,
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
