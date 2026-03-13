# API controller — contains planted secrets for testing
# Known complexity: process_request=6, handle_webhook=5
# Known secrets: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, STRIPE_SECRET_KEY

class ApiController < ApplicationController
  # PLANTED SECRET: AWS credentials (for scanner detection testing)
  AWS_ACCESS_KEY_ID = "AKIAIOSFODNN7TESTAB"
  AWS_SECRET_ACCESS_KEY = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYTESTKEYVAL"

  # PLANTED SECRET: Stripe key
  STRIPE_KEY = "sk_live_51H7example1234567890abcdefghijklmnop"

  # complexity: 1 (base) + 5 decision points = 6
  def process_request(params)
    unless params[:action]
      return { error: "Missing action" }
    end

    case params[:action]
    when "create"
      if params[:data].nil?
        return { error: "Missing data" }
      end
      create_resource(params[:data])
    when "update"
      if params[:id].nil?
        return { error: "Missing ID" }
      end
      update_resource(params[:id], params[:data])
    when "delete"
      delete_resource(params[:id])
    else
      { error: "Unknown action" }
    end
  end

  # complexity: 1 (base) + 4 decision points = 5
  def handle_webhook(payload)
    return { error: "Empty payload" } if payload.nil?

    event_type = payload[:type]

    if event_type == "payment.succeeded"
      handle_payment_success(payload[:data])
    elsif event_type == "payment.failed"
      handle_payment_failure(payload[:data])
    elsif event_type == "refund.created"
      handle_refund(payload[:data])
    else
      { status: "ignored", type: event_type }
    end
  end

  private

  def create_resource(data) = { created: true, data: data }
  def update_resource(id, data) = { updated: true, id: id }
  def delete_resource(id) = { deleted: true, id: id }
  def handle_payment_success(data) = { payment: "success" }
  def handle_payment_failure(data) = { payment: "failed" }
  def handle_refund(data) = { refund: "processed" }
end
