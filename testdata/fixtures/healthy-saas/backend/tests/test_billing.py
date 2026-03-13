"""Tests for billing routes."""

import pytest


def test_create_subscription_valid():
    from app.routes.billing import create_subscription
    sub = create_subscription("usr_1", "pro", "card_123")
    assert sub["plan"] == "pro"
    assert sub["status"] == "active"


def test_create_subscription_invalid_plan():
    from app.routes.billing import create_subscription, BillingError
    with pytest.raises(BillingError):
        create_subscription("usr_1", "invalid", "card_123")
