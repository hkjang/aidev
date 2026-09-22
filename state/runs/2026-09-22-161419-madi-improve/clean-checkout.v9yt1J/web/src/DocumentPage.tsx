import CollaborationSaveStatus from "./collaboration/SaveStatus";
import { TagChips } from "./review/TagChips";
import { DocumentStates } from "./review/DocumentStates";
import { DocumentModeControls } from "./review/DocumentModeControls";
import { selectedMarkdown, type MarkdownSelection } from "./review/selection";
import { applyMarkdownProposal } from "./review/documentMutation";
import { ApiError } from "./api";
import { offerDocumentUndo } from "./review/DocumentRecovery";
import {
  AccessChangePreview,
  useAccessChangeReview,
} from "./review/AccessChangePreview";
const SelectionAI = lazy(() => import("./review/SelectionAI"));
const DocumentSplit = lazy(() => import("./review/DocumentSplit"));
const DocumentPassport = lazy(() => import("./worksets/DocumentPassport"));
import SaveToWorkset from "./worksets/SaveToWorkset";
import { captureWorksetDocumentContext } from "./navigation/NavigationMemory";
import ContextTools from "./editor/ContextTools";
import PasteReview from "./editor/PasteReview";
const RequestDocumentAccess = lazy(() =>
  import("./review/AccessRequestsPage").then((m) => ({
    default: m.RequestDocumentAccess,
  })),
);
const CollaborationDiagnostics = lazy(
  () => import("./collaboration/CollaborationDiagnostics"),
);
import {
  DocumentInspector,
  InspectorToggle,
  useDocumentInspector,
} from "./review/DocumentInspector";
import { ChangeReview, RecoveryNotice } from "./review/ChangeReview";
import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import AttachmentList from "./AttachmentList";
import DocumentProtection from "./security/DocumentProtection";
import PublicShareManager from "./security/PublicShareManager";
import BlockOrganizer from "./editor/BlockOrganizer";
import DocumentPresentation from "./editor/DocumentPresentation";
import { reconcileSavedDocument } from "./editor/saveSnapshot";
import { resolveEditorMode } from "./editor/mode";
const DocumentRAGIndex = lazy(() => import("./DocumentRAGIndex"));
const DocumentHistory = lazy(() => import("./history/DocumentHistory"));
const DocumentPreview = lazy(() => import("./review/DocumentPreview"));
import { MoveDocumentModal } from "./navigation/DocumentActions";
import { copyText } from "./navigation/clipboard";
import ApprovalPanel from "./approval/ApprovalPanel";
import {
  Link,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { Markdown } from "@tiptap/markdown";
import TaskList from "@tiptap/extension-task-list";
import TaskItem from "@tiptap/extension-task-item";
import { TableCell, TableHeader, TableRow } from "@tiptap/extension-table";
import Image from "@tiptap/extension-image";
import TextAlign from "@tiptap/extension-text-align";
import {
  PortableTable,
  PortableParagraph,
  PortableHeading,
  unsupportedMarkdownReason,
} from "./editor/schema";
import { advancedNodes } from "./editor/nodes";
import {
  VisualCodeBlock,
  VisualInlineMath,
  VisualBlockMath,
  VisualSyncedEmbed,
  VisualBookmark,
} from "./editor/NodeViews";
import { MarkdownContent } from "./editor/MarkdownContent";
import { AdvancedToolbar } from "./editor/AdvancedToolbar";
import "./editor/style.css";
import Placeholder from "@tiptap/extension-placeholder";
import { UniqueID } from "@tiptap/extension-unique-id";
import Collaboration, { isChangeOrigin } from "@tiptap/extension-collaboration";
import CollaborationCaret from "@tiptap/extension-collaboration-caret";
import {
  MadiCollaborationProvider,
  type CollaborationSnapshot,
} from "./collaboration/provider";
import "./collaboration/style.css";
import type { Editor, JSONContent } from "@tiptap/core";
import {
  ArrowDownToLine,
  ShieldCheck,
  Network,
  ArrowUp,
  ArrowDown,
  GripVertical,
  Bold,
  Check,
  CheckCheck,
  CheckSquare,
  ChevronDown,
  Code,
  Code2,
  Columns2,
  Copy,
  Eye,
  FileText,
  Focus,
  Heading1,
  History,
  Italic,
  KeyRound,
  Link2,
  List,
  ListOrdered,
  MoreHorizontal,
  Paperclip,
  Quote,
  Redo,
  Save,
  Share2,
  Sparkles,
  Star,
  Table2,
  Trash2,
  Undo,
  X,
} from "lucide-react";
import {
  api,
  date,
  datetime,
  downloadText,
  type Doc,
  type DocSummary,
} from "./api";
import { useApp } from "./context";
import {
  Badge,
  Button,
  CopyButton,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  statusNames,
  DocIcon,
} from "./ui";
import DiscussionPanel from "./DiscussionPanel";
import DocumentQueryBuilder from "./query/DocumentQueryBuilder";

function splitFrontMatter(value: string) {
  const match = value.match(/^(---\r?\n[\s\S]*?\r?\n---\r?\n)/);
  return match
    ? { front: match[1], body: value.slice(match[1].length) }
    : { front: "", body: value };
}
export function MarkdownView({
  markdown,
  documents,
  metadata,
  documentId,
}: {
  markdown: string;
  documents: DocSummary[];
  metadata?: Doc["block_metadata"];
  documentId?: string;
}) {
  return (
    <MarkdownContent
      markdown={markdown}
      documents={documents}
      metadata={metadata}
      documentId={documentId}
    />
  );
}

type BlockEditorProps = {
  documentId: string;
  markdown: string;
  onChange: (v: string) => void;
  metadata: Doc["block_metadata"];
  onMetadata: (v: NonNullable<Doc["block_metadata"]>) => void;
  onProvider: (provider: MadiCollaborationProvider | null) => void;
  onSnapshot: (
    snapshot: CollaborationSnapshot,
    provider: MadiCollaborationProvider,
  ) => void;
};
function BlockEditor(props: BlockEditorProps) {
  const [provider, setProvider] = useState<MadiCollaborationProvider | null>(
    null,
  );
  const unsupported = unsupportedMarkdownReason(props.markdown);
  useEffect(() => {
    if (unsupported) return;
    const value = new MadiCollaborationProvider(props.documentId);
    setProvider(value);
    return () => {
      setTimeout(() => value.destroy(), 0);
    };
  }, [props.documentId]);
  if (unsupported && !provider)
    return (
      <div className="unsupported-markdown">
        <strong>원문 보존 모드</strong>
        <p>{unsupported}</p>
        <p>
          상단에서 Markdown 모드를 선택하면 원문을 그대로 편집할 수 있습니다.
        </p>
        <MarkdownContent markdown={props.markdown} />
      </div>
    );
  return provider ? (
    <NativeBlockEditor {...props} provider={provider} />
  ) : (
    <Loading />
  );
}
function NativeBlockEditor({
  documentId,
  markdown,
  onChange,
  metadata,
  onMetadata,
  onProvider,
  onSnapshot,
  provider,
}: BlockEditorProps & { provider: MadiCollaborationProvider }) {
  const { user } = useApp();
  const callbacks = useRef({ onProvider, onSnapshot, onChange, onMetadata });
  callbacks.current = { onProvider, onSnapshot, onChange, onMetadata };
  const [, setConnectionRevision] = useState(0);
  const original = useRef(markdown);
  const front = useRef(splitFrontMatter(markdown).front);
  const ready = useRef(false);
  const collect = (editor: Editor) => {
    const blocks: { id: string; type: string; text: string }[] = [];
    editor.state.doc.descendants((node) => {
      if (node.isBlock && node.attrs.id)
        blocks.push({
          id: node.attrs.id,
          type: node.type.name,
          text: Array.from(node.textContent).slice(0, 200).join(""),
        });
    });
    onMetadata({ blocks });
  };
  const [manage, setManage] = useState(false),
    [queryBuilder, setQueryBuilder] = useState(false),
    [revision, setRevision] = useState(0);
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        undoRedo: false,
        codeBlock: false,
        paragraph: false,
        heading: false,
      }),
      PortableParagraph,
      PortableHeading,
      VisualCodeBlock,
      Collaboration.configure({
        document: provider.document,
        field: "content",
      }),
      CollaborationCaret.configure({
        provider,
        user: { name: "사용자", color: "#0f766e" },
      }),
      Markdown,
      TaskList,
      TaskItem.configure({ nested: true }),
      PortableTable,
      TableCell,
      TableHeader,
      TableRow,
      Image,
      TextAlign.configure({ types: ["paragraph", "heading"] }),
      ...advancedNodes.filter(
        (extension) =>
          !["inlineMath", "blockMath", "syncedEmbed", "bookmark"].includes(
            extension.name,
          ),
      ),
      VisualInlineMath,
      VisualBlockMath,
      VisualSyncedEmbed,
      VisualBookmark,
      UniqueID.configure({
        types: "all",
        generateID: () => crypto.randomUUID(),
        filterTransaction: (transaction) => !isChangeOrigin(transaction),
      }),
      Placeholder.configure({
        placeholder: "생각을 기록하세요. / 를 입력해 블록을 추가할 수 있어요.",
      }),
    ],
    editable: false,
    editorProps: {
      attributes: { class: "tiptap-content", "aria-label": "블록 문서 편집기" },
    },
    onCreate: ({ editor }) => {
      ready.current = true;
    },
    onUpdate: ({ editor }) => {
      if (!ready.current) return;
      collect(editor);
      setRevision((v) => v + 1);
      const value = front.current + editor.getMarkdown();
      original.current = value;
      callbacks.current.onChange(value);
    },
  });
  useEffect(() => {
    if (editor)
      editor.view.dom.setAttribute(
        "spellcheck",
        String(user.preferences?.spell_check !== false),
      );
  }, [editor, user.preferences?.spell_check]);
  useEffect(() => {
    if (!editor) return;
    const seed = (snapshot: CollaborationSnapshot) => {
      const reason = unsupportedMarkdownReason(snapshot.markdown);
      if (reason) {
        provider.rejectUnsupported(reason);
        return;
      }
      const parts = splitFrontMatter(snapshot.markdown);
      front.current = parts.front;
      editor.commands.setContent(parts.body, { contentType: "markdown" });
      let index = 0;
      const transaction = editor.state.tr;
      editor.state.doc.descendants((node, pos) => {
        if (!node.isBlock) return;
        const stored = metadata?.blocks?.[index++];
        if (
          stored &&
          stored.type === node.type.name &&
          stored.text === Array.from(node.textContent).slice(0, 200).join("")
        )
          transaction.setNodeMarkup(pos, undefined, {
            ...node.attrs,
            id: stored.id,
          });
      });
      editor.view.dispatch(transaction);
      provider.seedFromCurrentDocument();
    };
    const snapshot = (value: CollaborationSnapshot) => {
      front.current = splitFrontMatter(value.markdown).front;
      callbacks.current.onSnapshot(value, provider);
    };
    const update = () => {
      editor.setEditable(
        provider.status === "connected" && !!provider.snapshot?.canWrite,
      );
      if (provider.snapshot)
        callbacks.current.onSnapshot(provider.snapshot, provider);
      setConnectionRevision((v) => v + 1);
      const anchor = decodeURIComponent(window.location.hash.slice(1));
      if (anchor.startsWith("^"))
        requestAnimationFrame(() =>
          editor.view.dom
            .querySelector(`[data-id="${CSS.escape(anchor.slice(1))}"]`)
            ?.scrollIntoView({ block: "center" }),
        );
    };
    const insert = (value: string) => {
      if (editor.isEditable)
        editor
          .chain()
          .focus()
          .insertContentAt(editor.state.doc.content.size, value, {
            contentType: "markdown",
          })
          .run();
    };
    provider
      .on("seed", seed)
      .on("snapshot", snapshot)
      .on("change", update)
      .on("presence", update)
      .on("insert", insert);
    callbacks.current.onProvider(provider);
    provider.connect();
    return () => {
      provider
        .off("seed", seed)
        .off("snapshot", snapshot)
        .off("change", update)
        .off("presence", update)
        .off("insert", insert);
      callbacks.current.onProvider(null);
    };
  }, [editor, provider]);
  const [slash, setSlash] = useState(false);
  if (!editor) return <Loading />;
  const insert = (value: string) => {
    editor
      .chain()
      .focus()
      .insertContent(value, { contentType: "markdown" })
      .run();
    setSlash(false);
  };
  return (
    <>
      <div
        className="collaboration-bar"
        data-state={provider.status}
        aria-live="polite"
      >
        <CollaborationSaveStatus provider={provider} />
        <div className="collaboration-peers">
          {[...provider.awareness.getStates()].map(
            ([client, state]) =>
              state.user && (
                <span
                  key={client}
                  style={{ background: state.user.color }}
                  title={state.user.name}
                >
                  {state.user.name?.slice(0, 1)}
                </span>
              ),
          )}
        </div>
      </div>
      {(provider.status === "error" || provider.status === "conflict") && (
        <div className="collaboration-recovery">
          <span>{provider.error}</span>
          <Button
            onClick={() =>
              downloadText(
                "madi-복구-초안.md",
                front.current + editor.getMarkdown(),
              )
            }
          >
            현재 초안 다운로드
          </Button>
          <Button
            onClick={() => {
              if (
                window.confirm(
                  "현재 초안을 다운로드했나요? 최신 서버 문서로 다시 연결합니다.",
                )
              )
                window.location.reload();
            }}
          >
            최신 문서 다시 연결
          </Button>
        </div>
      )}
      <div className="editor-toolbar">
        <button
          title="선언형 조회 표"
          aria-label="선언형 조회 표"
          onClick={() => setQueryBuilder(true)}
        >
          <Table2 size={18} />
        </button>
        <button
          title="굵게"
          aria-label="굵게"
          className={editor.isActive("bold") ? "active" : ""}
          onClick={() => editor.chain().focus().toggleBold().run()}
        >
          <Bold size={17} />
        </button>
        <button
          title="기울임"
          aria-label="기울임"
          onClick={() => editor.chain().focus().toggleItalic().run()}
        >
          <Italic size={17} />
        </button>
        <span />
        <select
          aria-label="블록 유형"
          value=""
          onChange={(e) => {
            if (e.target.value === "p")
              editor.chain().focus().setParagraph().run();
            else
              editor
                .chain()
                .focus()
                .toggleHeading({ level: Number(e.target.value) as 1 | 2 | 3 })
                .run();
          }}
        >
          <option value="" disabled>
            텍스트 / 제목
          </option>
          <option value="p">본문</option>
          <option value="1">제목 1</option>
          <option value="2">제목 2</option>
          <option value="3">제목 3</option>
        </select>
        <span />
        <button
          title="글머리 목록"
          aria-label="글머리 목록"
          onClick={() => editor.chain().focus().toggleBulletList().run()}
        >
          <List size={18} />
        </button>
        <button
          title="번호 목록"
          aria-label="번호 목록"
          onClick={() => editor.chain().focus().toggleOrderedList().run()}
        >
          <ListOrdered size={18} />
        </button>
        <button
          title="체크리스트"
          aria-label="체크리스트"
          onClick={() => editor.chain().focus().toggleTaskList().run()}
        >
          <CheckSquare size={17} />
        </button>
        <button
          title="인용"
          aria-label="인용"
          onClick={() => editor.chain().focus().toggleBlockquote().run()}
        >
          <Quote size={17} />
        </button>
        <button
          title="코드 블록"
          aria-label="코드 블록"
          onClick={() => editor.chain().focus().toggleCodeBlock().run()}
        >
          <Code2 size={18} />
        </button>
        <button
          title="표 삽입"
          aria-label="표 삽입"
          onClick={() =>
            editor
              .chain()
              .focus()
              .insertTable({ rows: 3, cols: 3, withHeaderRow: true })
              .run()
          }
        >
          <Table2 size={17} />
        </button>
        <span />
        <button
          title="실행 취소"
          aria-label="실행 취소"
          onClick={() => editor.chain().focus().undo().run()}
        >
          <Undo size={17} />
        </button>
        <button
          title="다시 실행"
          aria-label="다시 실행"
          onClick={() => editor.chain().focus().redo().run()}
        >
          <Redo size={17} />
        </button>
        <span />
        <button
          title="블록 정리"
          aria-label="블록 정리"
          onClick={() => setManage(true)}
        >
          <GripVertical size={18} />
        </button>
      </div>
      <div
        onKeyDown={(e) => {
          if (e.key === "/") setSlash(true);
          if (e.key === "Escape") setSlash(false);
        }}
      >
        <AdvancedToolbar editor={editor} documentId={documentId} />
        <ContextTools editor={editor} />
        <PasteReview editor={editor} />
        <EditorContent editor={editor} />
      </div>
      {slash && (
        <div className="slash-menu">
          <strong>블록 추가</strong>
          {[
            ["제목", "\n## 제목\n"],
            ["할 일", "\n- [ ] 할 일\n"],
            ["코드", "\n```text\n코드\n```\n"],
            ["표", "\n| 제목 | 내용 |\n| --- | --- |\n| 항목 | 값 |\n"],
            ["구분선", "\n---\n"],
            ["위키 링크", "[[문서 제목]]"],
            ["콜아웃", "> [!NOTE] 참고\n> 내용을 입력하세요.\n"],
            ["수식", "$$\nE=mc^2\n$$\n"],
            [
              "다이어그램",
              "```mermaid\nflowchart LR\n  A[아이디어] --> B[지식]\n```\n",
            ],
          ].map(([label, value]) => (
            <button
              key={label}
              onClick={() => {
                editor
                  .chain()
                  .focus()
                  .deleteRange({
                    from: Math.max(0, editor.state.selection.from - 1),
                    to: editor.state.selection.from,
                  })
                  .run();
                insert(value);
              }}
            >
              {label}
            </button>
          ))}
          <button onClick={() => setSlash(false)}>닫기</button>
        </div>
      )}
      {queryBuilder && (
        <DocumentQueryBuilder
          onClose={() => setQueryBuilder(false)}
          onInsert={(value) => {
            insert(value);
            setQueryBuilder(false);
          }}
        />
      )}
      <BlockOrganizer
        editor={editor}
        documentID={documentId}
        open={manage}
        onOpenChange={setManage}
      />
    </>
  );
}

