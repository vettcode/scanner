import { processPayment, validateCard, formatReceipt } from "../src/services/payment";

describe("processPayment", () => {
  it("rejects negative amounts", () => {
    const result = processPayment(-10, { number: "4111111111111111", expiry: "12/25", cvv: "123" }, "USD");
    expect(result.success).toBe(false);
  });

  it("processes valid payment", () => {
    const result = processPayment(50, { number: "4111111111111111", expiry: "12/25", cvv: "123" }, "USD");
    expect(result.success).toBe(true);
  });
});

describe("validateCard", () => {
  it("rejects short card numbers", () => {
    expect(validateCard({ number: "123", expiry: "12/25", cvv: "123" })).toBe(false);
  });

  it("accepts valid cards", () => {
    expect(validateCard({ number: "4111111111111111", expiry: "12/25", cvv: "123" })).toBe(true);
  });
});

describe("formatReceipt", () => {
  it("formats large receipt with label", () => {
    const receipt = formatReceipt("tx_123", 150);
    expect(receipt).toContain("[LARGE]");
  });
});
