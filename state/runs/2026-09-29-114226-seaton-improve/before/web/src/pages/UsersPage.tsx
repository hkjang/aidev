import { useEffect, useMemo, useState } from "react";
import {
  Alert,
  Avatar,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  TextField,
  Tooltip,
  MenuItem,
  Paper,
  Select,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import EditRounded from "@mui/icons-material/EditRounded";
import PeopleAltRounded from "@mui/icons-material/PeopleAltRounded";
import SearchRounded from "@mui/icons-material/SearchRounded";
import { api, patchJSON } from "../api";
import { useAuth } from "../auth";
import { EmptyState, PageHeader, TableSkeleton } from "../components/AdminUI";
import { relativeTime, absoluteTime } from "../lib/format";
import {
  canChangeRole,
  canToggleActive,
  emailEditable,
  normalizeEmail,
} from "../lib/users";
import type { Role, User } from "../types";
const labels: Record<Role, string> = {
  employee: "직원",
  department_manager: "부서 관리자",
  seat_manager: "좌석 관리자",
  system_admin: "시스템 관리자",
};
export function UsersPage() {
  const { user: me } = useAuth();
  const [items, setItems] = useState<User[]>([]),
    [query, setQuery] = useState(""),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    // 메일 주소 편집 창. 어느 계정을 고치는지와 입력 중인 값을 든다.
    [editing, setEditing] = useState<User | null>(null),
    [draft, setDraft] = useState(""),
    [draftError, setDraftError] = useState("");
  const load = async () => {
    try {
      const data = await api<{ items: User[] }>("/api/v1/users");
      setItems(data.items);
    } catch (e) {
      setError(e instanceof Error ? e.message : "사용자를 불러오지 못했습니다");
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load();
  }, []);
  // 사용자 수가 늘면 이름으로 찾는 것이 유일하게 현실적인 방법이다.
  const visible = useMemo(() => {
    const term = query.trim().toLowerCase();
    if (!term) return items;
    return items.filter((u) =>
      [u.displayName, u.username, u.email ?? ""].some((v) =>
        v.toLowerCase().includes(term),
      ),
    );
  }, [items, query]);
  const patch = async (
    id: string,
    body: { role?: Role; active?: boolean; email?: string },
    failure: string,
  ) => {
    try {
      await patchJSON(`/api/v1/users/${id}`, body);
      await load();
      return true;
    } catch (e) {
      setError(e instanceof Error ? e.message : failure);
      return false;
    }
  };
  const change = (id: string, role: Role) =>
    patch(id, { role }, "권한을 변경하지 못했습니다");
  const toggleActive = (u: User) =>
    patch(u.id, { active: !u.active }, "계정 상태를 변경하지 못했습니다");
  const openEmail = (u: User) => {
    setEditing(u);
    setDraft(u.email ?? "");
    setDraftError("");
  };
  const saveEmail = async () => {
    if (!editing) return;
    const { value, error: reason } = normalizeEmail(draft);
    if (reason) {
      setDraftError(reason);
      return;
    }
    if (
      await patch(editing.id, { email: value }, "메일 주소를 저장하지 못했습니다")
    ) {
      setEditing(null);
    }
  };
  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1100, mx: "auto" }}>
      <PageHeader
        eyebrow="ACCESS CONTROL"
        title="사용자 권한"
        description="SSO 사용자는 최초 로그인 시 자동 생성되고 Keycloak 그룹으로 기본 권한이 결정됩니다."
        actions={
          <TextField
            size="small"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="이름, 계정, 이메일"
            slotProps={{
              input: {
                startAdornment: (
                  <SearchRounded sx={{ mr: 1, color: "text.disabled" }} />
                ),
              },
            }}
          />
        }
      />
      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError("")}>
          {error}
        </Alert>
      )}
      {loading ? (
        <TableSkeleton />
      ) : visible.length === 0 ? (
        <EmptyState
          icon={<PeopleAltRounded />}
          title={
            query
              ? "조건에 맞는 사용자가 없습니다."
              : "아직 등록된 사용자가 없습니다."
          }
          description={
            query
              ? undefined
              : "Keycloak SSO로 로그인하면 사용자가 자동으로 만들어집니다."
          }
        />
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>사용자</TableCell>
                <TableCell>메일 주소</TableCell>
                <TableCell>로그인 방식</TableCell>
                <TableCell>최근 로그인</TableCell>
                <TableCell>권한</TableCell>
                <TableCell>사용</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {visible.map((u) => (
                <TableRow key={u.id} sx={u.active ? undefined : { opacity: 0.6 }}>
                  <TableCell>
                    <Stack direction="row" alignItems="center" spacing={1}>
                      <Avatar
                        sx={{
                          width: 34,
                          height: 34,
                          bgcolor: u.active ? "primary.main" : "text.disabled",
                          fontSize: 13,
                        }}
                      >
                        {u.displayName.slice(0, 1)}
                      </Avatar>
                      <Box>
                        <Typography variant="body2" fontWeight={700}>
                          {u.displayName}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {u.username}
                        </Typography>
                      </Box>
                    </Stack>
                  </TableCell>
                  <TableCell>
                    {/* 알림 메일이 닿는 주소. 로컬 계정은 처음부터 주소가 없어
                        여기서 넣지 않으면 분석 완료·키 만료 같은 알림을 못 받는다. */}
                    <Stack direction="row" alignItems="center" spacing={0.5}>
                      <Typography
                        variant="body2"
                        color={u.email ? "text.primary" : "text.disabled"}
                      >
                        {u.email || "주소 없음"}
                      </Typography>
                      {emailEditable(u) ? (
                        <Tooltip title="메일 주소 변경">
                          <IconButton
                            size="small"
                            aria-label={`${u.displayName} 메일 주소 변경`}
                            onClick={() => openEmail(u)}
                          >
                            <EditRounded fontSize="inherit" />
                          </IconButton>
                        </Tooltip>
                      ) : (
                        <Tooltip title="Keycloak 프로필의 주소를 로그인할 때마다 가져옵니다">
                          <Typography variant="caption" color="text.disabled">
                            SSO
                          </Typography>
                        </Tooltip>
                      )}
                    </Stack>
                  </TableCell>
                  <TableCell>
                    <Chip
                      size="small"
                      variant="outlined"
                      label={u.source === "oidc" ? "Keycloak SSO" : "Local"}
                    />
                  </TableCell>
                  <TableCell sx={{ whiteSpace: "nowrap" }}>
                    {u.lastLoginAt ? (
                      <Tooltip title={absoluteTime(u.lastLoginAt)}>
                        <span>{relativeTime(u.lastLoginAt)}</span>
                      </Tooltip>
                    ) : (
                      "-"
                    )}
                  </TableCell>
                  <TableCell>
                    {/* 자기 권한을 낮추면 이 화면을 더는 열 수 없어 자기 행은 막는다. */}
                    <Tooltip
                      title={
                        canChangeRole(u, me)
                          ? ""
                          : "자기 계정의 권한은 낮출 수 없습니다"
                      }
                    >
                      <span>
                        <Select
                          size="small"
                          value={u.role}
                          disabled={!canChangeRole(u, me)}
                          onChange={(e) =>
                            void change(u.id, e.target.value as Role)
                          }
                          inputProps={{ "aria-label": `${u.displayName} 권한` }}
                        >
                          {Object.entries(labels).map(([value, label]) => (
                            <MenuItem key={value} value={value}>
                              {label}
                            </MenuItem>
                          ))}
                        </Select>
                      </span>
                    </Tooltip>
                  </TableCell>
                  <TableCell>
                    {/* 끄면 로그인·세션·API 키가 모두 거부된다. 자기 계정은 끌 수 없다. */}
                    <Tooltip
                      title={
                        canToggleActive(u, me)
                          ? u.active
                            ? "끄면 로그인·세션·API 키가 모두 거부됩니다"
                            : "다시 켜면 바로 로그인할 수 있습니다"
                          : "자기 계정은 비활성화할 수 없습니다"
                      }
                    >
                      <span>
                        <Switch
                          size="small"
                          checked={u.active}
                          disabled={!canToggleActive(u, me)}
                          onChange={() => void toggleActive(u)}
                          slotProps={{
                            input: {
                              "aria-label": `${u.displayName} 계정 사용`,
                            },
                          }}
                        />
                      </span>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      <Dialog
        open={Boolean(editing)}
        onClose={() => setEditing(null)}
        fullWidth
        maxWidth="xs"
      >
        <DialogTitle>메일 주소 변경</DialogTitle>
        <DialogContent>
          <Typography variant="body2" color="text.secondary" mb={2}>
            <strong>{editing?.displayName}</strong>({editing?.username}) 계정이
            알림 메일을 받을 주소입니다. 비우면 주소를 지웁니다.
          </Typography>
          <TextField
            autoFocus
            fullWidth
            size="small"
            label="메일 주소"
            type="email"
            value={draft}
            error={Boolean(draftError)}
            helperText={draftError || " "}
            onChange={(e) => {
              setDraft(e.target.value);
              setDraftError("");
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                void saveEmail();
              }
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditing(null)}>취소</Button>
          <Button variant="contained" onClick={() => void saveEmail()}>
            저장
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
