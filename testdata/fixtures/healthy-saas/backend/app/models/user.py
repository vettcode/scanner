# User model
# Known complexity: each method is 1-2

class User:
    def __init__(self, email: str, name: str, role: str = "user"):
        self.email = email
        self.name = name
        self.role = role
        self.active = True

    def is_admin(self) -> bool:
        return self.role in ("admin", "superadmin")

    def deactivate(self):
        self.active = False

    def to_dict(self) -> dict:
        return {
            "email": self.email,
            "name": self.name,
            "role": self.role,
            "active": self.active,
        }
