require 'rails_helper'

RSpec.describe EmbeddingTarget do
  describe '#initialize' do
    it '文字列で初期化できること' do
      target = described_class.new('テストテキスト')
      expect(target.to_hash).to eq('テストテキスト')
    end

    it '整数配列で初期化できること' do
      target = described_class.new([1, 2, 3])
      expect(target.to_hash).to eq([1, 2, 3])
    end
  end

  describe '.build_targets!' do
    context '文字列が入力された場合' do
      it '単一のEmbeddingTargetインスタンスを含む配列を返すこと' do
        result = described_class.build_targets!('テストテキスト')
        expect(result.size).to eq(1)
        expect(result.first).to be_an(EmbeddingTarget)
        expect(result.first.to_hash).to eq('テストテキスト')
      end
    end

    context '整数配列が入力された場合' do
      it '単一のEmbeddingTargetインスタンスを含む配列を返すこと' do
        result = described_class.build_targets!([1, 2, 3])
        expect(result.size).to eq(1)
        expect(result.first).to be_an(EmbeddingTarget)
        expect(result.first.to_hash).to eq([1, 2, 3])
      end
    end

    context '文字列配列が入力された場合' do
      it '複数のEmbeddingTargetインスタンスを含む配列を返すこと' do
        result = described_class.build_targets!(['テスト1', 'テスト2'])
        expect(result.size).to eq(2)
        expect(result.all? { |r| r.is_a?(EmbeddingTarget) }).to be true
        expect(result.map(&:to_hash)).to eq(['テスト1', 'テスト2'])
      end
    end

    context '整数配列の配列が入力された場合' do
      it '複数のEmbeddingTargetインスタンスを含む配列を返すこと' do
        result = described_class.build_targets!([[1, 2], [3, 4]])
        expect(result.size).to eq(2)
        expect(result.all? { |r| r.is_a?(EmbeddingTarget) }).to be true
        expect(result.map(&:to_hash)).to eq([[1, 2], [3, 4]])
      end
    end

    context '無効な入力形式の場合' do
      it 'エラーを発生させること' do
        expect {
          described_class.build_targets!({ invalid: 'format' })
        }.to raise_error(RuntimeError, /Invalid input format/)
      end
    end
  end

  describe '#sha1sum' do
    it '文字列入力の場合、正しいハッシュを生成すること' do
      target = described_class.new('テストテキスト')
      expect(target.sha1sum).to eq('8e7c7515d882df9db2c4d807fa4b34a2e57b50bc')
    end

    it 'トークン配列の場合、カンマ区切りの文字列からハッシュを生成すること' do
      target = described_class.new([1, 2, 3])
      expected_hash = Digest::SHA1.hexdigest('1,2,3')
      expect(target.sha1sum).to eq(expected_hash)
    end
  end
end 