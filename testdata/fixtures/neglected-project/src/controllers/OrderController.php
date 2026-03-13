<?php
// OrderController — legacy controller with high complexity
// Known complexity: processOrder=18, calculateTotal=10

namespace App\Controllers;

class OrderController
{
    // complexity: 1 (base) + 17 decision points = 18
    public function processOrder($request)
    {
        $data = $request->getBody();

        if (!isset($data['items']) || empty($data['items'])) {
            return ['error' => 'No items in order'];
        }

        $total = 0;
        $errors = [];

        foreach ($data['items'] as $item) {
            if (!isset($item['product_id'])) {
                $errors[] = 'Missing product ID';
                continue;
            }

            $product = $this->getProduct($item['product_id']);
            if (!$product) {
                $errors[] = "Product {$item['product_id']} not found";
                continue;
            }

            if (!$product['in_stock']) {
                $errors[] = "Product {$product['name']} out of stock";
                continue;
            }

            $qty = isset($item['quantity']) ? $item['quantity'] : 1;
            if ($qty <= 0) {
                $errors[] = "Invalid quantity for {$product['name']}";
                continue;
            }

            if ($qty > $product['max_qty']) {
                $qty = $product['max_qty'];
            }

            $price = $product['price'] * $qty;

            if (isset($item['coupon'])) {
                $discount = $this->applyCoupon($item['coupon'], $price);
                if ($discount > 0) {
                    $price -= $discount;
                }
            }

            $total += $price;
        }

        if (!empty($errors)) {
            return ['errors' => $errors, 'partial' => true];
        }

        if ($total <= 0) {
            return ['error' => 'Invalid order total'];
        }

        if (isset($data['shipping'])) {
            if ($data['shipping'] === 'express') {
                $total += 15.99;
            } elseif ($data['shipping'] === 'overnight') {
                $total += 29.99;
            }
        }

        return [
            'order_id' => uniqid('ord_'),
            'total' => $total,
            'status' => 'pending',
        ];
    }

    // complexity: 1 (base) + 9 decision points = 10
    public function calculateTotal($items, $discountCode = null, $taxRate = 0.0)
    {
        $subtotal = 0;

        foreach ($items as $item) {
            if (!isset($item['price']) || !isset($item['qty'])) {
                continue;
            }

            $lineTotal = $item['price'] * $item['qty'];

            if (isset($item['discount_pct'])) {
                $lineTotal *= (1 - $item['discount_pct'] / 100);
            }

            if ($lineTotal < 0) {
                $lineTotal = 0;
            }

            $subtotal += $lineTotal;
        }

        if ($discountCode) {
            $discount = $this->lookupDiscount($discountCode);
            if ($discount && $discount['type'] === 'percent') {
                $subtotal *= (1 - $discount['value'] / 100);
            } elseif ($discount && $discount['type'] === 'fixed') {
                $subtotal -= $discount['value'];
            }
        }

        if ($subtotal < 0) {
            $subtotal = 0;
        }

        $tax = $subtotal * $taxRate;
        return $subtotal + $tax;
    }

    private function getProduct($id) { return null; }
    private function applyCoupon($code, $price) { return 0; }
    private function lookupDiscount($code) { return null; }
}
