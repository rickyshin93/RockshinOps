export function resolveLanguage(preferred: readonly string[]): "en" | "zh" {
  for (const tag of preferred) {
    const language = tag.toLowerCase().split(/[-_]/)[0];
    if (language === "zh" || language === "en") return language;
  }
  return "en";
}
