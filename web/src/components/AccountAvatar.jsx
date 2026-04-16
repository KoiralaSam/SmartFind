import { useState } from "react";
import { accountPictureUrl, legoAvatarFromSeed } from "../lib/avatarUrl";

export function AccountAvatar({
  user,
  sizeClass = "h-9 w-9",
  className = "",
}) {
  const initial = accountPictureUrl(user);
  const [src, setSrc] = useState(initial);
  const label = user?.name || user?.email || "Account";

  return (
    <span
      className={`relative inline-flex ${sizeClass} shrink-0 overflow-hidden rounded-full border border-border/80 bg-muted ${className}`}
    >
      <img
        src={src}
        alt={label}
        referrerPolicy="no-referrer"
        className="h-full w-full object-cover"
        onError={() => {
          const fallback = user?.email
            ? legoAvatarFromSeed(`${user.email}-fallback`)
            : accountPictureUrl({ email: "local" });
          if (src !== fallback) setSrc(fallback);
        }}
      />
    </span>
  );
}
