require 'rails_helper'

RSpec.describe UpstreamResponse do
  describe '#vector_cache_hashes' do
    let(:model) { 'text-embedding-ada-002' }
    let(:dimensions) { nil }
    let(:target) { EmbeddingTarget.new('Hello, world!') }
    let(:body) do
      {
        object: 'list',
        data: [
          {
            object: 'embedding',
            embedding: 'AAAAPgAAgD4AAAA/',
            index: 0
          }
        ],
        model: model,
        usage: {
          prompt_tokens: 8,
          total_tokens: 8
        }
      }
    end

    subject(:response) { described_class.new(body: body, targets: [ target ], model: model) }

    it 'returns array of hash with input_hash, content, model' do
      result = response.vector_cache_hashes
      expect(result).to be_an(Array)
      expect(result.size).to eq(1)

      hash = result.first
      expect(hash).to include(
        input_hash: target.sha1sum,
        model: model,
        dimensions: 3,
      )
      expect(hash[:content]).to be_a(String)
    end

    context 'when multiple targets are given' do
      let(:target2) { EmbeddingTarget.new('Another text') }
      let(:body) do
        {
          object: 'list',
          data: [
            {
              object: 'embedding',
              embedding: 'AAAAPgAAgD4AAAA/',
              index: 0
            },
            {
              object: 'embedding',
              embedding: 'gD4AAAA/AAAAPg==',
              index: 1
            }
          ],
          model: model,
          usage: {
            prompt_tokens: 16,
            total_tokens: 16
          }
        }
      end

      subject(:response) { described_class.new(body: body, targets: [ target, target2 ], model: model) }

      it 'returns array of hashes for each target' do
        result = response.vector_cache_hashes
        expect(result.size).to eq(2)
        expect(result[0][:input_hash]).to eq(target.sha1sum)
        expect(result[1][:input_hash]).to eq(target2.sha1sum)
      end
    end
  end
end
