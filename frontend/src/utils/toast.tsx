// Simple toast implementation to replace @nextui-org/toast which doesn't exist on npm
// Uses browser alerts as a fallback - can be replaced with a proper toast library later

type ToastType = "success" | "error" | "warning" | "info";

interface ToastOptions {
  title?: string;
  description?: string;
  type?: ToastType;
  color?: string;
}

export function addToast(options: ToastOptions): void {
  const { title, description, type = "info" } = options;
  const message = [title, description].filter(Boolean).join(": ");
  
  // For now, just use console.log - the UI will still work
  // In production, integrate a proper toast library like react-hot-toast
  const prefix = type === "error" ? "❌" : type === "success" ? "✅" : type === "warning" ? "⚠️" : "ℹ️";
  console.log(`${prefix} ${message}`);
}

// Dummy ToastProvider that does nothing - toasts are handled via console
export function ToastProvider(): null {
  return null;
}
