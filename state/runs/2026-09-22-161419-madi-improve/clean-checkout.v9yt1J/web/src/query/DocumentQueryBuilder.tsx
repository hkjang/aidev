import { useMemo, useRef, useState } from "react";
import { Button, ErrorBox, Field, Modal } from "../ui";
import { queryLabels, querySourceLabels, type QuerySource } from "./types";
import "./query.css";
const fields: Record<QuerySource, string[]> = {
  documents: [
    "title",
    "status",
    "tags",
    "owner_name",
    "updated_at",
    "classification",
    "kind",
    "version",
  ],
  tasks: [
    "document_title",
    "text",
    "done",
    "status",
    "priority",
    "assignee_name",
    "due_date",
  ],
  relations: ["source_title", "target_title", "relation_type", "depth"],
};
export default function DocumentQueryBuilder({
  onInsert,
  onClose,
}: {
  onInsert: (markdown: string) => void;
  onClose: () => void;
}) {
  const opener = useRef(
    document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null,
  );
  const [source, setSource] = useState<QuerySource>("documents"),
    [columns, setColumns] = useState(["title", "status", "updated_at"]),
    [limit, setLimit] = useState(20),
    [property, setProperty] = useState(""),
    [propertyType, setPropertyType] = useState("string"),
    [parameter, setParameter] = useState(false),
    [direction, setDirection] = useState("outgoing"),
    [depth, setDepth] = useState(1),
    [error, setError] = useState<unknown>(null);
  const definition = useMemo(
    () => ({
      version: 1,
      source,
      columns: [
        ...columns.map((field) => ({ field, label: queryLabels[field] })),
        ...(source === "documents" && property
          ? [{ field: `property.${property}`, label: property }]
          : []),
      ],
      ...(source === "documents" && property
        ? { properties: { [property]: propertyType } }
        : {}),
      ...(parameter
        ? {
            parameters: {
              검색어: { type: "string", label: "포함할 문구", required: true },
            },
            filters: [
              {
                field:
                  source === "documents"
                    ? "title"
                    : source === "tasks"
                      ? "text"
                      : "target_title",
                op: "contains",
                value: { parameter: "검색어" },
              },
            ],
          }
        : {}),
      ...(source === "relations"
        ? { direction, depth }
        : {
            order_by: [
              {
                field: source === "documents" ? "updated_at" : "document_title",
                direction: "desc",
              },
            ],
          }),
      limit,
    }),
    [
      source,
      columns,
      property,
      propertyType,
      parameter,
      direction,
      depth,
      limit,
    ],
  );
  const json = JSON.stringify(definition, null, 2),
    markdown = `\n\n\`\`\`madi-query\n${json}\n\`\`\`\n\n`;
  const insert = () => {
    if (
      !columns.length ||
      !Number.isInteger(limit) ||
      limit < 1 ||
      limit > 100 ||
      (property && !/^[\p{L}\p{N}_][\p{L}\p{N}_-]{0,63}$/u.test(property))
    ) {
      setError(new Error("표시할 열과 행 수(1~100), 속성 이름을 확인하세요."));
      return;
    }
    onInsert(markdown);
    onClose();
  };
  return (
    <Modal
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
      title="선언형 조회 표 넣기"
      description="문서·할 일·관계를 안전한 JSON 정의로 조회합니다. 정의만 편집기에 넣으며, 문서 저장과 읽기 화면의 조회 실행은 별도입니다."
      wide
      onCloseAutoFocus={(event) => {
        event.preventDefault();
        requestAnimationFrame(
          () =>
            opener.current?.isConnected &&
            opener.current.focus({ preventScroll: true }),
        );
      }}
    >
      <div className="query-builder">
        <ErrorBox error={error} />
        <div className="query-builder-grid">
          <Field label="조회 대상">
            <select
              value={source}
              onChange={(e) => {
                const value = e.target.value as QuerySource;
                setSource(value);
                setColumns(fields[value].slice(0, 3));
                setProperty("");
                setError(null);
              }}
            >
              {Object.entries(querySourceLabels).map(([key, name]) => (
                <option key={key} value={key}>
                  {name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="최대 결과 행">
            <input
              type="number"
              min={1}
              max={100}
              value={limit}
              onChange={(e) => setLimit(Number(e.target.value))}
            />
          </Field>
        </div>
        <fieldset>
          <legend>표시할 열</legend>
          <div className="query-column-options">
            {fields[source].map((field) => (
              <label key={field}>
                <input
                  type="checkbox"
                  checked={columns.includes(field)}
                  onChange={(e) =>
                    setColumns((old) =>
                      e.target.checked
                        ? [...old, field]
                        : old.filter((v) => v !== field),
                    )
                  }
                />
                {queryLabels[field]}
              </label>
            ))}
          </div>
        </fieldset>
        <label className="query-check">
          <input
            type="checkbox"
            checked={parameter}
            onChange={(e) => setParameter(e.target.checked)}
          />
          실행할 때 포함할 문구를 입력받기
        </label>
        {source === "documents" && (
          <div className="query-builder-grid">
            <Field
              label="추가 Front Matter 속성 (선택)"
              hint="최상위의 문자·숫자·날짜·참/거짓 값만 읽습니다."
            >
              <input
                value={property}
                onChange={(e) => setProperty(e.target.value)}
                maxLength={64}
                placeholder="예: 점검일"
              />
            </Field>
            <Field label="속성 유형">
              <select
                value={propertyType}
                onChange={(e) => setPropertyType(e.target.value)}
              >
                <option value="string">문자</option>
                <option value="number">숫자</option>
                <option value="date">날짜</option>
                <option value="boolean">참/거짓</option>
              </select>
            </Field>
          </div>
        )}
        {source === "relations" && (
          <div className="query-builder-grid">
            <Field label="관계 방향">
              <select
                value={direction}
                onChange={(e) => setDirection(e.target.value)}
              >
                <option value="outgoing">현재 문서가 연결하는 대상</option>
                <option value="incoming">현재 문서로 들어오는 관계</option>
              </select>
            </Field>
            <Field label="최대 경로 깊이">
              <select
                value={depth}
                onChange={(e) => setDepth(Number(e.target.value))}
              >
                <option value={1}>1단계</option>
                <option value={2}>2단계</option>
                <option value={3}>3단계</option>
              </select>
            </Field>
          </div>
        )}
        <p className="muted">
          최대 2,000개 후보·8 MiB만 검사합니다. SQL·JavaScript·외부 API 호출은
          지원하지 않으며, 조회 결과는 Markdown 원문이나 공개 공유에 포함되지
          않습니다.
        </p>
        <details>
          <summary>생성할 JSON 정의 확인</summary>
          <pre>
            <code>{json}</code>
          </pre>
        </details>
        <div className="query-builder-actions">
          <Button onClick={onClose}>취소</Button>
          <Button variant="primary" onClick={insert}>
            조회 정의 넣기
          </Button>
        </div>
      </div>
    </Modal>
  );
}
