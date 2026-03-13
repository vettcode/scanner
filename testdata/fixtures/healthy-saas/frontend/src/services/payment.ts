// Payment service — handles billing flow
// Known complexity: processPayment=7, validateCard=7, formatReceipt=2

interface PaymentResult {
  success: boolean;
  transactionId: string;
  error?: string;
}

interface Card {
  number: string;
  expiry: string;
  cvv: string;
  type?: string;
}

export function processPayment(amount: number, card: Card, currency: string): PaymentResult {
  // complexity: 1 (base) + 6 decision points = 7 (3 ifs + ternary + if + catch)
  if (amount <= 0) {
    return { success: false, transactionId: "", error: "Invalid amount" };
  }
  if (!validateCard(card)) {
    return { success: false, transactionId: "", error: "Invalid card" };
  }

  const fee = currency === "USD" ? amount * 0.029 : amount * 0.039;
  const total = amount + fee;

  if (total > 10000) {
    return { success: false, transactionId: "", error: "Amount exceeds limit" };
  }

  try {
    const txId = `tx_${Date.now()}`;
    if (card.type === "prepaid") {
      return { success: true, transactionId: txId };
    }
    return { success: true, transactionId: txId };
  } catch (e) {
    return { success: false, transactionId: "", error: "Processing failed" };
  }
}

export function validateCard(card: Card): boolean {
  // complexity: 1 (base) + 6 decision points = 7 (3 ifs + 3 || operators)
  if (!card.number || card.number.length < 13) {
    return false;
  }
  if (!card.expiry || !card.expiry.match(/^\d{2}\/\d{2}$/)) {
    return false;
  }
  if (!card.cvv || card.cvv.length < 3) {
    return false;
  }
  return true;
}

export function formatReceipt(txId: string, amount: number): string {
  // complexity: 1 (base) + 1 decision point = 2
  const date = new Date().toISOString();
  if (amount >= 100) {
    return `Receipt ${txId}: $${amount.toFixed(2)} on ${date} [LARGE]`;
  }
  return `Receipt ${txId}: $${amount.toFixed(2)} on ${date}`;
}
