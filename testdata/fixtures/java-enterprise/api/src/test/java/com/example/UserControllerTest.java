package com.example;

import com.example.controllers.UserController;
import java.util.Map;

/**
 * Basic tests for UserController.
 */
public class UserControllerTest {

    public void testCreateUser() {
        UserController controller = new UserController();
        Map<String, Object> result = controller.createUser(
            Map.of("email", "test@example.com", "name", "Test User")
        );
        assert result.containsKey("id") : "Should return user ID";
    }

    public void testCreateUserInvalidEmail() {
        UserController controller = new UserController();
        Map<String, Object> result = controller.createUser(
            Map.of("email", "invalid", "name", "Test")
        );
        assert result.containsKey("error") : "Should return error";
    }

    public void testGetUserNotFound() {
        UserController controller = new UserController();
        Map<String, Object> result = controller.getUser("nonexistent");
        assert result.containsKey("error") : "Should return not found";
    }
}
