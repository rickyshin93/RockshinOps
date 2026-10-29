import { useState } from "react";

export function MenuBarIcon({ image, name }: { image: string; name: string }) {
  const [failedImage, setFailedImage] = useState<string | null>(null);
  return image && image !== failedImage
    ? <img src={image} alt={name} draggable={false} onError={() => setFailedImage(image)} />
    : <svg className="menu-icon-placeholder" viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeDasharray="3 2" aria-hidden="true"><rect x="4" y="4" width="16" height="16" rx="4" /></svg>;
}
