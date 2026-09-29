import { useEffect, useState } from "react";
import {
  Alert,
  Box,
  Button,
  Checkbox,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  FormGroup,
  IconButton,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";
import AddRounded from "@mui/icons-material/AddRounded";
import AutorenewRounded from "@mui/icons-material/AutorenewRounded";
import DeleteOutlineRounded from "@mui/icons-material/DeleteOutlineRounded";
import ContentCopyRounded from "@mui/icons-material/ContentCopyRounded";
import VpnKeyRounded from "@mui/icons-material/VpnKeyRounded";
import { api, postJSON } from "../api";
import { EmptyState, PageHeader, TableSkeleton } from "../components/AdminUI";
type KeyItem = {
  id: string;
  name: string;
  prefix: string;
  scopes: string[];
  version: number;
  createdAt: string;
  expiresAt?: string;
  lastUsedAt?: string;
  revokedAt?: string;
  graceUntil?: string;
};
/** 키 한 줄의 상태. 폐기·유예·사용 중을 색과 글로 함께 구분한다. */
const keyStatus = (
  key: KeyItem,
): { label: string; color: "default" | "success" | "warning" } => {
  if (key.revokedAt && key.graceUntil && new Date(key.graceUntil) > new Date())
    return { label: "회전 유예", color: "warning" };
  if (key.revokedAt) return { label: "폐기됨", color: "default" };
  return { label: "사용 중", color: "success" };
};

export function KeysPage() {
  const [items, setItems] = useState<KeyItem[]>([]),
    [createOpen, setCreateOpen] = useState(false),
    [name, setName] = useState("내 연동 키"),
    [scopes, setScopes] = useState(["read", "mcp"]),
    [revealed, setRevealed] = useState<{ key: string; message: string } | null>(
      null,
    ),
    [loading, setLoading] = useState(true),
    // sso 는 서버가 내는 RFC 9728 메타데이터 그대로다. 관리자가 MCP SSO 를 켰을
    // 때만 200 이므로, 이 문서가 있다는 것이 곧 "키 없이 연결할 수 있다"는 뜻이다.
    [sso, setSso] = useState<{
      resource: string;
      authorization_servers: string[];
    } | null>(null),
    [error, setError] = useState("");
  useEffect(() => {
    void (async () => {
      try {
        const response = await fetch(
          "/.well-known/oauth-protected-resource/mcp",
        );
        if (response.ok) setSso(await response.json());
      } catch {
        // 메타데이터를 못 읽으면 키 안내만 보인다.
      }
    })();
  }, []);
  const load = async () => {
    try {
      const data = await api<{ items: KeyItem[] }>("/api/v1/api-keys");
      setItems(data.items);
    } catch (e) {
      setError(e instanceof Error ? e.message : "키를 불러오지 못했습니다");
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load();
  }, []);
  const create = async () => {
    try {
      const x = await postJSON<{ key: string; message: string }>(
        "/api/v1/api-keys",
        { name, scopes },
      );
      setCreateOpen(false);
      setRevealed(x);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "키를 만들지 못했습니다");
    }
  };
  const rotate = async (id: string) => {
    try {
      const x = await postJSON<{ key: string; message: string }>(
        `/api/v1/api-keys/${id}/rotate`,
        {},
      );
      setRevealed(x);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "키를 회전하지 못했습니다");
    }
  };
  // 폐기는 되돌릴 수 없고 그 키를 쓰던 연동이 즉시 끊긴다. 어떤 키를 지우는지
  // 보여 주고 한 번 더 확인받는다. 브라우저 기본 confirm은 화면 양식과 다르고,
  // 막혀 있는 환경에서는 눌러도 아무 일이 없는 것처럼 보인다.
  const [revoking, setRevoking] = useState<KeyItem | null>(null);
  const revoke = async () => {
    if (!revoking) return;
    try {
      await api(`/api/v1/api-keys/${revoking.id}`, { method: "DELETE" });
      setRevoking(null);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "키를 폐기하지 못했습니다");
      setRevoking(null);
    }
  };
  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 1100, mx: "auto" }}>
      <PageHeader
        eyebrow="PERSONAL ACCESS"
        title="내 API 키"
        description="REST API와 MCP에 사용하는 개인별 키를 생성하고 주기적으로 회전합니다."
        actions={
          <Button
            variant="contained"
            startIcon={<AddRounded />}
            onClick={() => setCreateOpen(true)}
          >
            키 만들기
          </Button>
        }
      />
      {error && (
        <Alert severity="error" onClose={() => setError("")} sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}
      <Alert severity="info" sx={{ mb: 2 }}>
        키 원문은 생성·회전 직후 한 번만 표시됩니다. 서버에는 복원할 수 없는
        HMAC 해시만 저장됩니다.
      </Alert>
      {loading ? (
        <TableSkeleton rows={4} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={<VpnKeyRounded />}
          title="아직 발급한 키가 없습니다."
          description="키를 만들면 REST API와 MCP 클라이언트에서 SeatOn에 접근할 수 있습니다."
          action={
            <Button
              variant="contained"
              startIcon={<AddRounded />}
              onClick={() => setCreateOpen(true)}
            >
              키 만들기
            </Button>
          }
        />
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>이름 / 식별자</TableCell>
                <TableCell>상태</TableCell>
                <TableCell>범위</TableCell>
                <TableCell>버전</TableCell>
                <TableCell>마지막 사용</TableCell>
                <TableCell>만료</TableCell>
                <TableCell align="right">관리</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {items.map((k) => (
                <TableRow
                  key={k.id}
                  sx={{ opacity: k.revokedAt && !k.graceUntil ? 0.5 : 1 }}
                >
                  <TableCell>
                    <Typography variant="body2" fontWeight={700}>
                      {k.name}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {k.prefix}••••••
                    </Typography>
                  </TableCell>
                  <TableCell>
                    {/* 흐리게만 표시하면 폐기된 키인지 알 수 없고 화면 낭독기에는
                        아무것도 전해지지 않는다. */}
                    <Chip
                      size="small"
                      label={keyStatus(k).label}
                      color={keyStatus(k).color}
                      variant={k.revokedAt ? "outlined" : "filled"}
                    />
                  </TableCell>
                  <TableCell>
                    {k.scopes.map((s) => (
                      <Chip key={s} size="small" label={s} sx={{ mr: 0.5 }} />
                    ))}
                  </TableCell>
                  {/* 회전 유예는 상태 칸이 알린다. 두 곳에서 말하면 유예가
                      끝난 키가 폐기됨과 유예 중으로 동시에 보인다. */}
                  <TableCell>v{k.version}</TableCell>
                  <TableCell>
                    {k.lastUsedAt
                      ? new Date(k.lastUsedAt).toLocaleString("ko-KR")
                      : "사용 전"}
                  </TableCell>
                  <TableCell>
                    {k.expiresAt
                      ? new Date(k.expiresAt).toLocaleDateString("ko-KR")
                      : "제한 없음"}
                  </TableCell>
                  <TableCell align="right">
                    <Tooltip title="새 키를 만들고 이 키는 유예기간 뒤 만료합니다">
                      <IconButton
                        aria-label="회전"
                        onClick={() => void rotate(k.id)}
                        disabled={Boolean(k.revokedAt)}
                      >
                        <AutorenewRounded />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="이 키를 즉시 무효로 만듭니다">
                      <IconButton
                        aria-label="폐기"
                        color="error"
                        onClick={() => setRevoking(k)}
                        disabled={Boolean(k.revokedAt)}
                      >
                        <DeleteOutlineRounded />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      <Typography variant="body2" color="text.secondary" mt={2}>
        MCP Endpoint: <code>{window.location.origin}/mcp</code> · Authorization:{" "}
        <code>Bearer seat_…</code>
      </Typography>
      {sso && (
        <Alert severity="info" sx={{ mt: 2 }} data-testid="mcp-sso-hint">
          <Typography variant="subtitle2">키 없이 SSO 로 연결하기</Typography>
          <Typography variant="body2">
            이 서버의 MCP 는 사내 SSO(Keycloak) 로그인도 받습니다. MCP
            클라이언트(Claude, Cursor 등)에 주소 <code>{sso.resource}</code>{" "}
            하나만 등록하면 클라이언트가 스스로 로그인 화면을 띄우고 토큰을 받아
            옵니다 — 키를 만들거나 붙여 넣을 필요가 없습니다. 지금 로그인한 이
            계정이 그대로 쓰이며, 권한도 화면과 같습니다. 자동화 스크립트처럼
            사람이 로그인할 수 없는 곳에서는 여전히 개인 키를 쓰세요.
          </Typography>
        </Alert>
      )}
      <Dialog open={createOpen} onClose={() => setCreateOpen(false)}>
        <DialogTitle>개인 API 키 만들기</DialogTitle>
        <DialogContent>
          <TextField
            label="키 이름"
            fullWidth
            value={name}
            onChange={(e) => setName(e.target.value)}
            sx={{ mt: 1 }}
          />
          <Typography variant="subtitle2" mt={2}>
            허용 범위
          </Typography>
          <FormGroup row>
            {["read", "write", "mcp"].map((scope) => (
              <FormControlLabel
                key={scope}
                control={
                  <Checkbox
                    checked={scopes.includes(scope)}
                    onChange={(e) =>
                      setScopes((v) =>
                        e.target.checked
                          ? [...v, scope]
                          : v.filter((x) => x !== scope),
                      )
                    }
                  />
                }
                label={scope}
              />
            ))}
          </FormGroup>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)}>취소</Button>
          <Button
            variant="contained"
            disabled={!name || !scopes.length}
            onClick={() => void create()}
          >
            생성
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog
        open={Boolean(revealed)}
        onClose={() => setRevealed(null)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>지금 키를 복사하세요</DialogTitle>
        <DialogContent>
          <Alert severity="warning" sx={{ mb: 2 }}>
            {revealed?.message}
          </Alert>
          <Paper
            variant="outlined"
            sx={{
              p: 2,
              fontFamily: "monospace",
              wordBreak: "break-all",
              bgcolor: "#F5F8F9",
            }}
          >
            {revealed?.key}
          </Paper>
        </DialogContent>
        <DialogActions>
          <Button
            startIcon={<ContentCopyRounded />}
            onClick={() =>
              void navigator.clipboard.writeText(revealed?.key || "")
            }
          >
            복사
          </Button>
          <Button variant="contained" onClick={() => setRevealed(null)}>
            보관 완료
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog
        open={Boolean(revoking)}
        onClose={() => setRevoking(null)}
        fullWidth
        maxWidth="xs"
      >
        <DialogTitle>API 키 폐기</DialogTitle>
        <DialogContent>
          <Typography variant="body2">
            <strong>{revoking?.name}</strong>({revoking?.prefix}…) 키를 즉시
            무효로 만듭니다. 이 키를 쓰던 연동은 바로 끊기며 되돌릴 수 없습니다.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRevoking(null)}>취소</Button>
          <Button
            color="error"
            variant="contained"
            onClick={() => void revoke()}
          >
            폐기
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
