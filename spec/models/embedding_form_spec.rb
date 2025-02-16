require 'rails_helper'

RSpec.describe EmbeddingForm do
  let(:valid_attributes) do
    {
      model: "text-embedding-ada-002",
      api_key: "sk-validapikey123",
      input: ["テストテキスト"]
    }
  end

  describe 'バリデーション' do
    it '有効な属性の場合は有効であること' do
      form = EmbeddingForm.new(valid_attributes)
      expect(form).to be_valid
    end

    it 'modelがない場合は無効であること' do
      form = EmbeddingForm.new(valid_attributes.merge(model: nil))
      form.valid?
      expect(form.errors[:model]).to include("can't be blank")
    end

    it '許可されていないmodelの場合は無効であること' do
      form = EmbeddingForm.new(valid_attributes.merge(model: "invalid-model"))
      form.valid?
      expect(form.errors[:model]).to include("is not included in the list")
    end

    it 'api_keyがない場合は無効であること' do
      form = EmbeddingForm.new(valid_attributes.merge(api_key: nil))
      form.valid?
      expect(form.errors[:api_key]).to include("can't be blank")
    end

    it '不正なapi_keyの場合は無効であること' do
      form = EmbeddingForm.new(valid_attributes.merge(api_key: "invalid-key"))
      form.valid?
      expect(form.errors[:api_key]).to include("is invalid")
    end

    it 'inputが空配列の場合はtargetsが1つの要素を持つこと' do
      form = EmbeddingForm.new(valid_attributes.merge(input: []))
      expect(form.targets.size).to eq(1)
    end

    context 'dimensionsのバリデーション' do
      it '1以下の場合は無効であること' do
        form = EmbeddingForm.new(valid_attributes.merge(dimensions: 1))
        form.valid?
        expect(form.errors[:dimensions]).to include("must be greater than 1")
      end

      it '10000以上の場合は無効であること' do
        form = EmbeddingForm.new(valid_attributes.merge(dimensions: 10000))
        form.valid?
        expect(form.errors[:dimensions]).to include("must be less than 10000")
      end

      it 'nilの場合は有効であること' do
        form = EmbeddingForm.new(valid_attributes.merge(dimensions: nil))
        expect(form).to be_valid
      end
    end

    context 'encoding_formatのバリデーション' do
      it '許可されていない形式の場合は無効であること' do
        form = EmbeddingForm.new(valid_attributes.merge(encoding_format: 'invalid'))
        form.valid?
        expect(form.errors[:encoding_format]).to include("is not included in the list")
      end

      it 'nilの場合は有効であること' do
        form = EmbeddingForm.new(valid_attributes.merge(encoding_format: nil))
        expect(form).to be_valid
      end
    end
  end

  describe '#save!' do
    before do
      stub_request(:post, "https://api.openai.com/v1/embeddings")
        .with(
          body: {
            model: "text-embedding-ada-002",
            input: ["テストテキスト"],
            encoding_format: "base64"
          }.to_json,
          headers: {
            'Authorization' => 'Bearer sk-validapikey123',
            'Content-Type' => 'application/json'
          }
        )
        .to_return(
          status: 200,
          body: {
            object: "list",
            data: [
              {
                object: "embedding",
                embedding: "AAAAPgAAgD4AAAA/",
                index: 0
              }
            ],
            model: "text-embedding-ada-002",
            usage: {
              prompt_tokens: 5,
              total_tokens: 5
            }
          }.to_json,
          headers: { 'Content-Type' => 'application/json' }
        )
    end

    it '有効な属性で実行した場合、埋め込みベクトルの配列を返すこと' do
      form = EmbeddingForm.new(valid_attributes)
      result = form.save!
      
      expect(result).to be_an(Array)
      expect(result.first).to include(
        object: "embedding",
        index: 0,
        embedding: [0.125, 0.25, 0.5]
      )
    end
    
    it 'EmbeddingRequestが1つ作成されること' do
      form = EmbeddingForm.new(valid_attributes)
      expect{form.save!}.to change(EmbeddingRequest, :count).by(1)
    end

    it 'EmbeddingRequestのinput_hashがハッシュ化されていること' do
      form = EmbeddingForm.new(valid_attributes)
      form.save!
      expect(EmbeddingRequest.first.input_hash).to eq(Digest::SHA1.hexdigest("テストテキスト"))
    end
    
    it 'EmbeddingRequestのinput_lengthが正しいこと' do
      form = EmbeddingForm.new(valid_attributes)
      form.save!
      expect(EmbeddingRequest.first.input_length).to eq(21)
    end
  end
end
