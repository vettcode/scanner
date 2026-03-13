require 'rails_helper'

RSpec.describe ApiController, type: :controller do
  describe '#process_request' do
    it 'returns error for missing action' do
      result = subject.process_request({})
      expect(result[:error]).to eq("Missing action")
    end

    it 'creates a resource' do
      result = subject.process_request(action: "create", data: { name: "test" })
      expect(result[:created]).to be true
    end
  end
end
