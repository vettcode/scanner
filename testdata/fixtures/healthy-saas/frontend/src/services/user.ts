// User service — handles authentication and profile
// Known complexity: authenticateUser=7, updateProfile=3, getUserRole=4

interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  active: boolean;
}

interface AuthResult {
  authenticated: boolean;
  token?: string;
  error?: string;
}

export function authenticateUser(email: string, password: string, mfaCode?: string): AuthResult {
  // complexity: 1 (base) + 6 decision points = 7 (4 ifs + || + &&)
  if (!email || !password) {
    return { authenticated: false, error: "Missing credentials" };
  }

  const user = findUserByEmail(email);
  if (!user) {
    return { authenticated: false, error: "User not found" };
  }

  if (!user.active) {
    return { authenticated: false, error: "Account disabled" };
  }

  if (user.role === "admin" && !mfaCode) {
    return { authenticated: false, error: "MFA required for admin" };
  }

  return { authenticated: true, token: `jwt_${user.id}_${Date.now()}` };
}

export function updateProfile(user: User, updates: Partial<User>): User {
  // complexity: 1 (base) + 2 decision points = 3
  if (!user.active) {
    throw new Error("Cannot update inactive user");
  }

  const updated = { ...user, ...updates };
  if (updates.email && updates.email !== user.email) {
    // Send verification email for email changes
    sendVerification(updates.email);
  }

  return updated;
}

export function getUserRole(user: User): string {
  // complexity: 1 (base) + 3 decision points = 4 (2 ifs + ||)
  if (!user.active) {
    return "disabled";
  }
  if (user.role === "admin" || user.role === "superadmin") {
    return "privileged";
  }
  return "standard";
}

function findUserByEmail(_email: string): User | null {
  return null;
}

function sendVerification(_email: string): void {
  // stub
}
