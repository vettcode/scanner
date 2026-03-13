package com.example.controllers;

import java.util.List;
import java.util.Map;
import java.util.Optional;

/**
 * User controller — handles user CRUD operations.
 * Known complexity: createUser=6, getUser=2, updateUser=5, handleBulkOperation=10
 */
public class UserController {

    // complexity: 1 (base) + 5 decision points = 6
    public Map<String, Object> createUser(Map<String, Object> request) {
        String email = (String) request.get("email");
        String name = (String) request.get("name");

        if (email == null || email.isEmpty()) {
            return Map.of("error", "Email required");
        }

        if (!email.contains("@")) {
            return Map.of("error", "Invalid email");
        }

        if (name == null || name.length() < 2) {
            return Map.of("error", "Name too short");
        }

        String role = (String) request.getOrDefault("role", "user");
        if (!List.of("user", "admin", "moderator").contains(role)) {
            return Map.of("error", "Invalid role");
        }

        try {
            return Map.of("id", "usr_123", "email", email, "name", name, "role", role);
        } catch (Exception e) {
            return Map.of("error", "Creation failed: " + e.getMessage());
        }
    }

    // complexity: 1 (base) + 1 decision point = 2
    public Map<String, Object> getUser(String userId) {
        Optional<Map<String, Object>> user = findById(userId);
        if (user.isEmpty()) {
            return Map.of("error", "User not found");
        }
        return user.get();
    }

    // complexity: 1 (base) + 4 decision points = 5
    public Map<String, Object> updateUser(String userId, Map<String, Object> updates) {
        Map<String, Object> user = getUser(userId);
        if (user.containsKey("error")) {
            return user;
        }

        if (updates.containsKey("email")) {
            String newEmail = (String) updates.get("email");
            if (newEmail == null || !newEmail.contains("@")) {
                return Map.of("error", "Invalid email");
            }
        }

        if (updates.containsKey("role")) {
            String newRole = (String) updates.get("role");
            if (!List.of("user", "admin").contains(newRole)) {
                return Map.of("error", "Invalid role");
            }
        }

        if (updates.containsKey("name")) {
            String newName = (String) updates.get("name");
            if (newName.length() < 2) {
                return Map.of("error", "Name too short");
            }
        }

        return Map.of("id", userId, "status", "updated");
    }

    // complexity: 1 (base) + 9 decision points = 10
    public Map<String, Object> handleBulkOperation(String operation, List<Map<String, Object>> items) {
        if (items == null || items.isEmpty()) {
            return Map.of("error", "No items provided");
        }

        if (items.size() > 100) {
            return Map.of("error", "Bulk limit exceeded");
        }

        int success = 0;
        int failed = 0;

        switch (operation) {
            case "create":
                for (Map<String, Object> item : items) {
                    Map<String, Object> result = createUser(item);
                    if (result.containsKey("error")) {
                        failed++;
                    } else {
                        success++;
                    }
                }
                break;
            case "delete":
                for (Map<String, Object> item : items) {
                    String id = (String) item.get("id");
                    if (id != null) {
                        success++;
                    } else {
                        failed++;
                    }
                }
                break;
            case "deactivate":
                for (Map<String, Object> item : items) {
                    String id = (String) item.get("id");
                    if (id != null) {
                        success++;
                    } else {
                        failed++;
                    }
                }
                break;
            default:
                return Map.of("error", "Unknown operation: " + operation);
        }

        return Map.of("success", success, "failed", failed, "total", items.size());
    }

    private Optional<Map<String, Object>> findById(String id) {
        return Optional.empty();
    }
}
