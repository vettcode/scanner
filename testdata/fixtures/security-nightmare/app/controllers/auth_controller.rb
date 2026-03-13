# Auth controller
# Known complexity: authenticate=5, authorize=4

class AuthController < ApplicationController
  # complexity: 1 (base) + 4 decision points = 5
  def authenticate(email, password)
    return { error: "Missing credentials" } if email.nil? || password.nil?

    user = User.find_by(email: email)
    return { error: "User not found" } unless user

    unless user.valid_password?(password)
      return { error: "Invalid password" }
    end

    if user.locked?
      return { error: "Account locked" }
    end

    { token: generate_token(user), user: user.as_json }
  end

  # complexity: 1 (base) + 3 decision points = 4
  def authorize(token, required_role)
    return false if token.nil?

    payload = decode_token(token)
    return false unless payload

    user_role = payload[:role]
    return true if user_role == "admin"

    user_role == required_role
  end

  private

  def generate_token(user) = "jwt_#{user.id}_#{Time.now.to_i}"
  def decode_token(token) = { role: "user" }
end
