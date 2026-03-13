<?php
// UserController — legacy controller with high complexity
// Known complexity: handleRequest=15, processForm=12, validateInput=8

namespace App\Controllers;

class UserController
{
    // complexity: 1 (base) + 14 decision points = 15
    public function handleRequest($request, $response)
    {
        $method = $request->getMethod();
        $path = $request->getPath();

        if ($method === 'GET') {
            if ($path === '/users') {
                $page = $request->get('page', 1);
                $limit = $request->get('limit', 20);
                if ($limit > 100) {
                    $limit = 100;
                }
                $users = $this->getUsers($page, $limit);
                if (empty($users)) {
                    return $response->json(['error' => 'No users found'], 404);
                }
                return $response->json($users);
            } elseif (preg_match('/\/users\/(\d+)/', $path, $matches)) {
                $user = $this->getUser($matches[1]);
                if (!$user) {
                    return $response->json(['error' => 'User not found'], 404);
                }
                return $response->json($user);
            }
        } elseif ($method === 'POST') {
            if ($path === '/users') {
                $data = $request->getBody();
                $errors = $this->validateInput($data);
                if (!empty($errors)) {
                    return $response->json(['errors' => $errors], 422);
                }
                $user = $this->createUser($data);
                if (!$user) {
                    return $response->json(['error' => 'Failed to create user'], 500);
                }
                return $response->json($user, 201);
            }
        } elseif ($method === 'DELETE') {
            if (preg_match('/\/users\/(\d+)/', $path, $matches)) {
                $this->deleteUser($matches[1]);
                return $response->json(['status' => 'deleted']);
            }
        }

        return $response->json(['error' => 'Not found'], 404);
    }

    // complexity: 1 (base) + 11 decision points = 12
    public function processForm($data)
    {
        $result = [];

        if (!isset($data['name']) || empty($data['name'])) {
            $result['errors'][] = 'Name is required';
        } elseif (strlen($data['name']) < 2) {
            $result['errors'][] = 'Name too short';
        } elseif (strlen($data['name']) > 100) {
            $result['errors'][] = 'Name too long';
        }

        if (isset($data['email'])) {
            if (!filter_var($data['email'], FILTER_VALIDATE_EMAIL)) {
                $result['errors'][] = 'Invalid email';
            } else {
                $existing = $this->findByEmail($data['email']);
                if ($existing && $existing['id'] !== ($data['id'] ?? null)) {
                    $result['errors'][] = 'Email already taken';
                }
            }
        }

        if (isset($data['age'])) {
            if (!is_numeric($data['age'])) {
                $result['errors'][] = 'Age must be numeric';
            } elseif ($data['age'] < 0 || $data['age'] > 150) {
                $result['errors'][] = 'Invalid age';
            }
        }

        if (empty($result['errors'])) {
            $result['valid'] = true;
        }

        return $result;
    }

    // complexity: 1 (base) + 7 decision points = 8
    public function validateInput($data)
    {
        $errors = [];

        if (!is_array($data)) {
            return ['Invalid input format'];
        }

        foreach (['name', 'email'] as $required) {
            if (!isset($data[$required]) || empty($data[$required])) {
                $errors[] = "$required is required";
            }
        }

        if (isset($data['email']) && !filter_var($data['email'], FILTER_VALIDATE_EMAIL)) {
            $errors[] = 'Invalid email format';
        }

        if (isset($data['role'])) {
            $validRoles = ['user', 'admin', 'moderator'];
            if (!in_array($data['role'], $validRoles)) {
                $errors[] = 'Invalid role';
            }
        }

        if (isset($data['password'])) {
            if (strlen($data['password']) < 8) {
                $errors[] = 'Password too short';
            }
            if (!preg_match('/[A-Z]/', $data['password'])) {
                $errors[] = 'Password needs uppercase';
            }
        }

        return $errors;
    }

    private function getUsers($page, $limit) { return []; }
    private function getUser($id) { return null; }
    private function createUser($data) { return $data; }
    private function deleteUser($id) { }
    private function findByEmail($email) { return null; }
}
