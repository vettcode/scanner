"""Tests for user routes."""

import pytest


def test_create_user_valid():
    from app.routes.users import create_user
    user = create_user("test@example.com", "Test User")
    assert user["email"] == "test@example.com"
    assert user["role"] == "user"


def test_create_user_invalid_email():
    from app.routes.users import create_user, ValidationError
    with pytest.raises(ValidationError):
        create_user("invalid", "Test")


def test_list_users_default():
    from app.routes.users import list_users
    users = list_users()
    assert isinstance(users, list)
