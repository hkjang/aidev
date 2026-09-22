// Runs in a CSP-restricted Worker: no document, cookies, DOM or direct network.
(async () => {
  const context = await madi.ready();
  let query = "",
    status = "필요한 지식과 다음 행동을 연결해 보세요.",
    documents = [];
  function render() {
    madi.ui.render({
      type: "stack",
      children: [
        { type: "badge", text: "MADI PLUGIN SDK · OFFLINE READY" },
        { type: "heading", text: "팀의 지식을 한곳에서" },
        { type: "text", text: status },
        {
          type: "input",
          label: "문서 검색어",
          placeholder: "찾고 싶은 주제를 입력하세요",
          value: query,
          onChange: (value) => {
            query = value;
          },
        },
        {
          type: "row",
          children: [
            { type: "button", text: "문서 찾기", onClick: search },
            { type: "button", text: "오늘의 기록 만들기", onClick: createNote },
          ],
        },
        ...documents
          .slice(0, 8)
          .map((doc) => ({
            type: "button",
            text: doc.title,
            onClick: () => madi.navigate("/app/documents/" + doc.id),
          })),
        { type: "divider" },
        {
          type: "text",
          text: "이 확장은 현재 사용자가 접근 가능한 문서만 조회합니다. 파일과 AI는 별도로 승인된 권한을 사용합니다.",
        },
      ],
    });
  }
  async function search() {
    try {
      documents = await madi.documents.list({ q: query });
      status = `접근 가능한 문서 ${documents.length}개를 찾았습니다.`;
      render();
      return { count: documents.length };
    } catch (error) {
      status = error.message;
      render();
    }
  }
  async function createNote() {
    try {
      const today = new Date().toLocaleDateString("ko-KR");
      const doc = await madi.documents.create({
        title: today + " · 오늘의 기록",
        markdown: "# 오늘의 기록\n\n## 기억할 것\n\n## 다음 행동\n\n- [ ] ",
        visibility: "private",
      });
      status = "개인 문서에 오늘의 기록을 만들었습니다.";
      render();
      return { id: doc.id, title: doc.title };
    } catch (error) {
      status = error.message;
      render();
    }
  }
  madi.registerCommand("recent", search);
  madi.registerCommand("create-note", createNote);
  madi.registerSidebar("knowledge-sidebar", () => {
    render();
  });
  madi.registerMenu("my-preferences", async () => {
    const last = await madi.storage.get("last-opened");
    await madi.storage.set("last-opened", new Date().toISOString());
    return { last_opened: last, current_user: context.user.name };
  });
  madi.registerBlock("summary-card", (data) => {
    madi.ui.render({
      type: "stack",
      children: [
        { type: "badge", text: "KNOWLEDGE CARD" },
        { type: "heading", text: String(data?.title || "연결되는 지식") },
        {
          type: "text",
          text: String(
            data?.summary ||
              "Markdown 원본과 플러그인 데이터를 함께 보존합니다.",
          ),
        },
        { type: "button", text: "관련 문서 찾아보기", onClick: search },
      ],
    });
  });
  madi.registerImporter("markdown", async (file) => {
    const doc = await madi.documents.create({
      title: file.name.replace(/\.[^.]+$/, ""),
      markdown: file.content,
      visibility: "private",
    });
    return { created_document: doc.title, id: doc.id };
  });
  madi.registerExporter("document-list", async () => {
    const docs = await madi.documents.list({});
    return {
      name: "madi-knowledge.md",
      content:
        "# 나의 지식 목록\n\n" +
        docs.map((doc) => "- [[" + doc.id + "|" + doc.title + "]]").join("\n"),
    };
  });
  madi.registerAIProvider("workspace-ai");
  render();
})();
