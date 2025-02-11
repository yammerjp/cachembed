require 'rails_helper'

RSpec.describe Input do
  describe 'validations' do
    it 'requires input to be present' do
      input = Input.new(nil)
      expect(input).not_to be_valid
      expect(input.errors[:input]).to include("can't be blank")
    end
  end

  describe '.input_format' do
    context 'when input is a string' do
      it 'returns :string' do
        expect(Input.input_format('hello')).to eq(:string)
      end
    end

    context 'when input is an array of integers' do
      it 'returns :tokens' do
        expect(Input.input_format([1, 2, 3])).to eq(:tokens)
      end
    end

    context 'when input is an array of strings' do
      it 'returns :strings' do
        expect(Input.input_format(['hello', 'world'])).to eq(:strings)
      end
    end

    context 'when input is an array of integer arrays' do
      it 'returns :tokens' do
        expect(Input.input_format([[1, 2], [3, 4]])).to eq(:tokens)
      end
    end

    context 'when input is invalid' do
      it 'returns nil for mixed array' do
        expect(Input.input_format([1, 'string'])).to be_nil
      end

      it 'returns nil for hash' do
        expect(Input.input_format({ key: 'value' })).to be_nil
      end
    end
  end

  describe '#input_format' do
    it 'delegates to the class method' do
      input = Input.new('test')
      expect(input.input_format).to eq(:string)
    end
  end

  describe '#to_hash' do
    it 'returns the original input' do
      original_input = [[1, 2], [3, 4]]
      input = Input.new(original_input)
      expect(input.to_hash).to eq(original_input)
    end
  end
end 
