import { alpha, createTheme } from "@mui/material/styles";

export const theme = createTheme({
  palette: {
    mode: "light",
    primary: {
      main: "#087E8B",
      dark: "#045C65",
      light: "#3CAAB4",
      contrastText: "#fff",
    },
    secondary: { main: "#FFB703", dark: "#D38B00" },
    background: { default: "#F4F7F9", paper: "#FFFFFF" },
    text: { primary: "#102A3B", secondary: "#5E7180" },
    success: { main: "#2E9D67" },
    warning: { main: "#E79418" },
    error: { main: "#D64D55" },
    info: { main: "#3478C8" },
  },
  shape: { borderRadius: 14 },
  typography: {
    fontFamily:
      'Pretendard, Inter, "Noto Sans KR", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
    h4: { fontWeight: 750, letterSpacing: "-0.035em" },
    h5: { fontWeight: 750, letterSpacing: "-0.025em" },
    h6: { fontWeight: 700 },
    button: { fontWeight: 700, textTransform: "none" },
  },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: { minWidth: 320 },
        "*": { boxSizing: "border-box" },
        "::selection": { backgroundColor: alpha("#087E8B", 0.2) },
      },
    },
    MuiPaper: { styleOverrides: { root: { backgroundImage: "none" } } },
    MuiButton: {
      defaultProps: { disableElevation: true },
      styleOverrides: { root: { borderRadius: 10, minHeight: 40 } },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          border: "1px solid #E4EBEF",
          boxShadow: "0 8px 30px rgba(14,45,62,.06)",
        },
      },
    },
    MuiTextField: { defaultProps: { size: "small" } },
    MuiFormControl: { defaultProps: { size: "small" } },
  },
});
