# Billing routes — handles subscription management
# Known complexity: create_subscription=6, cancel_subscription=3, process_webhook=8

from typing import Optional
from datetime import datetime


class BillingError(Exception):
    pass


def create_subscription(user_id: str, plan: str, payment_method: str) -> dict:
    """Create a new subscription.

    Complexity: 1 (base) + 5 decision points = 6
    """
    if plan not in ("starter", "pro", "enterprise"):
        raise BillingError("Invalid plan")

    if not payment_method:
        raise BillingError("Payment method required")

    existing = get_active_subscription(user_id)
    if existing:
        if existing["plan"] == plan:
            raise BillingError("Already subscribed to this plan")
        return upgrade_subscription(existing, plan)

    price = {"starter": 29, "pro": 99, "enterprise": 299}.get(plan, 0)

    return {
        "id": f"sub_{user_id}",
        "user_id": user_id,
        "plan": plan,
        "price": price,
        "status": "active",
        "created_at": datetime.utcnow().isoformat(),
    }


def cancel_subscription(user_id: str, reason: Optional[str] = None) -> dict:
    """Cancel an active subscription.

    Complexity: 1 (base) + 2 decision points = 3
    """
    sub = get_active_subscription(user_id)
    if not sub:
        raise BillingError("No active subscription")

    sub["status"] = "cancelled"
    sub["cancelled_at"] = datetime.utcnow().isoformat()
    if reason:
        sub["cancel_reason"] = reason

    return sub


def process_webhook(event_type: str, payload: dict) -> dict:
    """Process a payment provider webhook.

    Complexity: 1 (base) + 7 decision points = 8
    """
    if not event_type or not payload:
        raise BillingError("Invalid webhook")

    if event_type == "payment.succeeded":
        sub_id = payload.get("subscription_id")
        if not sub_id:
            raise BillingError("Missing subscription_id")
        return {"action": "confirm", "subscription_id": sub_id}

    elif event_type == "payment.failed":
        sub_id = payload.get("subscription_id")
        attempt = payload.get("attempt", 1)
        if attempt >= 3:
            return {"action": "cancel", "subscription_id": sub_id}
        return {"action": "retry", "subscription_id": sub_id, "attempt": attempt}

    elif event_type == "subscription.cancelled":
        return {"action": "deactivate", "subscription_id": payload.get("subscription_id")}

    elif event_type == "refund.processed":
        return {"action": "refund", "amount": payload.get("amount", 0)}

    return {"action": "ignore", "event_type": event_type}


# Stubs
def get_active_subscription(_user_id: str):
    return None


def upgrade_subscription(existing: dict, plan: str):
    existing["plan"] = plan
    return existing