export default function DocumentPage({ onAI }: { onAI: () => void }) {
  const inspector = useDocumentInspector();
  const { id } = useParams();
  const [modeParams, setModeParams] = useSearchParams();
  const sourceLine = /^\d{1,7}$/.test(modeParams.get("line") || "")
    ? Number(modeParams.get("line"))
    : 0;
  const sourceInput = useRef<HTMLTextAreaElement>(null),
    appliedSourceLocation = useRef("");
  const {
    documents,
    reload,
    notify,
    user,
    publicInfo,
    workspace,
    setWorkspace,
    createDocument,
  } = useApp();
  const navigate = useNavigate();
  const [doc, setDoc] = useState<Doc | null>(null),
    [markdown, setMarkdown] = useState(""),
    [title, setTitle] = useState(""),
    [tags, setTags] = useState(""),
    [mode, setMode] = useState(
      resolveEditorMode(
        modeParams.get("mode"),
        user.preferences?.editor_mode,
        sourceLine,
      ),
    ),
    [error, setError] = useState(""),
    [errorStatus, setErrorStatus] = useState(0),
    [saving, setSaving] = useState(false),
    [dirty, setDirty] = useState(false),
    [saved, setSaved] = useState(false),
    [history, setHistory] = useState(false),
    [backlinks, setBacklinks] = useState<Doc[]>([]),
    [share, setShare] = useState(false),
    [passport, setPassport] = useState(false),
    [visibility, setVisibility] = useState("workspace"),
    [parent, setParent] = useState(""),
    [aliases, setAliases] = useState(""),
    [shares, setShares] = useState<any[]>([]),
    [shareEmail, setShareEmail] = useState(""),
    [shareRole, setShareRole] = useState("viewer");
  const [pendingDraft, setPendingDraft] = useState<any>(null);
  const [selectionAI, setSelectionAI] = useState<{
    doc: Doc;
    selection: MarkdownSelection;
  } | null>(null);
  const [splitSelection, setSplitSelection] = useState<{
    doc: Doc;
    selection: MarkdownSelection;
  } | null>(null);
  const [previewDoc, setPreviewDoc] = useState<{
    id: string;
    version?: number;
  } | null>(null);
  const [reviewServer, setReviewServer] = useState<Doc | null>(null);
  const [reviewBusy, setReviewBusy] = useState(false);
  const [moveDoc, setMoveDoc] = useState<Doc | null>(null);
  const [ragIndexOpen, setRAGIndexOpen] = useState(false);
  const [diagnosticsOpen, setDiagnosticsOpen] = useState(false);
  const [favoriteBusy, setFavoriteBusy] = useState(false);
  const [actionBusy, setActionBusy] = useState(false);
  const [shareBusy, setShareBusy] = useState(false);
  const shareAccessReview = useAccessChangeReview(
    doc,
    { parent_id: parent, visibility },
    share && doc?.owner_id === user.id,
    JSON.stringify(
      shares.map((item) => [item.user_id, item.role, item.permission]),
    ),
  );
  const favoritePending = useRef(false),
    actionPending = useRef(false),
    sharePending = useRef(false),
    shareRevision = useRef(0),
    shareReadSequence = useRef(0),
    activeUser = useRef(user.id);
  activeUser.current = user.id;
  const changeShare = (open: boolean) => {
    shareRevision.current++;
    setShare(open);
  };
  const focus = modeParams.get("view") === "focus";
  const setFocus = (value: boolean) =>
    setModeParams((current) => {
      const next = new URLSearchParams(current);
      if (value) next.set("view", "focus");
      else next.delete("view");
      return next;
    });
  const [attachmentRevision, setAttachmentRevision] = useState(0);
  const collaboration = useRef<MadiCollaborationProvider | null>(null);
  const currentDocument = useRef(doc);
  currentDocument.current = doc;
  const fileInput = useRef<HTMLInputElement>(null),
    publicShareAnchor = useRef<HTMLDivElement>(null),
    activeRoute = useRef(id),
    loadSequence = useRef(0),
    blockMetadata = useRef<NonNullable<Doc["block_metadata"]>>({ blocks: [] }),
    latest = useRef({ markdown, title, tags }),
    savingRef = useRef(false);
  latest.current = { markdown, title, tags };
  const canWrite = doc?.can_write ?? false;
  useEffect(() => setHistory(false), [id]);
  useEffect(() => setSelectionAI(null), [id, user.id, workspace?.id]);
  useEffect(() => setSplitSelection(null), [id, user.id, workspace?.id]);
  useEffect(() => {
    const open = () => inspector.change("ai");
    window.addEventListener("madi:document-ai", open);
    return () => window.removeEventListener("madi:document-ai", open);
  }, [user.id]);
  useEffect(() => {
    if (modeParams.get("comment")) inspector.change("comments");
  }, [id, modeParams.get("comment")]);
  useEffect(() => {
    setMode(
      resolveEditorMode(
        modeParams.get("mode"),
        user.preferences?.editor_mode,
        sourceLine,
      ),
    );
  }, [id, modeParams.get("mode"), sourceLine]);
  useEffect(() => {
    if (!sourceLine || mode !== "source") appliedSourceLocation.current = "";
    if (
      mode !== "source" ||
      !doc ||
      doc.id !== id ||
      !sourceLine ||
      !sourceInput.current
    )
      return;
    const key = `${id}:${sourceLine}`;
    if (appliedSourceLocation.current === key) return;
    const lines = markdown.split("\n"),
      line = Math.min(lines.length, Math.max(1, sourceLine));
    const offset = lines
      .slice(0, line - 1)
      .reduce((total, value) => total + value.length + 1, 0);
    const input = sourceInput.current;
    appliedSourceLocation.current = key;
    input.focus({ preventScroll: true });
    input.setSelectionRange(offset, offset + lines[line - 1].length);
    input.scrollTop = Math.max(
      0,
      (line - 3) * (parseFloat(getComputedStyle(input).lineHeight) || 30),
    );
    input.scrollIntoView({ block: "center" });
  }, [id, doc?.id, mode, sourceLine, markdown]);
  activeRoute.current = id;
  // Route ID alone is insufficient for A → B → A. Invalidate every async UI
  // completion on reload, unmount, identity change, or navigation generation.
  const actionGuard = () => {
    const route = id,
      sequence = loadSequence.current,
      actor = user.id;
    return () =>
      window.location.pathname.replace(/\/+$/, "") ===
        `/app/documents/${route}` &&
      activeRoute.current === route &&
      currentDocument.current?.id === route &&
      sequence === loadSequence.current &&
      activeUser.current === actor;
  };
  const draftKey = (documentId: string) =>
    `madi.draft.${user.id}.${documentId}`;
  const load = useCallback(async () => {
    if (!id || activeRoute.current !== id) return;
    const sequence = ++loadSequence.current;
    savingRef.current = false;
    setSaving(false);
    setError("");
    setErrorStatus(0);
    try {
      const d = await api<Doc>("/documents/" + id);
      if (activeRoute.current !== id || sequence !== loadSequence.current)
        return;
      if (workspace?.id !== d.workspace_id) setWorkspace(d.workspace_id);
      setDoc(d);
      blockMetadata.current = d.block_metadata || { blocks: [] };
      setMarkdown(d.markdown);
      setTitle(d.title);
      setTags((d.tags || []).join(", "));
      setAliases((d.aliases || []).join(", "));
      setVisibility(d.visibility);
      setParent(d.parent_id || "");
      setDirty(false);
      setPendingDraft(null);
      try {
        const draft = JSON.parse(
          sessionStorage.getItem(draftKey(id)) || "null",
        );
        if (
          draft &&
          (draft.markdown !== d.markdown ||
            draft.title !== d.title ||
            draft.tags !== (d.tags || []).join(", "))
        )
          setPendingDraft(draft);
      } catch {}
      await Promise.allSettled([
        api<Doc[]>(`/documents/${id}/backlinks`).then((v) => {
          if (activeRoute.current === id && sequence === loadSequence.current)
            setBacklinks(v);
        }),
      ]);
    } catch (e) {
      if (activeRoute.current === id && sequence === loadSequence.current) {
        setError((e as Error).message);
        setErrorStatus(Number((e as { status?: number }).status) || 0);
      }
    }
  }, [id, user.id]);
  useEffect(() => {
    setDoc(null);
    setReviewServer(null);
    setPreviewDoc(null);
    setReviewBusy(false);
    setDiagnosticsOpen(false);
    setRAGIndexOpen(false);
    setMoveDoc(null);
    shareRevision.current++;
    setShare(false);
    setPassport(false);
    setShares([]);
    setShareEmail("");
    setShareRole("viewer");
    setFavoriteBusy(false);
    setActionBusy(false);
    setShareBusy(false);
    favoritePending.current =
      actionPending.current =
      sharePending.current =
        false;
    setDirty(false);
    setSaving(false);
    savingRef.current = false;
    setBacklinks([]);
    load();
    return () => {
      loadSequence.current++;
    };
  }, [load]);
  useEffect(() => {
    let active = true;
    const current = actionGuard();
    const sequence = ++shareReadSequence.current;
    setShares([]);
    if (share && id && canWrite)
      api<any[]>(`/documents/${id}/shares`)
        .then((value) => {
          if (active && current() && sequence === shareReadSequence.current)
            setShares(value);
        })
        .catch((e) => {
          if (active && current()) notify(e.message, "error");
        });
    return () => {
      active = false;
    };
  }, [share, id, canWrite]);
  const save = useCallback(
    async (silent = false) => {
      if (
        !doc ||
        !doc.can_write ||
        doc.id !== activeRoute.current ||
        savingRef.current
      )
        return false;
      const sequence = loadSequence.current;
      savingRef.current = true;
      setSaving(true);
      const snapshot = { ...latest.current };
      try {
        const live = collaboration.current;
        if (live) {
          await live.waitForSaved();
          if (
            activeRoute.current !== doc.id ||
            sequence !== loadSequence.current
          )
            return true;
          // Metadata remains editable while the body awaits a CRDT ACK. Read
          // it again before deciding that the whole document is saved. Adopt
          // acknowledged remote metadata only where the local field did not
          // change from this save's baseline; never replay its stale value.
          const confirmedTitle = live.snapshot?.title ?? doc.title;
          const originalTags = (doc.tags || []).join(", ");
          const confirmedTags = (live.snapshot?.tags ?? doc.tags ?? []).join(
            ", ",
          );
          snapshot.title =
            latest.current.title === doc.title
              ? confirmedTitle
              : latest.current.title;
          snapshot.tags =
            latest.current.tags === originalTags
              ? confirmedTags
              : latest.current.tags;
          if (
            snapshot.title === confirmedTitle &&
            snapshot.tags === confirmedTags
          ) {
            setDirty(live.hasUnsavedChanges);
            setSaved(!live.hasUnsavedChanges);
            if (!silent) notify("공동 편집 변경 내용을 저장했습니다.");
            return true;
          }
        }
        const d = await api<
          Doc & {
            protection?: {
              changed: boolean;
              mode: string;
              findings?: unknown[];
            };
          }
        >(`/documents/${doc.id}`, "PUT", {
          title: snapshot.title,
          ...(!live
            ? {
                markdown: snapshot.markdown,
                block_metadata: blockMetadata.current,
              }
            : {}),
          tags: snapshot.tags
            .split(",")
            .map((t) => t.trim())
            .filter(Boolean),
          version: live?.snapshot?.version || doc.version,
        });
        try {
          const draft = JSON.parse(
            sessionStorage.getItem(draftKey(doc.id)) || "null",
          );
          if (
            draft &&
            draft.markdown === snapshot.markdown &&
            draft.title === snapshot.title &&
            draft.tags === snapshot.tags
          )
            sessionStorage.removeItem(draftKey(doc.id));
        } catch {}
        if (activeRoute.current !== doc.id || sequence !== loadSequence.current)
          return true;
        // A newer CRDT acknowledgement can arrive while this REST save is in
        // flight. Never regress the server version or replace newer typing.
        const existing = currentDocument.current;
        const canonical =
          existing?.id === d.id && existing.version > d.version ? existing : d;
        const reconciled = reconcileSavedDocument(
          snapshot,
          latest.current,
          canonical,
          !!live?.hasUnsavedChanges,
        );
        latest.current = reconciled.values;
        currentDocument.current = canonical;
        setDoc(canonical);
        setTitle(reconciled.values.title);
        setTags(reconciled.values.tags);
        setMarkdown(reconciled.values.markdown);
        if (reconciled.applyMetadata)
          blockMetadata.current = canonical.block_metadata || { blocks: [] };
        setDirty(reconciled.dirty);
        setSaved(true);
        setError("");
        setErrorStatus(0);
        if (d.protection?.changed)
          notify(
            reconciled.dirty
              ? "보안 정책에 따라 저장본의 민감정보를 마스킹했습니다. 저장 중 추가한 내용은 초안으로 유지합니다."
              : "보안 정책에 따라 민감정보를 마스킹한 문서를 저장했습니다.",
          );
        else if (d.protection?.mode === "warn" && d.protection.findings?.length)
          notify(
            "민감정보가 탐지되었습니다. 문서의 공개 범위와 내용을 확인하세요.",
          );
        else if (!silent) notify("문서를 저장했습니다.");
        // Canonical PUT/CRDT confirmation completes this save. A sidebar list
        // refresh must not keep it pending or turn it into a failed save.
        const refreshCurrent = actionGuard();
        void reload().catch(() => {
          if (
            refreshCurrent() &&
            currentDocument.current?.version === canonical.version &&
            !savingRef.current
          )
            notify(
              "문서 목록을 새로 읽지 못했습니다. 목록을 새로고침해 주세요.",
              "error",
            );
        });
        return true;
      } catch (e) {
        if (
          activeRoute.current === doc.id &&
          sequence === loadSequence.current
        ) {
          setError((e as Error).message);
          setErrorStatus(Number((e as { status?: number }).status) || 0);
        }
        return false;
      } finally {
        if (
          activeRoute.current === doc.id &&
          sequence === loadSequence.current
        ) {
          savingRef.current = false;
          setSaving(false);
        }
      }
    },
    [doc, notify, reload],
  );
  useEffect(() => {
    if (!dirty || error) return;
    const t = setTimeout(() => save(true), 1800);
    return () => clearTimeout(t);
  }, [dirty, markdown, title, tags, save, error]);
  useEffect(() => {
    if (!dirty || !doc || doc.id !== id) return;
    try {
      sessionStorage.setItem(
        draftKey(doc.id),
        JSON.stringify({
          markdown,
          title,
          tags,
          version: doc.version,
          block_metadata: blockMetadata.current,
          saved_at: new Date().toISOString(),
        }),
      );
    } catch {
      setError(
        "브라우저 임시 저장 공간이 부족합니다. 문서를 저장한 후 이동하세요.",
      );
    }
  }, [dirty, markdown, title, tags, id, doc?.version]);
  useEffect(() => {
    const h = (e: BeforeUnloadEvent) => {
      if (dirty) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", h);
    return () => window.removeEventListener("beforeunload", h);
  }, [dirty]);
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "s") {
        e.preventDefault();
        save();
      }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [save]);
  const changeMarkdown = (v: string) => {
    if (mode === "source") blockMetadata.current = { blocks: [] };
    setMarkdown(v);
    setDirty(
      collaboration.current
        ? collaboration.current.hasUnsavedChanges ||
            latest.current.title !== currentDocument.current?.title ||
            latest.current.tags !==
              (currentDocument.current?.tags || []).join(", ")
        : true,
    );
    setSaved(false);
  };
  const collaborationSnapshot = (
    snapshot: CollaborationSnapshot,
    live: MadiCollaborationProvider,
  ) => {
    const previous = currentDocument.current;
    if (
      !previous ||
      previous.id !== live.documentId ||
      activeRoute.current !== live.documentId ||
      previous.version > snapshot.version
    )
      return;
    if (latest.current.title === previous.title) setTitle(snapshot.title);
    if (latest.current.tags === (previous.tags || []).join(", "))
      setTags(snapshot.tags.join(", "));
    setDoc((value) =>
      value?.id === live.documentId
        ? {
            ...value,
            version: snapshot.version,
            markdown: snapshot.markdown,
            can_write: snapshot.canWrite,
            title: snapshot.title,
            tags: snapshot.tags,
            status: snapshot.documentStatus,
          }
        : value,
    );
    if (!live.hasUnsavedChanges) setMarkdown(snapshot.markdown);
    const metadataDirty =
      latest.current.title !== previous.title ||
      latest.current.tags !== (previous.tags || []).join(", ");
    setDirty(live.hasUnsavedChanges || metadataDirty);
    setSaved(!live.hasUnsavedChanges);
  };
  const switchMode = async (value: string) => {
    const target = resolveEditorMode(value, undefined);
    if (target === mode) return;
    const current = actionGuard();
    try {
      if (collaboration.current) await collaboration.current.waitForSaved();
      if (!current()) return;
      if (dirty && !(await save(true))) return;
      if (!current()) return;
      // Source mode is an explicit optimistic REST editing session. Native CRDT
      // binding is unmounted before its first source edit; any remote session is
      // safely reset by the server's epoch barrier when source Markdown saves.
      // A mode switch is not a document reload. An in-flight GET here used to
      // replace typing made immediately after clicking an already active mode.
      // REST save and CRDT acknowledgements already reconcile the canonical
      // snapshot. Preserve newer input if it arrived while save was in flight.
      if (target === "edit") {
        const canonical = currentDocument.current;
        if (
          canonical &&
          (latest.current.markdown !== canonical.markdown ||
            latest.current.title !== canonical.title ||
            latest.current.tags !== (canonical.tags || []).join(", "))
        ) {
          notify(
            "저장 중 새로 입력한 내용이 남아 있습니다. 저장이 끝난 뒤 블록 편집기로 전환하세요.",
          );
          return;
        }
      }
      setMode(target);
      setModeParams((current) => {
        const next = new URLSearchParams(current);
        next.set("mode", value);
        next.delete("line");
        return next;
      });
    } catch (e) {
      if (current()) setError((e as Error).message);
    }
  };
  useEffect(() => {
    const handler = (event: Event) => {
      const action = (event as CustomEvent).detail?.action;
      if (
        !doc ||
        doc.id !== id ||
        ![
          "save",
          "focus",
          "duplicate",
          "move",
          "export",
          "print",
          "copy",
          "present",
        ].includes(action)
      )
        return;
      const current = actionGuard();
      // BrowserRouter can update the address before the previous document's
      // passive listener is removed. Never run that listener for the new URL.
      if (!current()) return;
      void (async () => {
        if (action === "focus") {
          setFocus(!focus);
          return;
        }
        if (action === "copy") {
          await copyText(`${window.location.origin}/app/documents/${doc.id}`);
          if (current()) notify("문서 링크를 복사했습니다.");
          return;
        }
        if (action === "save") {
          if (!canWrite) throw new Error("읽기 전용 문서입니다.");
          await save();
          return;
        }
        if (collaboration.current) await collaboration.current.waitForSaved();
        if (!current()) return;
        if (dirty && !(await save(true))) return;
        if (!current()) return;
        const fresh = await api<Doc>("/documents/" + doc.id);
        if (!current()) return;
        if (action === "export")
          downloadText(fresh.title + ".md", fresh.markdown);
        else if (action === "duplicate") {
          const copy = await createDocument(
            fresh.title + " 사본",
            fresh.markdown,
            { visibility: "private", tags: fresh.tags, block_metadata: {} },
          );
          if (copy && current()) navigate("/app/documents/" + copy.id);
        } else if (action === "move") {
          if (!fresh.can_write)
            throw new Error("문서를 이동할 권한이 없습니다.");
          setMoveDoc(fresh);
        } else if (action === "present" || action === "print") {
          setMode("preview");
          setModeParams((current) => {
            const next = new URLSearchParams(current);
            next.set("mode", "preview");
            if (action === "present") {
              next.set("view", "presentation");
              next.set("slide", "0");
            } else {
              next.delete("view");
              next.delete("slide");
            }
            return next;
          });
          if (action === "print") {
            await new Promise((resolve) =>
              requestAnimationFrame(() => requestAnimationFrame(resolve)),
            );
            if (current()) window.print();
          }
        }
      })().catch((e) => {
        if (current()) notify(e.message, "error");
      });
    };
    window.addEventListener("madi-document-command", handler);
    return () => window.removeEventListener("madi-document-command", handler);
  }, [doc, dirty, save, focus, createDocument, navigate, canWrite]);
  const upload = async (file: File) => {
    if (!doc || !canWrite) return;
    const current = actionGuard();
    const data = new FormData();
    data.append("file", file);
    try {
      const a = await api(`/attachments?document_id=${doc.id}`, "POST", data);
      if (!current()) return;
      const insertion = `\n\n${file.type.startsWith("image/") ? "!" : ""}[${a.name.replace(/[\[\]]/g, "")}](${a.url})\n`;
      setAttachmentRevision((n) => n + 1);
      if (collaboration.current)
        collaboration.current.insertMarkdown(insertion);
      else changeMarkdown(latest.current.markdown + insertion);
      notify("파일을 첨부했습니다.");
    } catch (e) {
      if (current()) notify((e as Error).message, "error");
    }
  };
  if (!doc || doc.id !== id)
    return error ? (
      <div>
        <RecoveryNotice
          error={error}
          status={errorStatus}
          onRetry={() => void load()}
          onReauthenticate={() => navigate("/login")}
        />
        {id && [403, 404].includes(errorStatus) && (
          <Suspense fallback={null}>
            <RequestDocumentAccess documentID={id} />
          </Suspense>
        )}
      </div>
    ) : (
      <Loading />
    );
  const headings = [...markdown.matchAll(/^(#{1,3})\s+(.+)$/gm)].map(
    (m, i) => ({ level: m[1].length, text: m[2], id: i }),
  );
  return (
    <div className={`document-page ${focus ? "focus-mode" : ""}`}>
      {splitSelection && (
        <Suspense fallback={<Loading />}>
          <DocumentSplit
            doc={splitSelection.doc}
            selection={splitSelection.selection}
            stale={
              doc.id !== splitSelection.doc.id ||
              doc.version !== splitSelection.doc.version ||
              dirty ||
              markdown !== splitSelection.doc.markdown ||
              !canWrite
            }
            onClose={() => setSplitSelection(null)}
            onCommit={async (ticket, requestID) => {
              const current = actionGuard(),
                snapshot = currentDocument.current;
              if (
                !snapshot ||
                !current() ||
                savingRef.current ||
                collaboration.current?.hasUnsavedChanges ||
                snapshot.version !== splitSelection.doc.version ||
                latest.current.markdown !== splitSelection.doc.markdown ||
                latest.current.title !== snapshot.title ||
                latest.current.tags !== (snapshot.tags || []).join(", ")
              )
                throw new ApiError(
                  "문서에 새로운 변경이 있습니다. 현재 원문을 저장하고 다시 선택하세요.",
                  409,
                );
              // Detach the previous CRDT epoch before the atomic REST mutation.
              setMode("source");
              setModeParams((value) => {
                const next = new URLSearchParams(value);
                next.set("mode", "source");
                next.delete("line");
                return next;
              });
              await new Promise<void>((resolve) =>
                requestAnimationFrame(() => resolve()),
              );
              if (!current()) return;
              const result = await api<
                import("./review/DocumentSplit").SplitResult
              >(`/documents/${snapshot.id}/split`, "POST", {
                ticket,
                client_request_id: requestID,
                consent: true,
              });
              if (!current()) return;
              const reconciled = reconcileSavedDocument(
                {
                  markdown: snapshot.markdown,
                  title: snapshot.title,
                  tags: (snapshot.tags || []).join(", "),
                },
                latest.current,
                result.source,
                false,
              );
              latest.current = reconciled.values;
              currentDocument.current = result.source;
              setDoc(result.source);
              setTitle(reconciled.values.title);
              setTags(reconciled.values.tags);
              setMarkdown(reconciled.values.markdown);
              if (reconciled.applyMetadata)
                blockMetadata.current = result.source.block_metadata || {
                  blocks: [],
                };
              setDirty(reconciled.dirty);
              setSaved(true);
              setError("");
              setErrorStatus(0);
              notify(
                result.replayed
                  ? "이미 완료한 분리 결과를 다시 확인했습니다."
                  : "선택 블록을 하위 초안으로 분리했습니다. 원본에 남은 참조로 새 문서를 열 수 있습니다.",
              );
              setSplitSelection(null);
              await reload();
              return result;
            }}
          />
        </Suspense>
      )}
      {selectionAI && (
        <Suspense fallback={<Loading />}>
          <SelectionAI
            doc={selectionAI.doc}
            selection={selectionAI.selection}
            stale={
              doc.id !== selectionAI.doc.id ||
              doc.version !== selectionAI.doc.version ||
              dirty ||
              markdown !== selectionAI.doc.markdown ||
              doc.can_write !== selectionAI.doc.can_write
            }
            onClose={() => setSelectionAI(null)}
            onApply={async (nextMarkdown, expectedVersion) => {
              const current = actionGuard(),
                snapshot = currentDocument.current;
              if (
                !snapshot ||
                !current() ||
                savingRef.current ||
                collaboration.current?.hasUnsavedChanges ||
                latest.current.markdown !== selectionAI.doc.markdown ||
                latest.current.title !== snapshot.title ||
                latest.current.tags !== (snapshot.tags || []).join(", ")
              )
                throw new ApiError(
                  "저장되지 않은 변경이 있습니다. 현재 원문을 저장하고 다시 선택하세요.",
                  409,
                );
              if (snapshot.version !== expectedVersion)
                throw new ApiError(
                  "문서 버전이 변경되었습니다. 현재 원문에서 다시 선택하세요.",
                  409,
                );
              setMode("source");
              setModeParams((value) => {
                const next = new URLSearchParams(value);
                next.set("mode", "source");
                next.delete("line");
                return next;
              });
              await new Promise<void>((resolve) =>
                requestAnimationFrame(() => resolve()),
              );
              if (!current()) return;
              const result = await applyMarkdownProposal(
                snapshot,
                nextMarkdown,
                expectedVersion,
              );
              if (!current()) return;
              const before = {
                markdown: snapshot.markdown,
                title: snapshot.title,
                tags: (snapshot.tags || []).join(", "),
              };
              const reconciled = reconcileSavedDocument(
                before,
                latest.current,
                result,
                false,
              );
              latest.current = reconciled.values;
              currentDocument.current = result;
              setDoc(result);
              setTitle(reconciled.values.title);
              setTags(reconciled.values.tags);
              setMarkdown(reconciled.values.markdown);
              if (reconciled.applyMetadata)
                blockMetadata.current = result.block_metadata || { blocks: [] };
              setDirty(reconciled.dirty);
              setSaved(true);
              setError("");
              setErrorStatus(0);
              notify(
                result.protection?.changed
                  ? "AI 변경을 적용하고 정보보호 정책에 따라 정본을 마스킹했습니다."
                  : "비교한 AI 변경을 원문에 적용했습니다.",
              );
              await reload();
            }}
          />
        </Suspense>
      )}
      {previewDoc && (
        <Suspense fallback={<Loading />}>
          <DocumentPreview
            documentId={previewDoc.id}
            expectedVersion={previewDoc.version}
            onClose={() => setPreviewDoc(null)}
          />
        </Suspense>
      )}
      {diagnosticsOpen && (
        <Suspense fallback={<Loading />}>
          <CollaborationDiagnostics
            documentId={doc.id}
            open={diagnosticsOpen}
            onOpenChange={setDiagnosticsOpen}
          />
        </Suspense>
      )}
      {ragIndexOpen && (
        <Suspense fallback={<Loading />}>
          <DocumentRAGIndex
            documentID={doc.id}
            onClose={() => setRAGIndexOpen(false)}
          />
        </Suspense>
      )}
      <MoveDocumentModal doc={moveDoc} close={() => setMoveDoc(null)} />
      {modeParams.get("view") === "presentation" && (
        <DocumentPresentation
          title={title}
          source={markdown}
          documents={documents}
          slide={Number(modeParams.get("slide") || 0)}
          onSlide={(value) =>
            setModeParams(
              (current) => {
                const next = new URLSearchParams(current);
                next.set("slide", String(value));
                return next;
              },
              { replace: true },
            )
          }
          close={() =>
            setModeParams((current) => {
              const next = new URLSearchParams(current);
              next.delete("view");
              next.delete("slide");
              return next;
            })
          }
        />
      )}
      <div className="document-actionbar">
        <InspectorToggle
          open={inspector.open}
          onClick={() => inspector.toggle(!inspector.open)}
        />
        <Link to="/app/documents">
          <FileText size={16} /> 문서
        </Link>
        <div>
          <span className="save-state">
            {saving ? (
              "저장 중…"
            ) : dirty ? (
              "저장되지 않은 변경"
            ) : (
              <>
                <CheckCheck size={15} /> 저장됨
              </>
            )}
          </span>
          <button
            className="icon-button"
            title="즐겨찾기"
            aria-label="즐겨찾기"
            aria-pressed={!!doc.is_favorite}
            disabled={favoriteBusy}
            onClick={async () => {
              if (favoritePending.current) return;
              const current = actionGuard();
              favoritePending.current = true;
              setFavoriteBusy(true);
              try {
                const result = await api<Doc>(
                  `/documents/${id}/favorite`,
                  "POST",
                );
                if (!current()) return;
                setDoc((value) =>
                  value?.id === result.id
                    ? { ...value, is_favorite: result.is_favorite }
                    : value,
                );
                await reload();
              } catch (e) {
                if (current()) notify((e as Error).message, "error");
              } finally {
                if (current()) {
                  favoritePending.current = false;
                  setFavoriteBusy(false);
                }
              }
            }}
          >
            <Star size={19} fill={doc.is_favorite ? "currentColor" : "none"} />
          </button>
          <Button
            onClick={() => {
              setVisibility(doc.visibility);
              setParent(doc.parent_id || "");
              setAliases((doc.aliases || []).join(", "));
              changeShare(true);
            }}
          >
            <Share2 size={16} /> 공유
          </Button>
          <Button
            variant="primary"
            disabled={saving || !dirty || !canWrite}
            onClick={() => save()}
          >
            <Save size={16} /> 저장
          </Button>
        </div>
      </div>
      <DocumentStates
        doc={doc}
        dirty={dirty}
        saving={saving}
        owner={doc.owner_id === user.id}
        onSharing={() => {
          setVisibility(doc.visibility);
          setParent(doc.parent_id || "");
          setAliases((doc.aliases || []).join(", "));
          changeShare(true);
        }}
        onPublishing={() => navigate(`/app/documents/${doc.id}/knowledge`)}
        onExternal={() => {
          inspector.change("properties");
          requestAnimationFrame(() =>
            publicShareAnchor.current?.querySelector("button")?.click(),
          );
        }}
      />
      <RecoveryNotice
        error={error}
        status={errorStatus}
        dirty={dirty}
        busy={reviewBusy || saving}
        onCopy={
          dirty
            ? () => downloadText(`${title || "madi"}-미확정-초안.md`, markdown)
            : undefined
        }
        onRetry={() => (dirty ? void save() : void load())}
        onReauthenticate={() => navigate("/login")}
        onReview={
          dirty
            ? async () => {
                const current = actionGuard();
                setReviewBusy(true);
                try {
                  const latestServer = await api<Doc>(`/documents/${doc.id}`);
                  if (current()) setReviewServer(latestServer);
                } catch (e) {
                  if (current()) {
                    setError((e as Error).message);
                    setErrorStatus(
                      Number((e as { status?: number }).status) || 0,
                    );
                  }
                } finally {
                  if (current()) setReviewBusy(false);
                }
              }
            : undefined
        }
      />
      {pendingDraft && canWrite && (
        <div className="notice">
          <History size={20} />
          <div>
            <strong>이전에 저장하지 못한 내용이 있습니다.</strong>
            <p>
              이 탭에 임시 보관된 내용을 복구할 수 있습니다. 서버 문서를 확인한
              후 선택하세요.
            </p>
            <div className="button-row">
              <Button
                onClick={() => {
                  if (collaboration.current) {
                    downloadText("madi-임시-복구.md", pendingDraft.markdown);
                    notify(
                      "공동 편집 중에는 복구 초안을 다운로드하여 확인한 뒤 필요한 내용을 붙여넣으세요.",
                    );
                    return;
                  }
                  setMarkdown(pendingDraft.markdown);
                  setTitle(pendingDraft.title);
                  setTags(pendingDraft.tags);
                  blockMetadata.current = pendingDraft.block_metadata || {
                    blocks: [],
                  };
                  setPendingDraft(null);
                  setDirty(true);
                  notify("임시 내용을 복구했습니다. 자동 저장으로 이어집니다.");
                }}
              >
                임시 내용 복구
              </Button>
              <Button
                onClick={() => {
                  sessionStorage.removeItem(draftKey(doc.id));
                  setPendingDraft(null);
                }}
              >
                서버 내용 유지
              </Button>
            </div>
          </div>
        </div>
      )}
      {error && (
        <Button
          onClick={() => {
            if (
              !dirty ||
              window.confirm(
                "현재 초안은 이 탭에 임시 보관됩니다. 서버 문서를 다시 불러올까요?",
              )
            )
              void load();
          }}
        >
          서버 문서 다시 불러오기
        </Button>
      )}
      {!canWrite && (
        <div className="notice subtle">
          <Eye size={19} />
          <span>
            읽기 전용 문서입니다. 편집 권한이 필요한 경우 문서 소유자에게
            요청하세요.
          </span>
        </div>
      )}
      <div className="document-layout">
        <article
          className="document-main"
          data-document-id={doc.id}
          data-document-version={doc.version}
        >
          <div className="doc-cover">
            <span>
              <DocIcon icon={doc.icon} size={38} />
            </span>
            <div className="cover-pattern" />
          </div>
          <div className="document-body">
            <div className="doc-overline">
              <Badge tone={doc.status === "published" ? "green" : ""}>
                {statusNames[doc.status] || doc.status}
              </Badge>
              <span>{date(doc.updated_at)} 업데이트</span>
              <span>버전 {doc.version}</span>
            </div>
            <input
              className="document-title"
              aria-label="문서 제목"
              value={title}
              readOnly={!canWrite}
              onChange={(e) => {
                setTitle(e.target.value);
                setDirty(true);
              }}
              placeholder="제목 없는 문서"
            />
            <div className="document-metadata">
              <TagChips
                key={doc.id}
                value={tags}
                readOnly={!canWrite}
                suggestions={documents.flatMap((d) => d.tags || [])}
                onChange={(value) => {
                  setTags(value);
                  setDirty(true);
                }}
              />
            </div>
            <DocumentModeControls
              mode={mode}
              canWrite={canWrite}
              onMode={(value) => void switchMode(value)}
              onFocus={() => setFocus(!focus)}
              onPresent={() =>
                window.dispatchEvent(
                  new CustomEvent("madi-document-command", {
                    detail: { action: "present" },
                  }),
                )
              }
              onPrint={() =>
                window.dispatchEvent(
                  new CustomEvent("madi-document-command", {
                    detail: { action: "print" },
                  }),
                )
              }
              onAttach={() => fileInput.current?.click()}
              onAI={() => inspector.change("ai")}
              onSplit={() => {
                if (
                  dirty ||
                  savingRef.current ||
                  collaboration.current?.hasUnsavedChanges ||
                  markdown !== doc.markdown
                ) {
                  notify(
                    "현재 변경을 저장한 뒤 완전한 블록을 선택하세요.",
                    "error",
                  );
                  return;
                }
                const selected = selectedMarkdown(
                  markdown,
                  sourceInput.current,
                );
                if (!selected?.text.trim()) {
                  notify(
                    "분리할 완전한 문단·목록·표·코드 블록을 선택하세요. 서식 위치가 모호하면 Markdown 원문에서 선택할 수 있습니다.",
                    "error",
                  );
                  return;
                }
                setSplitSelection({ doc: { ...doc }, selection: selected });
              }}
              onSelectionAI={() => {
                if (
                  dirty ||
                  savingRef.current ||
                  collaboration.current?.hasUnsavedChanges ||
                  markdown !== doc.markdown
                ) {
                  notify(
                    "현재 변경을 먼저 저장한 뒤 원문을 선택하세요.",
                    "error",
                  );
                  return;
                }
                const selected = selectedMarkdown(
                  markdown,
                  sourceInput.current,
                );
                if (!selected || !selected.text.trim()) {
                  notify(
                    "문서에서 AI로 다룰 문장을 선택하세요. 서식 때문에 위치가 모호하면 Markdown 원문에서 정확한 범위를 선택할 수 있습니다.",
                    "error",
                  );
                  return;
                }
                if (new TextEncoder().encode(selected.text).length > 32768) {
                  notify(
                    "선택 원문은 32KiB 이하여야 합니다. 범위를 줄여주세요.",
                    "error",
                  );
                  return;
                }
                setSelectionAI({ doc: { ...doc }, selection: selected });
              }}
            />
            {mode === "source" && (
              <p className="muted small-text">
                Markdown 원문 편집 모드 · 보기 도구에서 선택한 고급 편집
                방식입니다.
              </p>
            )}
            <input
              ref={fileInput}
              type="file"
              className="sr-only"
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) upload(file);
                e.target.value = "";
              }}
            />
            <div
              className="editor-area"
              onDragOver={(e) => {
                if (e.dataTransfer.types.includes("Files")) e.preventDefault();
              }}
              onDrop={(e) => {
                if (e.dataTransfer.files.length) {
                  e.preventDefault();
                  for (const file of e.dataTransfer.files) upload(file);
                }
              }}
            >
              <DocumentProtection
                key={doc.id}
                documentID={doc.id}
                version={doc.version}
              />
              {mode === "edit" && canWrite ? (
                <BlockEditor
                  key={id}
                  documentId={id!}
                  markdown={markdown}
                  onChange={changeMarkdown}
                  metadata={blockMetadata.current}
                  onMetadata={(v) => {
                    if (activeRoute.current === id) blockMetadata.current = v;
                  }}
                  onProvider={(value) => {
                    const previous = collaboration.current;
                    if (!value && previous?.hasUnsavedChanges) {
                      try {
                        sessionStorage.setItem(
                          draftKey(previous.documentId),
                          JSON.stringify({
                            ...latest.current,
                            block_metadata: blockMetadata.current,
                            saved_at: new Date().toISOString(),
                          }),
                        );
                      } catch {}
                    }
                    collaboration.current = value;
                  }}
                  onSnapshot={collaborationSnapshot}
                />
              ) : mode === "source" ? (
                <>
                  {canWrite && (
                    <div className="notice subtle">
                      Markdown 원문은 버전 충돌 검사를 사용하는 별도 편집
                      모드입니다. 저장하면 다른 공동 편집 세션은 초안을 보관하고
                      새 기준으로 다시 연결합니다.
                    </div>
                  )}
                  {sourceLine > 0 && (
                    <p className="notice subtle">
                      검색 결과 원문 · {sourceLine}번째 줄을 선택했습니다.
                      문서가 변경되었다면 위치가 달라질 수 있습니다.
                    </p>
                  )}
                  <textarea
                    ref={sourceInput}
                    className="markdown-source"
                    aria-label="Markdown 원문 편집"
                    value={markdown}
                    readOnly={!canWrite}
                    onChange={(e) => changeMarkdown(e.target.value)}
                    spellCheck={user.preferences?.spell_check !== false}
                  />
                </>
              ) : (
                <MarkdownView
                  markdown={markdown}
                  documents={documents}
                  metadata={blockMetadata.current}
                  documentId={
                    !dirty && markdown === doc.markdown ? doc.id : undefined
                  }
                />
              )}
            </div>
            <div className="doc-bottom-line">
              <span>{markdown.length.toLocaleString()}자 · Markdown 원본</span>
              <span>Ctrl + S 저장</span>
            </div>
            {publicInfo.approval_enabled && (
              <ApprovalPanel
                key={doc.id}
                resourceID={doc.id}
                revision={doc.version}
                canSubmit={canWrite}
                beforeSubmit={async () => {
                  const current = actionGuard();
                  const ready = !dirty || !!(await save(true));
                  return ready && current();
                }}
                onChanged={load}
              />
            )}
            <AttachmentList
              key={`attachments-${doc.id}`}
              documentID={doc.id}
              revision={attachmentRevision}
            />
          </div>
        </article>
        <DocumentInspector
          documentID={doc.id}
          version={doc.version}
          {...inspector}
          onChange={inspector.change}
          onOpen={inspector.toggle}
          backlinks={
            <>
              <section>
                <h3>이 문서에서</h3>
                {headings.length ? (
                  headings.map((h) => (
                    <button
                      key={h.id}
                      className={`outline-item depth-${h.level}`}
                      onClick={() => {
                        const root = document.querySelector(".editor-area");
                        const found = [
                          ...(root?.querySelectorAll("h1,h2,h3") || []),
                        ].find((el) => el.textContent === h.text);
                        found?.scrollIntoView({
                          behavior: "smooth",
                          block: "center",
                        });
                      }}
                    >
                      {h.text}
                    </button>
                  ))
                ) : (
                  <p className="muted small-text">
                    제목을 추가하면 목차가 표시됩니다.
                  </p>
                )}
              </section>
              <section>
                <h3>
                  <Link2 size={16} /> 연결된 문서{" "}
                  <Badge>{backlinks.length}</Badge>
                </h3>
                {backlinks.length ? (
                  backlinks.map((d) => (
                    <div className="backlink-with-preview" key={d.id}>
                      <Link className="backlink" to={`/app/documents/${d.id}`}>
                        <FileText size={16} />
                        {d.title}
                      </Link>
                      <button
                        className="icon-button"
                        aria-label={`${d.title} 미리보기`}
                        onClick={() =>
                          setPreviewDoc({ id: d.id, version: d.version })
                        }
                      >
                        <Eye size={17} />
                      </button>
                    </div>
                  ))
                ) : (
                  <p className="muted small-text">
                    이 문서를 참조하는 백링크가 아직 없어요.
                  </p>
                )}
                <Link className="text-button" to={`/app/graph?focus=${doc.id}`}>
                  전체 그래프 보기 →
                </Link>
              </section>

              <Link
                className="aside-action"
                to={`/app/tasks?document_id=${doc.id}`}
              >
                <CheckSquare size={17} />
                문서의 할 일 확인
              </Link>
              <Link className="aside-action" to="/app/evidence">
                <ShieldCheck size={17} />
                근거 보관함
              </Link>
            </>
          }
          properties={
            <>
              <div className="workset-actions">
                <Button onClick={() => setPassport(true)}>문서 여권</Button>
                <SaveToWorkset
                  disabled={
                    dirty ||
                    saving ||
                    !!collaboration.current?.hasUnsavedChanges
                  }
                  item={() => ({
                    kind: "document",
                    resource_id: doc.id,
                    context: captureWorksetDocumentContext(doc.version),
                  })}
                />
              </div>
              <section>
                <h3>현재 문서 속성</h3>
                {passport && (
                  <Suspense fallback={<Loading />}>
                    <DocumentPassport
                      documentID={doc.id}
                      onClose={() => setPassport(false)}
                      allowCleanup={
                        !dirty &&
                        !saving &&
                        !collaboration.current?.hasUnsavedChanges
                      }
                      onUpdated={() => void load()}
                    />
                  </Suspense>
                )}
                <dl className="inspector-properties">
                  <dt>공개 범위</dt>
                  <dd>
                    {doc.visibility === "private"
                      ? "나만 보기"
                      : doc.visibility === "selected"
                        ? "선택한 사용자"
                        : "워크스페이스"}
                  </dd>
                  <dt>버전</dt>
                  <dd>{doc.version}</dd>
                  <dt>태그</dt>
                  <dd>{doc.tags?.join(", ") || "없음"}</dd>
                </dl>
                <Button disabled={!canWrite} onClick={() => changeShare(true)}>
                  <Share2 size={17} />
                  공유·속성 변경
                </Button>
              </section>
              <section>
                <h3>문서 도구</h3>
                <button
                  className="aside-action"
                  onClick={() => setDiagnosticsOpen(true)}
                >
                  <Network size={17} />
                  공동 편집 진단
                </button>
                <Link
                  className="aside-action"
                  to="/app/access-requests?view=received"
                >
                  <KeyRound size={17} />내 문서 접근 요청
                </Link>
                <Link
                  className="aside-action"
                  to={`/app/knowledge-proposals?document_id=${doc.id}`}
                >
                  <Network size={17} />
                  문서 변경 제안
                </Link>
                <Link
                  className="aside-action"
                  to={`/app/knowledge-time?document_id=${doc.id}`}
                >
                  <History size={17} /> 시점 기준 지식
                </Link>
                {doc.owner_id === user.id && (
                  <div ref={publicShareAnchor}>
                    <PublicShareManager key={doc.id} documentID={doc.id} />
                  </div>
                )}
                <button
                  className="aside-action"
                  onClick={() => setRAGIndexOpen(true)}
                >
                  <Sparkles size={17} /> AI 검색 색인
                </button>
                <Link
                  className="aside-action"
                  to={`/app/documents/${id}/knowledge`}
                >
                  <FileText size={17} /> 문서 운영 속성
                </Link>
                {publicInfo.runbook_enabled && (
                  <Link
                    className="aside-action"
                    to={`/app/documents/${doc.id}/runbook`}
                  >
                    <Code2 size={17} /> 실행 런북
                  </Link>
                )}
                <button
                  className="aside-action"
                  onClick={async () => {
                    const current = actionGuard();
                    try {
                      if (dirty && !(await save(true))) return;
                      if (current()) setHistory(true);
                    } catch (e) {
                      if (current()) notify((e as Error).message, "error");
                    }
                  }}
                >
                  <History size={17} /> 변경 이력
                </button>
                <button
                  className="aside-action"
                  onClick={() => downloadText(`${title}.md`, markdown)}
                >
                  <ArrowDownToLine size={17} /> Markdown 다운로드
                </button>
                <button className="aside-action" onClick={() => window.print()}>
                  <FileText size={17} /> 인쇄 / PDF 저장
                </button>
                <button
                  className="aside-action"
                  disabled={!canWrite || actionBusy || dirty || saving}
                  onClick={async () => {
                    if (actionPending.current) return;
                    const current = actionGuard();
                    actionPending.current = true;
                    setActionBusy(true);
                    try {
                      const d = await api<Doc>("/documents", "POST", {
                        workspace_id: doc.workspace_id,
                        title: title + " (복사)",
                        markdown,
                        tags: doc.tags,
                        visibility: doc.visibility,
                      });
                      if (!current()) return;
                      await reload();
                      if (!current()) return;
                      navigate(`/app/documents/${d.id}`);
                      notify("문서를 복제했습니다.");
                    } catch (e) {
                      if (current()) notify((e as Error).message, "error");
                    } finally {
                      if (current()) {
                        actionPending.current = false;
                        setActionBusy(false);
                      }
                    }
                  }}
                >
                  <Copy size={17} /> 문서 복제
                </button>
                <button
                  className="aside-action danger-text"
                  disabled={!canWrite || actionBusy || dirty || saving}
                  onClick={async () => {
                    if (actionPending.current) return;
                    if (!window.confirm("이 문서를 휴지통으로 이동할까요?"))
                      return;
                    const current = actionGuard();
                    actionPending.current = true;
                    setActionBusy(true);
                    try {
                      const removed = await api<{ version: number }>(
                        "/documents/" + id,
                        "DELETE",
                        { expected_version: doc.version },
                      );
                      if (!current()) return;
                      await reload();
                      if (!current()) return;
                      navigate("/app/documents");
                      offerDocumentUndo({
                        id: doc.id,
                        version: removed.version,
                        title: doc.title,
                        actor: user.id,
                        workspace: doc.workspace_id,
                      });
                    } catch (e) {
                      if (current()) notify((e as Error).message, "error");
                    } finally {
                      if (current()) {
                        actionPending.current = false;
                        setActionBusy(false);
                      }
                    }
                  }}
                >
                  <Trash2 size={17} /> 휴지통으로 이동
                </button>
              </section>
            </>
          }
          comments={<DiscussionPanel key={doc.id} document={doc} />}
          onHistory={() => setHistory(true)}
        />
      </div>
      {history && (
        <Suspense fallback={<Loading />}>
          <DocumentHistory
            key={doc.id}
            documentID={doc.id}
            currentVersion={doc.version}
            canWrite={canWrite && !dirty}
            documents={documents}
            onClose={() => setHistory(false)}
            onRestored={async () => {
              if (activeRoute.current !== doc.id) return;
              // load() advances the generation synchronously; capture after it.
              const pending = load();
              const current = actionGuard();
              await pending;
              if (!current()) return;
              await reload();
              if (current())
                notify("이전 버전을 새 문서 버전으로 복원했습니다.");
            }}
          />
        </Suspense>
      )}
      <Modal
        open={!!reviewServer}
        onOpenChange={(v) => {
          if (!v) setReviewServer(null);
        }}
        title="서버 내용과 내 변경 비교"
        description="확인한 서버 버전을 기준으로만 다시 저장합니다. 그사이 다른 변경이 생기면 다시 충돌로 안내합니다."
        wide
      >
        {reviewServer && (
          <ChangeReview
            title="문서 변경 확인"
            description={`서버 버전 ${reviewServer.version} · 내 편집 기준 버전 ${doc.version}`}
            changes={[
              { label: "제목", before: reviewServer.title, after: title },
              {
                label: "태그",
                before: reviewServer.tags?.join(", "),
                after: tags,
              },
              {
                label: "Markdown 원문",
                before: reviewServer.markdown.slice(0, 100000),
                after: markdown.slice(0, 100000),
              },
            ]}
            warnings={[
              "서버의 최신 내용 대신 내 변경을 저장합니다. 필요한 서버 변경을 먼저 내 초안에 반영하세요.",
              ...(markdown.length > 100000 ||
              reviewServer.markdown.length > 100000
                ? [
                    "미리보기는 각 원문의 앞 100,000자만 표시합니다. 저장 대상 원문은 잘리지 않습니다.",
                  ]
                : []),
              ...(!reviewServer.can_write
                ? ["현재 쓰기 권한이 없어 적용할 수 없습니다."]
                : []),
              ...(collaboration.current
                ? [
                    "공동 편집 중에는 이 화면에서 덮어쓰지 않습니다. 초안을 보관하고 서버 문서를 다시 여세요.",
                  ]
                : []),
            ]}
            disabled={!reviewServer.can_write || !!collaboration.current}
            confirmLabel="확인한 버전을 기준으로 저장"
            onCancel={() => setReviewServer(null)}
            onConfirm={() => {
              if (!reviewServer.can_write || collaboration.current) return;
              currentDocument.current = reviewServer;
              setDoc(reviewServer);
              setDirty(true);
              setError("");
              setErrorStatus(0);
              setReviewServer(null);
            }}
          />
        )}
      </Modal>
      <Modal
        open={share}
        onOpenChange={changeShare}
        title="문서 공유 및 속성"
        description="문서의 공개 범위와 탐색 위치를 설정하세요."
      >
        <Field label="문서 링크">
          <div className="input-with-button">
            <input
              readOnly
              value={window.location.origin + `/app/documents/${id}`}
            />
            <CopyButton
              value={window.location.origin + `/app/documents/${id}`}
            />
          </div>
        </Field>
        <Field label="공개 범위">
          <select
            value={visibility}
            disabled={!canWrite || doc.owner_id !== user.id || shareBusy}
            onChange={(e) => setVisibility(e.target.value)}
          >
            <option value="private">개인 문서 · 나만 보기</option>
            <option value="workspace">워크스페이스 멤버</option>
            <option value="selected">선택한 사용자</option>
          </select>
        </Field>
        <Field label="상위 문서">
          <select
            disabled={!canWrite || doc.owner_id !== user.id || shareBusy}
            value={parent}
            onChange={(e) => setParent(e.target.value)}
          >
            <option value="">최상위 문서</option>
            {documents
              .filter((d) => d.id !== id)
              .map((d) => (
                <option key={d.id} value={d.id}>
                  {d.title}
                </option>
              ))}
          </select>
        </Field>
        <Field
          label="문서 별칭"
          hint="이전 문서명 등을 쉼표로 구분해 입력하세요."
        >
          <input
            readOnly={!canWrite}
            disabled={shareBusy}
            value={aliases}
            onChange={(e) => setAliases(e.target.value)}
          />
        </Field>
        {visibility === "selected" && canWrite && doc.owner_id === user.id && (
          <div className="sharing-users">
            <Field label="공유할 사용자 이메일">
              <input
                type="email"
                disabled={shareBusy}
                value={shareEmail}
                onChange={(e) => setShareEmail(e.target.value)}
              />
            </Field>
            <Field label="공유 권한">
              <select
                disabled={shareBusy}
                value={shareRole}
                onChange={(e) => setShareRole(e.target.value)}
              >
                <option value="viewer">읽기</option>
                <option value="editor">편집</option>
              </select>
            </Field>
            <Button
              disabled={!shareEmail || shareBusy}
              onClick={async () => {
                if (sharePending.current) return;
                const current = actionGuard(),
                  revision = shareRevision.current;
                const currentModal = () =>
                  current() && revision === shareRevision.current;
                sharePending.current = true;
                setShareBusy(true);
                try {
                  await api(`/documents/${id}/shares`, "POST", {
                    email: shareEmail,
                    role: shareRole,
                  });
                  if (!currentModal()) return;
                  const sequence = ++shareReadSequence.current;
                  const result = await api(`/documents/${id}/shares`);
                  if (!currentModal() || sequence !== shareReadSequence.current)
                    return;
                  setShares(result);
                  setShareEmail("");
                  notify("사용자 공유를 추가했습니다.");
                } catch (e) {
                  if (currentModal()) notify((e as Error).message, "error");
                } finally {
                  if (current()) {
                    sharePending.current = false;
                    setShareBusy(false);
                  }
                }
              }}
            >
              공유 사용자 추가
            </Button>
            {shares.map((s) => (
              <p key={s.user_id || s.id}>
                {s.name || s.email} · {s.role === "editor" ? "편집" : "읽기"}
              </p>
            ))}
          </div>
        )}
        <AccessChangePreview review={shareAccessReview} busy={shareBusy} />
        <div className="modal-actions">
          <Button onClick={() => changeShare(false)}>취소</Button>
          <Button
            variant="primary"
            disabled={!canWrite || shareBusy || !shareAccessReview.ready}
            onClick={async () => {
              if (sharePending.current || !shareAccessReview.ready) return;
              const current = actionGuard(),
                revision = shareRevision.current;
              const currentModal = () =>
                current() && revision === shareRevision.current;
              const previous = currentDocument.current;
              if (!previous || previous.id !== id) return;
              // This request changes properties only: use the last canonical
              // body as the baseline so unsaved editor text is never discarded.
              const snapshot = {
                title: previous.title,
                markdown: previous.markdown,
                tags: (previous.tags || []).join(", "),
              };
              sharePending.current = true;
              setShareBusy(true);
              try {
                const d = await api<
                  Doc & { protection?: { changed?: boolean } }
                >(`/documents/${id}`, "PUT", {
                  version: previous.version,
                  ...(doc.owner_id === user.id
                    ? {
                        visibility,
                        parent_id: parent || null,
                        access_preview_ticket: shareAccessReview.ticket,
                      }
                    : {}),
                  aliases: aliases
                    .split(",")
                    .map((a) => a.trim())
                    .filter(Boolean),
                });
                if (!current()) return;
                const existing = currentDocument.current;
                const canonical =
                  existing?.id === d.id && existing.version > d.version
                    ? existing
                    : d;
                const reconciled = reconcileSavedDocument(
                  snapshot,
                  latest.current,
                  canonical,
                  !!collaboration.current?.hasUnsavedChanges,
                );
                currentDocument.current = canonical;
                latest.current = reconciled.values;
                setDoc(canonical);
                setTitle(reconciled.values.title);
                setMarkdown(reconciled.values.markdown);
                setTags(reconciled.values.tags);
                if (reconciled.applyMetadata)
                  blockMetadata.current = canonical.block_metadata || {
                    blocks: [],
                  };
                setDirty(reconciled.dirty);
                await reload();
                if (!currentModal()) return;
                changeShare(false);
                notify(
                  d.protection?.changed
                    ? "문서 속성을 저장하고 보안 정책에 따라 저장본을 마스킹했습니다."
                    : "문서 속성을 저장했습니다.",
                );
              } catch (e) {
                if (currentModal()) notify((e as Error).message, "error");
              } finally {
                if (current()) {
                  sharePending.current = false;
                  setShareBusy(false);
                }
              }
            }}
          >
            설정 저장
          </Button>
        </div>
      </Modal>
    </div>
  );
}
