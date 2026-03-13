package com.example.services;

import java.math.BigDecimal;
import java.util.Map;

/**
 * Payment service — processes payments with validation.
 * Known complexity: processPayment=7, calculateFees=4
 */
public class PaymentService {

    // complexity: 1 (base) + 6 decision points = 7
    public Map<String, Object> processPayment(BigDecimal amount, String currency, String method) {
        if (amount == null || amount.compareTo(BigDecimal.ZERO) <= 0) {
            return Map.of("error", "Invalid amount");
        }

        if (amount.compareTo(new BigDecimal("50000")) > 0) {
            return Map.of("error", "Amount exceeds limit");
        }

        if (currency == null || currency.isEmpty()) {
            currency = "USD";
        }

        BigDecimal fees = calculateFees(amount, currency);
        BigDecimal total = amount.add(fees);

        try {
            String txId = "tx_" + System.currentTimeMillis();

            if ("wire".equals(method)) {
                return Map.of("transaction_id", txId, "total", total, "method", "wire", "status", "pending");
            } else if ("card".equals(method)) {
                return Map.of("transaction_id", txId, "total", total, "method", "card", "status", "completed");
            }

            return Map.of("error", "Unsupported payment method");
        } catch (Exception e) {
            return Map.of("error", "Payment processing failed");
        }
    }

    // complexity: 1 (base) + 3 decision points = 4
    public BigDecimal calculateFees(BigDecimal amount, String currency) {
        BigDecimal rate;

        if ("USD".equals(currency)) {
            rate = new BigDecimal("0.029");
        } else if ("EUR".equals(currency)) {
            rate = new BigDecimal("0.034");
        } else {
            rate = new BigDecimal("0.039");
        }

        BigDecimal fee = amount.multiply(rate);
        if (fee.compareTo(new BigDecimal("0.50")) < 0) {
            fee = new BigDecimal("0.50");
        }

        return fee;
    }
}
