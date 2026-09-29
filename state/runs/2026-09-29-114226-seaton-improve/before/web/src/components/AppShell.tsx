import { useEffect, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import {
  AppBar,
  Avatar,
  Badge,
  BottomNavigation,
  BottomNavigationAction,
  Box,
  Button,
  Chip,
  Dialog,
  DialogContent,
  DialogTitle,
  Divider,
  Drawer,
  IconButton,
  InputAdornment,
  List,
  ListItemAvatar,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Paper,
  Stack,
  TextField,
  Toolbar,
  Tooltip,
  Typography,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import MenuRounded from "@mui/icons-material/MenuRounded";
import MapRounded from "@mui/icons-material/MapRounded";
import RuleRounded from "@mui/icons-material/RuleRounded";
import LayersRounded from "@mui/icons-material/LayersRounded";
import BadgeRounded from "@mui/icons-material/BadgeRounded";
import HistoryRounded from "@mui/icons-material/HistoryRounded";
import SettingsRounded from "@mui/icons-material/SettingsRounded";
import KeyRounded from "@mui/icons-material/KeyRounded";
import ManageAccountsRounded from "@mui/icons-material/ManageAccountsRounded";
import LogoutRounded from "@mui/icons-material/LogoutRounded";
import SearchRounded from "@mui/icons-material/SearchRounded";
import KeyboardCommandKeyRounded from "@mui/icons-material/KeyboardCommandKeyRounded";
import PersonSearchRounded from "@mui/icons-material/PersonSearchRounded";
import { Logo } from "./Logo";
import { useAuth } from "../auth";
import { api } from "../api";
import type { Employee } from "../types";

const drawerWidth = 252;
const roleName: Record<string, string> = {
  employee: "직원",
  department_manager: "부서 관리자",
  seat_manager: "좌석 관리자",
  system_admin: "시스템 관리자",
};
const pageNames: Record<string, string> = {
  "/": "좌석맵",
  "/admin/actions": "처리필요",
  "/admin/maps": "도면 · 좌석",
  "/admin/employees": "직원",
  "/admin/history": "변경 이력",
  "/admin/users": "사용자 권한",
  "/admin/settings": "시스템 설정",
  "/profile/keys": "내 API 키",
};

export function AppShell() {
  const { user, version, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const theme = useTheme();
  const mobile = useMediaQuery(theme.breakpoints.down("md"));
  const manager =
    user?.role === "seat_manager" || user?.role === "system_admin";
  const admin = user?.role === "system_admin";
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const [commandOpen, setCommandOpen] = useState(false);
  const [commandQuery, setCommandQuery] = useState("");
  const [commandResults, setCommandResults] = useState<Employee[]>([]);
  const [commandSearched, setCommandSearched] = useState(false);
  const [actionCount, setActionCount] = useState(0);

  useEffect(() => {
    const keyboard = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setCommandOpen(true);
      }
    };
    window.addEventListener("keydown", keyboard);
    return () => window.removeEventListener("keydown", keyboard);
  }, []);
  // 배지 숫자만 필요하므로 대시보드 전체가 아니라 전용 집계를 부른다.
  // 화면을 옮길 때마다 13개 쿼리를 돌리던 것을 4개로 줄인다.
  useEffect(() => {
    if (!manager) return;
    api<{ actionRequired: number }>("/api/v1/dashboard/action-count")
      .then((data) => setActionCount(data.actionRequired))
      .catch(() => undefined);
  }, [manager, location.pathname]);
  useEffect(() => {
    setCommandSearched(false);
    if (!commandOpen || commandQuery.trim().length < 2) {
      setCommandResults([]);
      return;
    }
    const timer = window.setTimeout(() => {
      api<{ items: Employee[] }>(
        `/api/v1/employees?q=${encodeURIComponent(commandQuery)}&limit=8`,
      )
        .then((data) => setCommandResults(data.items))
        .catch(() => setCommandResults([]))
        .finally(() => setCommandSearched(true));
    }, 180);
    return () => window.clearTimeout(timer);
  }, [commandOpen, commandQuery]);

  if (!user) return null;
  const items = [
    { label: "좌석맵", path: "/", icon: <MapRounded /> },
    ...(manager
      ? [
          {
            label: "처리필요",
            path: "/admin/actions",
            icon: (
              <Badge color="error" badgeContent={actionCount} max={99}>
                <RuleRounded />
              </Badge>
            ),
          },
          {
            label: "도면 · 좌석",
            path: "/admin/maps",
            icon: <LayersRounded />,
          },
          { label: "직원", path: "/admin/employees", icon: <BadgeRounded /> },
          {
            label: "변경 이력",
            path: "/admin/history",
            icon: <HistoryRounded />,
          },
        ]
      : []),
    ...(admin
      ? [
          {
            label: "사용자 권한",
            path: "/admin/users",
            icon: <ManageAccountsRounded />,
          },
          {
            label: "시스템 설정",
            path: "/admin/settings",
            icon: <SettingsRounded />,
          },
        ]
      : []),
  ];
  const goTo = (path: string) => {
    navigate(path);
    setDrawerOpen(false);
  };
  const drawer = (
    <Box
      sx={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        bgcolor: "#071A2B",
        color: "white",
      }}
    >
      <Box sx={{ px: 2.5, height: 72, display: "flex", alignItems: "center" }}>
        <Logo inverse />
      </Box>
      <Typography
        variant="overline"
        sx={{ px: 2.5, color: "rgba(255,255,255,.4)", letterSpacing: 1.4 }}
      >
        WORKSPACE
      </Typography>
      <List sx={{ px: 1.25, pt: 0.7 }}>
        {items.map((item) => (
          <ListItemButton
            key={item.path}
            selected={location.pathname === item.path}
            onClick={() => goTo(item.path)}
            sx={{
              borderRadius: 2.5,
              mb: 0.45,
              color: "rgba(255,255,255,.72)",
              "& .MuiListItemIcon-root": { color: "inherit", minWidth: 40 },
              "&.Mui-selected": {
                bgcolor: "rgba(78,201,211,.15)",
                color: "#78E1E8",
                "&:hover": { bgcolor: "rgba(78,201,211,.2)" },
              },
              "&:hover": { bgcolor: "rgba(255,255,255,.07)", color: "white" },
            }}
          >
            <ListItemIcon>{item.icon}</ListItemIcon>
            <ListItemText
              primary={item.label}
              primaryTypographyProps={{ fontSize: 14, fontWeight: 650 }}
            />
            {item.path === "/admin/actions" && actionCount > 0 && (
              <Typography variant="caption" fontWeight={800}>
                {actionCount}
              </Typography>
            )}
          </ListItemButton>
        ))}
      </List>
      <Box sx={{ mt: "auto", p: 2 }}>
        <Box
          sx={{
            border: "1px solid rgba(255,255,255,.09)",
            borderRadius: 3,
            p: 1.5,
            bgcolor: "rgba(255,255,255,.035)",
          }}
        >
          <Stack direction="row" spacing={0.8} alignItems="center">
            <Box
              sx={{
                width: 7,
                height: 7,
                borderRadius: "50%",
                bgcolor: "#63D1D9",
                boxShadow: "0 0 0 4px rgba(99,209,217,.1)",
              }}
            />
            <Typography variant="caption" color="rgba(255,255,255,.48)">
              OFFLINE READY
            </Typography>
          </Stack>
          <Typography
            variant="body2"
            sx={{ mt: 0.7, color: "rgba(255,255,255,.78)" }}
          >
            사내 데이터는 망 밖으로 전송되지 않습니다.
          </Typography>
        </Box>
      </Box>
    </Box>
  );
  return (
    <Box sx={{ display: "flex", minHeight: "100vh" }}>
      <AppBar
        position="fixed"
        elevation={0}
        color="inherit"
        sx={{
          ml: { md: `${drawerWidth}px` },
          width: { md: `calc(100% - ${drawerWidth}px)` },
          borderBottom: "1px solid #E3EAEE",
          bgcolor: "rgba(255,255,255,.93)",
          backdropFilter: "blur(12px)",
        }}
      >
        <Toolbar sx={{ height: 72 }}>
          {mobile && (
            <IconButton
              onClick={() => setDrawerOpen(true)}
              edge="start"
              sx={{ mr: 1 }}
            >
              <MenuRounded />
            </IconButton>
          )}
          {mobile ? (
            <Logo compact />
          ) : (
            <Box>
              <Typography variant="caption" color="text.secondary">
                SeatOn / 관리자
              </Typography>
              <Typography variant="body1" fontWeight={750}>
                {pageNames[location.pathname] ?? "SeatOn"}
              </Typography>
            </Box>
          )}
          <Box sx={{ flex: 1 }} />
          <Button
            variant="outlined"
            color="inherit"
            startIcon={<SearchRounded />}
            endIcon={
              <Stack
                direction="row"
                spacing={0.2}
                alignItems="center"
                sx={{ color: "text.disabled" }}
              >
                <KeyboardCommandKeyRounded sx={{ fontSize: 15 }} />
                <Typography variant="caption">K</Typography>
              </Stack>
            }
            onClick={() => setCommandOpen(true)}
            sx={{
              mr: 1.5,
              display: { xs: "none", sm: "flex" },
              minWidth: 215,
              justifyContent: "space-between",
              color: "text.secondary",
              borderColor: "divider",
            }}
          >
            직원 빠른 검색
          </Button>
          <Tooltip title="프로필 및 버전">
            <IconButton
              onClick={(event) => setAnchor(event.currentTarget)}
              sx={{ p: 0.5, borderRadius: 3 }}
            >
              <Avatar
                sx={{
                  width: 36,
                  height: 36,
                  bgcolor: "primary.main",
                  fontSize: 15,
                  fontWeight: 750,
                }}
              >
                {user.displayName.slice(0, 1).toUpperCase()}
              </Avatar>
            </IconButton>
          </Tooltip>
          <Box
            onClick={(event) => setAnchor(event.currentTarget)}
            sx={{
              ml: 1,
              display: { xs: "none", sm: "block" },
              cursor: "pointer",
            }}
          >
            <Typography variant="body2" fontWeight={750}>
              {user.displayName}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              {roleName[user.role]}
            </Typography>
          </Box>
          <Menu
            anchorEl={anchor}
            open={Boolean(anchor)}
            onClose={() => setAnchor(null)}
            slotProps={{
              paper: {
                sx: { mt: 1, minWidth: 250, border: "1px solid #E3EAEE" },
              },
            }}
          >
            <Box sx={{ px: 2, py: 1.3 }}>
              <Typography fontWeight={750}>{user.displayName}</Typography>
              <Typography variant="body2" color="text.secondary">
                {user.email || user.username}
              </Typography>
            </Box>
            <Divider />
            <MenuItem
              onClick={() => {
                navigate("/profile/keys");
                setAnchor(null);
              }}
            >
              <ListItemIcon>
                <KeyRounded fontSize="small" />
              </ListItemIcon>
              내 API 키
            </MenuItem>
            <MenuItem disabled sx={{ opacity: "1!important" }}>
              <ListItemText
                primary={`SeatOn ${version?.version ?? "dev"}`}
                secondary={
                  version?.commit && version.commit !== "unknown"
                    ? `commit ${version.commit.slice(0, 8)}`
                    : "오프라인 배포판"
                }
              />
            </MenuItem>
            <Divider />
            <MenuItem onClick={() => void logout()}>
              <ListItemIcon>
                <LogoutRounded fontSize="small" />
              </ListItemIcon>
              로그아웃
            </MenuItem>
          </Menu>
        </Toolbar>
      </AppBar>
      <Box
        component="nav"
        sx={{ width: { md: drawerWidth }, flexShrink: { md: 0 } }}
      >
        <Drawer
          variant={mobile ? "temporary" : "permanent"}
          open={mobile ? drawerOpen : true}
          onClose={() => setDrawerOpen(false)}
          ModalProps={{ keepMounted: true }}
          sx={{ "& .MuiDrawer-paper": { width: drawerWidth, border: 0 } }}
        >
          {drawer}
        </Drawer>
      </Box>
      <Box
        component="main"
        sx={{
          flex: 1,
          minWidth: 0,
          pt: "72px",
          pb: { xs: manager ? 8 : 0, md: 0 },
        }}
      >
        <Outlet />
      </Box>
      {mobile && manager && (
        <Paper
          elevation={8}
          sx={{
            position: "fixed",
            left: 0,
            right: 0,
            bottom: 0,
            zIndex: 1200,
            borderRadius: 0,
          }}
        >
          <BottomNavigation
            showLabels
            value={location.pathname}
            onChange={(_, value) => navigate(value)}
          >
            <BottomNavigationAction
              label="좌석맵"
              value="/"
              icon={<MapRounded />}
            />
            <BottomNavigationAction
              label="처리필요"
              value="/admin/actions"
              icon={
                <Badge color="error" badgeContent={actionCount} max={99}>
                  <RuleRounded />
                </Badge>
              }
            />
            <BottomNavigationAction
              label="도면"
              value="/admin/maps"
              icon={<LayersRounded />}
            />
            <BottomNavigationAction
              label="직원"
              value="/admin/employees"
              icon={<BadgeRounded />}
            />
          </BottomNavigation>
        </Paper>
      )}
      <Dialog
        open={commandOpen}
        onClose={() => setCommandOpen(false)}
        fullWidth
        maxWidth="sm"
        slotProps={{
          paper: {
            sx: {
              position: "fixed",
              top: { xs: 8, sm: 70 },
              m: 0,
              borderRadius: 3,
            },
          },
        }}
      >
        <DialogTitle sx={{ pb: 1 }}>사람 또는 조직 빠른 검색</DialogTitle>
        <DialogContent sx={{ px: 2, pb: 2 }}>
          <TextField
            autoFocus
            fullWidth
            value={commandQuery}
            onChange={(event) => setCommandQuery(event.target.value)}
            placeholder="이름, 사번, 이메일, 조직을 입력하세요"
            slotProps={{
              input: {
                startAdornment: (
                  <InputAdornment position="start">
                    <PersonSearchRounded />
                  </InputAdornment>
                ),
              },
            }}
          />
          <List sx={{ mt: 1, maxHeight: 420, overflowY: "auto" }}>
            {commandResults.map((employee) => (
              <ListItemButton
                key={employee.id}
                onClick={() => {
                  navigate(`/?q=${encodeURIComponent(employee.employeeNo)}`);
                  setCommandOpen(false);
                  setCommandQuery("");
                }}
                sx={{ borderRadius: 2 }}
              >
                <ListItemAvatar>
                  <Avatar
                    sx={{
                      width: 36,
                      height: 36,
                      bgcolor: employee.seatNo ? "primary.main" : "grey.400",
                      fontSize: 14,
                    }}
                  >
                    {employee.name.slice(0, 1)}
                  </Avatar>
                </ListItemAvatar>
                <ListItemText
                  primary={employee.name}
                  secondary={`${employee.organizationName || "소속 없음"} · ${employee.seatNo || "미배정"}`}
                />
                <Chip
                  size="small"
                  label={employee.employeeNo}
                  variant="outlined"
                />
              </ListItemButton>
            ))}
            {commandQuery.length >= 2 &&
              commandSearched &&
              commandResults.length === 0 && (
                <Stack alignItems="center" spacing={1} sx={{ py: 5 }}>
                  <SearchRounded color="disabled" />
                  <Typography variant="body2" color="text.secondary">
                    일치하는 직원이나 조직이 없습니다.
                  </Typography>
                </Stack>
              )}
            {commandQuery.length < 2 && (
              <Typography
                variant="body2"
                color="text.secondary"
                textAlign="center"
                sx={{ py: 4 }}
              >
                두 글자 이상 입력하면 바로 검색합니다. · Ctrl/⌘ + K
              </Typography>
            )}
          </List>
        </DialogContent>
      </Dialog>
    </Box>
  );
}
