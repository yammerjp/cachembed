class UpstreamResponse
  attr_reader :data, :targets, :model, :dimensions

  def initialize(response_body, targets)
    @data = response_body[:data] || []
    @targets = targets
    @model = response_body[:model]
    @dimensions = response_body[:dimensions]
  end

  # targets の順番に対応した sha1sum と embedding のペアを返す
  def vectors
    targets.zip(data).map do |target, item|
      [target.sha1sum, item[:embedding]]
    end
  end
end
