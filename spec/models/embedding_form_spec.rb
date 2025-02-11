require 'rails_helper'

RSpec.describe EmbeddingForm do
  describe 'validations' do
    let(:valid_attributes) do
      {
        model: 'text-embedding-ada-002',
        dimensions: 1536,
        input: 'Hello, world!',
        encoding_format: 'float'
      }
    end

    subject(:form) { described_class.new(valid_attributes) }

    it 'is valid with valid attributes' do
      expect(form).to be_valid
    end

    describe '#model' do
      it 'requires model to be present' do
        form.model = nil
        expect(form).not_to be_valid
        expect(form.errors[:model]).to include("can't be blank")
      end

      it 'requires model to be in MODEL_NAMES' do
        form.model = 'invalid-model'
        expect(form).not_to be_valid
        expect(form.errors[:model]).to include('is not included in the list')
      end
    end

    describe '#dimensions' do
      it 'validates dimensions is a positive integer' do
        form.dimensions = 0
        expect(form).not_to be_valid
        expect(form.errors[:dimensions]).to include('must be greater than 0')
      end

      it 'validates dimensions is less than 10000' do
        form.dimensions = 10000
        expect(form).not_to be_valid
        expect(form.errors[:dimensions]).to include('must be less than 10000')
      end

      it 'is optional' do
        form.dimensions = nil
        expect(form).to be_valid
      end
    end

    describe '#encoding_format' do
      it 'validates encoding_format is in ENCODING_FORMATS' do
        form.encoding_format = 'invalid'
        expect(form).not_to be_valid
        expect(form.errors[:encoding_format]).to include('is not included in the list')
      end

      it 'is optional' do
        form.encoding_format = nil
        expect(form).to be_valid
      end
    end

    describe '#input' do
      it 'validates input presence' do
        form.input = nil
        expect(form).not_to be_valid
        expect(form.errors[:input]).to include("can't be blank")
      end

      it 'validates input format for string' do
        expect(form).to be_valid
      end

      it 'validates input format for array of integers' do
        form.input = Input.new([1, 2, 3])
        expect(form).to be_valid
      end

      it 'validates input format for array of strings' do
        form.input = Input.new(['hello', 'world'])
        expect(form).to be_valid
      end

      it 'validates input format for array of integer arrays' do
        form.input = Input.new([[1, 2], [3, 4]])
        expect(form).to be_valid
      end

      it 'is invalid with invalid input format' do
        form.input = Input.new([1, 'invalid'])
        expect(form).not_to be_valid
        expect(form.errors[:input]).to include('has invalid format')
      end
    end
  end

  describe '#initialize' do
    it 'sets attributes from hash' do
      form = described_class.new(
        model: 'text-embedding-ada-002',
        dimensions: 1536,
        input: 'Hello, world!',
        encoding_format: 'float',
        api_key: 'test-key'
      )

      expect(form.model).to eq('text-embedding-ada-002')
      expect(form.dimensions).to eq(1536)
      expect(form.input).to be_a(Input)
      expect(form.input.to_hash).to eq('Hello, world!')
      expect(form.encoding_format).to eq('float')
      expect(form.api_key).to eq('test-key')
    end
  end

  describe '#to_hash' do
    it 'returns hash with required attributes' do
      form = described_class.new(
        model: 'text-embedding-ada-002',
        input: 'Hello, world!'
      )

      expect(form.to_hash).to eq({
        model: 'text-embedding-ada-002',
        input: 'Hello, world!'
      })
    end

    it 'includes optional attributes when present' do
      form = described_class.new(
        model: 'text-embedding-ada-002',
        input: 'Hello, world!',
        dimensions: 1536,
        encoding_format: 'float'
      )

      expect(form.to_hash).to eq({
        model: 'text-embedding-ada-002',
        input: 'Hello, world!',
        dimensions: 1536,
        encoding_format: 'float'
      })
    end
  end

  describe '#do_embedding' do
    let(:form) do
      described_class.new(
        model: 'text-embedding-ada-002',
        input: 'Hello, world!',
        api_key: 'test-key'
      )
    end

    it 'delegates to UpstreamClient' do
      client = instance_double(UpstreamClient)
      expect(UpstreamClient).to receive(:new)
        .with(api_key: 'test-key', embedding_form: form)
        .and_return(client)
      expect(client).to receive(:post)

      form.do_embedding
    end
  end
end 
