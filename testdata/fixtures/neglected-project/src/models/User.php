<?php
// User model — simple data class
// Known complexity: all methods 1-2

namespace App\Models;

class User
{
    public $id;
    public $name;
    public $email;
    public $role;
    public $active;

    public function __construct($data = [])
    {
        $this->id = $data['id'] ?? null;
        $this->name = $data['name'] ?? '';
        $this->email = $data['email'] ?? '';
        $this->role = $data['role'] ?? 'user';
        $this->active = $data['active'] ?? true;
    }

    public function isAdmin()
    {
        return $this->role === 'admin';
    }

    public function toArray()
    {
        return [
            'id' => $this->id,
            'name' => $this->name,
            'email' => $this->email,
            'role' => $this->role,
            'active' => $this->active,
        ];
    }
}
