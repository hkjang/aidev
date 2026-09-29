import { Box, Typography } from "@mui/material";

export function Logo({
  compact = false,
  inverse = false,
}: {
  compact?: boolean;
  inverse?: boolean;
}) {
  return (
    <Box sx={{ display: "flex", alignItems: "center", gap: 1.2, minWidth: 0 }}>
      <Box
        aria-hidden
        sx={{
          width: 34,
          height: 34,
          borderRadius: "11px",
          display: "grid",
          placeItems: "center",
          bgcolor: inverse ? "rgba(255,255,255,.13)" : "primary.main",
          boxShadow: inverse ? "none" : "0 6px 18px rgba(8,126,139,.25)",
        }}
      >
        <Box
          sx={{
            position: "relative",
            width: 18,
            height: 18,
            border: "2px solid white",
            borderRadius: "5px 5px 8px 8px",
            "&::after": {
              content: '""',
              position: "absolute",
              width: 8,
              height: 2,
              bgcolor: "secondary.main",
              left: 3,
              top: 4,
              borderRadius: 2,
            },
          }}
        />
      </Box>
      {!compact && (
        <Typography
          variant="h6"
          sx={{
            fontWeight: 850,
            letterSpacing: "-.045em",
            color: inverse ? "white" : "text.primary",
          }}
        >
          Seat<span style={{ color: inverse ? "#63D1D9" : "#087E8B" }}>On</span>
        </Typography>
      )}
    </Box>
  );
}
