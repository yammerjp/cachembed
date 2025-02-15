require 'rails_helper'

RSpec.describe UpstreamClient do
  let(:api_key) { "test_api_key" }
  let(:model) { "text-embedding-3-small" }
  let(:dimensions) { 1536 }
  let(:targets) { [ double("Target", to_hash: { text: "Hello world" }) ] }
  let(:client) { described_class.new(api_key: api_key, model: model, dimensions: dimensions, targets: targets) }

  describe '#initialize' do
    it '正しくインスタンス変数が設定されること' do
      expect(client.api_key).to eq(api_key)
      expect(client.instance_variable_get(:@model)).to eq(model)
      expect(client.instance_variable_get(:@dimensions)).to eq(dimensions)
      expect(client.instance_variable_get(:@targets)).to eq(targets)
    end
  end

  describe '#request_body' do
    it '正しいリクエストボディを生成すること' do
      expected_body = {
        model: model,
        input: [ { text: "Hello world" } ],
        encoding_format: "base64",
        dimensions: dimensions
      }
      expect(client.request_body).to eq(expected_body)
    end

    context 'dimensionsが設定されていない場合' do
      let(:dimensions) { nil }

      it 'dimensionsを含まないリクエストボディを生成すること' do
        expected_body = {
          model: model,
          input: [ { text: "Hello world" } ],
          encoding_format: "base64"
        }
        expect(client.request_body).to eq(expected_body)
      end
    end
  end

  describe '#post' do
    let(:float_array) { [ 0.1, 0.2, 0.3 ] }
    let(:mock_response) do
      {
        data: [
          {
            embedding: Base64.strict_encode64(float_array.pack('f*')),
            index: 0
          }
        ],
        model: model,
        usage: {
          prompt_tokens: 10,
          total_tokens: 10
        }
      }
    end

    before do
      stub_request(:post, UpstreamClient::URL)
        .with(
          headers: {
            'Authorization' => "Bearer #{api_key}",
            'Content-Type' => 'application/json'
          },
          body: client.request_body
        )
        .to_return(
          status: 200,
          body: mock_response.to_json,
          headers: { 'Content-Type' => 'application/json' }
        )
    end

    it '正常なレスポンスを処理できること' do
      response = client.post
      expect(response).to be_a(UpstreamResponse)
      expect(response.body).to eq(mock_response)
      expect(response.targets).to eq(targets)
      expect(response.model).to eq(model)
      expect(response.dimensions).to eq(float_array.size)
    end

    context 'エラーレスポンスの場合' do
      before do
        stub_request(:post, UpstreamClient::URL)
          .to_return(
            status: 400,
            body: { error: "Invalid request" }.to_json
          )
      end

      it 'エラーを発生させること' do
        expect { client.post }.to raise_error(/Failed to get embedding from upstream/)
      end
    end
  end
end
