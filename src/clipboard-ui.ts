export function clipboardImageURL(id: string, preview = false): string {
  return `clipboard-preview://localhost/${preview ? "preview" : "thumbnail"}?id=${encodeURIComponent(id)}`;
}

export function clipboardWarningKey(code?: string): string {
  if (code === "too-large") return "这条剪贴板内容超过单条大小上限，未记录。";
  if (code === "write") return "无法保存剪贴板内容，请检查本机存储。";
  return "";
}
