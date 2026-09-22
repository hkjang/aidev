import { setDateFormat, setDateTimezone } from "../api";
import "./style.css";
export function applyPersonalization(
  preferences: Record<string, unknown> = {},
) {
  const root = document.documentElement;
  root.dataset.density = ["comfortable", "compact", "relaxed"].includes(
    String(preferences.density),
  )
    ? String(preferences.density)
    : "comfortable";
  root.dataset.fontFamily = ["sans", "system", "serif"].includes(
    String(preferences.font_family),
  )
    ? String(preferences.font_family)
    : "sans";
  root.dataset.pageWidth = ["standard", "wide", "full"].includes(
    String(preferences.page_width),
  )
    ? String(preferences.page_width)
    : "standard";
  const dark =
    preferences.code_theme === "dark" ||
    (preferences.code_theme !== "light" && preferences.theme === "dark");
  root.dataset.codeTheme = dark ? "dark" : "light";
  setDateTimezone(
    typeof preferences.timezone === "string" ? preferences.timezone : undefined,
  );
  setDateFormat(
    typeof preferences.date_format === "string"
      ? preferences.date_format
      : undefined,
  );
}
