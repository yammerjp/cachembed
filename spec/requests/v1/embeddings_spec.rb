require 'rails_helper'

RSpec.describe "V1::Embeddings", type: :request do
  describe "POST /create" do
    context "input is string" do
      it "returns a 200 status code" do
        post v1_embeddings_path, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: "Hello, world!",
          }
        }
      end
    end
    context "input is string array" do
      it "returns a 200 status code" do
        post v1_embeddings_path, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: ["Hello, world!"],
          }
        }
      end 
    end
    context "input is integer array" do
      it "returns a 200 status code" do
        post v1_embeddings_path, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: [1, 2, 3],
          }
        }
      end
    end
    context "input is integer array array" do
      it "returns a 200 status code" do
        post v1_embeddings_path, params: {
          embedding: {
            model: "text-embedding-ada-002",
            input: [[1, 2, 3]],
          } 
        }
      end
    end
  end
end
