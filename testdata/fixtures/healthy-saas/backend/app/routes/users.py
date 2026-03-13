# User routes — handles CRUD operations
# Known complexity: create_user=5, get_user=2, update_user=4, list_users=3

from typing import Optional


class UserNotFoundError(Exception):
    pass


class ValidationError(Exception):
    pass


def create_user(email: str, name: str, role: str = "user") -> dict:
    """Create a new user with validation.

    Complexity: 1 (base) + 4 decision points = 5
    """
    if not email or "@" not in email:
        raise ValidationError("Invalid email")

    if not name or len(name) < 2:
        raise ValidationError("Name too short")

    if role not in ("user", "admin", "moderator"):
        raise ValidationError("Invalid role")

    existing = find_by_email(email)
    if existing:
        raise ValidationError("Email already registered")

    return {
        "id": generate_id(),
        "email": email,
        "name": name,
        "role": role,
        "active": True,
    }


def get_user(user_id: str) -> dict:
    """Get user by ID.

    Complexity: 1 (base) + 1 decision point = 2
    """
    user = find_by_id(user_id)
    if not user:
        raise UserNotFoundError(f"User {user_id} not found")
    return user


def update_user(user_id: str, updates: dict) -> dict:
    """Update user fields with validation.

    Complexity: 1 (base) + 3 decision points = 4
    """
    user = get_user(user_id)

    if "email" in updates and "@" not in updates["email"]:
        raise ValidationError("Invalid email")

    if "role" in updates and updates["role"] not in ("user", "admin", "moderator"):
        raise ValidationError("Invalid role")

    for key, value in updates.items():
        if key in ("email", "name", "role"):
            user[key] = value

    return user


def list_users(
    role: Optional[str] = None,
    active_only: bool = True,
    limit: int = 50,
) -> list:
    """List users with optional filtering.

    Complexity: 1 (base) + 2 decision points = 3
    """
    users = get_all_users()

    if role:
        users = [u for u in users if u["role"] == role]

    if active_only:
        users = [u for u in users if u["active"]]

    return users[:limit]


# Stubs
def find_by_email(_email: str):
    return None


def find_by_id(_user_id: str):
    return None


def get_all_users():
    return []


def generate_id():
    return "usr_12345"
